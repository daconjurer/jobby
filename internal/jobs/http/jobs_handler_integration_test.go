//go:build integration

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	"github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/daconjurer/jobby/internal/testutil"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Integration tests require MongoDB (for example: task mongo-up).
// Run: task test-integration
//
// Required: MONGODB_URI (host: replicaSet=rs0 and directConnection=true — see .env.example).
// Optional (defaults match cmd/jobs-server and .env.example): MONGODB_DATABASE, MONGODB_COLLECTION_METADATA,
// MONGODB_COLLECTION_LOGS.
//
// Database lifecycle matches internal/jobs/mongodb/mongo_integration_test.go: collections are cleared
// before each subtest and again on teardown so runs stay idempotent.

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func integrationMongoEnv(tb testing.TB) mongodb.MongoConfig {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test (-short)")
	}
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		tb.Fatalf("MONGODB_URI is not set (required for integration tests; see .env.example)")
	}
	db := os.Getenv("MONGODB_DATABASE")
	if db == "" {
		db = "jobby"
	}
	metaColl := os.Getenv("MONGODB_COLLECTION_METADATA")
	if metaColl == "" {
		metaColl = "job_metadata"
	}
	logsColl := os.Getenv("MONGODB_COLLECTION_LOGS")
	if logsColl == "" {
		logsColl = "job_logs"
	}
	return mongodb.MongoConfig{
		URI:                uri,
		Database:           db,
		CollectionMetadata: metaColl,
		CollectionLogs:     logsColl,
		Timeout:            30 * time.Second,
		MaxPoolSize:        50,
		MinPoolSize:        0,
	}
}

func clearJobCollections(ctx context.Context, db *mongo.Database, cfg mongodb.MongoConfig) error {
	meta := db.Collection(cfg.CollectionMetadata)
	logs := db.Collection(cfg.CollectionLogs)
	if _, err := meta.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("clear %q: %w", cfg.CollectionMetadata, err)
	}
	if _, err := logs.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("clear %q: %w", cfg.CollectionLogs, err)
	}
	return nil
}

func setupIntegrationCollections(ctx context.Context, db *mongo.Database, cfg mongodb.MongoConfig) error {
	return clearJobCollections(ctx, db, cfg)
}

func teardownIntegrationCollections(ctx context.Context, db *mongo.Database, cfg mongodb.MongoConfig) {
	_ = clearJobCollections(ctx, db, cfg)
}

// prepareIntegrationMongoPersistence opens reader/writer against MongoDB, clears both collections before the test,
// and registers teardown cleanup so writes do not leak across runs (same pattern as metadata integration tests).
func prepareIntegrationMongoPersistence(t *testing.T) (*mongodb.MongoJobsReader, *mongodb.MongoJobsWriter) {
	t.Helper()
	cfg := integrationMongoEnv(t)
	ctx := context.Background()

	reader, writer, client, err := mongodb.OpenMongoJobs(ctx, cfg)
	if err != nil {
		t.Fatalf("OpenMongoJobs: %v", err)
	}
	db := client.Database(cfg.Database)
	if err := setupIntegrationCollections(ctx, db, cfg); err != nil {
		t.Fatalf("setupIntegrationCollections: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})
	t.Cleanup(func() {
		teardownIntegrationCollections(context.Background(), db, cfg)
	})

	return reader, writer
}

func startJobsIntegrationServer(tb testing.TB, reader *mongodb.MongoJobsReader, writer *mongodb.MongoJobsWriter) (baseURL string, httpClient *http.Client) {
	tb.Helper()
	resolver, err := pulsar.NewFileTopicResolver(testutil.JobTopicsConfigPath(tb))
	if err != nil {
		tb.Fatalf("NewFileTopicResolver: %v", err)
	}
	svc := service.NewMetadataService(reader, writer)
	enqueue := service.NewEnqueueService(svc, resolver)
	h := NewJobsHandler(svc, enqueue)

	r := gin.New()
	apiRoutes := r.Group("/api")
	jobs := apiRoutes.Group("/jobs")
	{
		jobs.GET("", h.ListJobs)
		jobs.POST("", h.EnqueueJob)
		jobs.GET("/stats", h.GetJobStats)
		jobs.GET("/:id", h.GetJob)
		jobs.POST("/:id/fail", h.FailJob)
		jobs.POST("/:id/cancel", h.CancelJob)
		jobs.POST("/:id/retry", h.RetryJob)
		jobs.GET("/:id/logs", h.GetJobLogs)
	}

	srv := httptest.NewServer(r)
	tb.Cleanup(func() {
		srv.Close()
		if tr, ok := srv.Client().Transport.(closeIdleTransport); ok {
			tr.CloseIdleConnections()
		}
	})

	return srv.URL, srv.Client()
}

type closeIdleTransport interface {
	CloseIdleConnections()
}

func apiJobs(baseURL string, parts ...string) string {
	path := strings.Join(parts, "")
	return strings.TrimSuffix(baseURL, "/") + "/api/jobs" + path
}

func mustDecodeJSON(tb testing.TB, body io.Reader, v any) {
	tb.Helper()
	if err := json.NewDecoder(body).Decode(v); err != nil {
		tb.Fatalf("decode json: %v", err)
	}
}

func readBody(tb testing.TB, resp *http.Response) []byte {
	tb.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		tb.Fatalf("read body: %v", err)
	}
	_ = resp.Body.Close()
	return b
}

func markJobRunningForIntegrationTest(t *testing.T, writer *mongodb.MongoJobsWriter, jobID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	running := metadata.JobStatusRunning
	patch := metadata.UpdateJob{Status: &running, StartedAt: &now}
	if err := writer.Update(ctx, jobID, patch); err != nil {
		t.Fatalf("mark job running: %v", err)
	}
}

func getJobViaHTTP(t *testing.T, client *http.Client, baseURL, jobID string) metadata.JobMetadataModel {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, apiJobs(baseURL, "/", jobID), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get job: %d %s", resp.StatusCode, body)
	}
	var job metadata.JobMetadataModel
	mustDecodeJSON(t, bytes.NewReader(body), &job)
	return job
}

func ginErrFromBody(tb testing.TB, body []byte) string {
	tb.Helper()
	var wrap struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		tb.Fatalf("unmarshal error json: %v body=%s", err, body)
	}
	return wrap.Error
}

func TestIntegration_JobsHandler_HTTP(t *testing.T) {
	t.Run("Enqueue_and_Get_roundTrip", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		payload := map[string]any{"k": "v"}
		meta := map[string]any{"region": "eu"}
		prio := 7
		body := map[string]any{
			"name":     "account-lifecycle",
			"payload":  payload,
			"priority": prio,
			"tags":     []string{"http", "integration"},
			"metadata": meta,
		}
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		respBody := readBody(t, resp)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("enqueue: status=%d body=%s", resp.StatusCode, respBody)
		}

		var created metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(respBody), &created)
		if created.JobID == "" || created.Name != "account-lifecycle" {
			t.Fatalf("unexpected created job: %+v", created)
		}
		if created.Topic != "persistent://public/default/accounts/jobs" {
			t.Fatalf("topic=%q", created.Topic)
		}
		if created.Status != metadata.JobStatusPendingDispatch {
			t.Fatalf("status=%s want pending_dispatch", created.Status)
		}

		req, err := http.NewRequest(http.MethodGet, apiJobs(baseURL, "/", created.JobID), nil)
		if err != nil {
			t.Fatal(err)
		}
		getResp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		getBody := readBody(t, getResp)
		if getResp.StatusCode != http.StatusOK {
			t.Fatalf("get: status=%d body=%s", getResp.StatusCode, getBody)
		}
		var got metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(getBody), &got)
		if got.JobID != created.JobID || got.Name != created.Name {
			t.Fatalf("get mismatch: %+v vs %+v", got, created)
		}
	})

	t.Run("Enqueue_validation_priority", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		raw := []byte(`{"name":"account-lifecycle","priority":11}`)
		resp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		b := readBody(t, resp)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("want 400 got %d body=%s", resp.StatusCode, b)
		}
		if errMsg := ginErrFromBody(t, b); !strings.Contains(errMsg, "priority") {
			t.Fatalf("error=%q want priority mention", errMsg)
		}
	})

	t.Run("Enqueue_validation_missing_name", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		raw := []byte(`{"payload":{}}`)
		resp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		b := readBody(t, resp)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("want 400 got %d body=%s", resp.StatusCode, b)
		}
	})

	t.Run("Enqueue_unknown_job_name", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		raw := []byte(`{"name":"not-a-registered-job-type","payload":{"k":"v"}}`)
		resp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		b := readBody(t, resp)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("want 400 got %d body=%s", resp.StatusCode, b)
		}
		if errMsg := ginErrFromBody(t, b); !strings.Contains(errMsg, "unknown job type") {
			t.Fatalf("error=%q want unknown job type mention", errMsg)
		}

		statsResp, err := client.Get(apiJobs(baseURL, "/stats"))
		if err != nil {
			t.Fatal(err)
		}
		statsBody := readBody(t, statsResp)
		if statsResp.StatusCode != http.StatusOK {
			t.Fatalf("stats: %d %s", statsResp.StatusCode, statsBody)
		}
		var stats service.JobStats
		mustDecodeJSON(t, bytes.NewReader(statsBody), &stats)
		if stats.Total != 0 {
			t.Fatalf("want no persisted jobs, stats=%+v", stats)
		}
	})

	t.Run("Get_not_found", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		id := metadata.GenerateJobID()
		req, err := http.NewRequest(http.MethodGet, apiJobs(baseURL, "/", id), nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b := readBody(t, resp)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("want 404 got %d body=%s", resp.StatusCode, b)
		}
	})

	t.Run("List_invalid_status", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		req, err := http.NewRequest(http.MethodGet, apiJobs(baseURL)+"?status=nope", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b := readBody(t, resp)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("want 400 got %d body=%s", resp.StatusCode, b)
		}
	})

	t.Run("List_filter_status_and_stats", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		statsResp0, err := client.Get(apiJobs(baseURL, "/stats"))
		if err != nil {
			t.Fatal(err)
		}
		statsBody0 := readBody(t, statsResp0)
		if statsResp0.StatusCode != http.StatusOK {
			t.Fatalf("stats: %d %s", statsResp0.StatusCode, statsBody0)
		}
		var stats0 service.JobStats
		mustDecodeJSON(t, bytes.NewReader(statsBody0), &stats0)
		if stats0.Total != 0 {
			t.Fatalf("want empty stats total=0 got %+v", stats0)
		}

		raw := []byte(`{"name":"account-lifecycle"}`)
		postResp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		postBody := readBody(t, postResp)
		if postResp.StatusCode != http.StatusCreated {
			t.Fatalf("enqueue: %d %s", postResp.StatusCode, postBody)
		}
		var created metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(postBody), &created)

		listResp, err := client.Get(apiJobs(baseURL) + "?status=pending_dispatch")
		if err != nil {
			t.Fatal(err)
		}
		listBody := readBody(t, listResp)
		if listResp.StatusCode != http.StatusOK {
			t.Fatalf("list: %d %s", listResp.StatusCode, listBody)
		}
		var listWrap struct {
			Jobs []metadata.JobMetadataModel `json:"jobs"`
		}
		mustDecodeJSON(t, bytes.NewReader(listBody), &listWrap)
		found := false
		for _, j := range listWrap.Jobs {
			if j.JobID == created.JobID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("pending list missing job %+v", listWrap.Jobs)
		}

		statsResp1, err := client.Get(apiJobs(baseURL, "/stats"))
		if err != nil {
			t.Fatal(err)
		}
		statsBody1 := readBody(t, statsResp1)
		var stats1 service.JobStats
		mustDecodeJSON(t, bytes.NewReader(statsBody1), &stats1)
		if stats1.Total < 1 || stats1.PendingDispatch < 1 {
			t.Fatalf("stats after enqueue: %+v", stats1)
		}
	})

	t.Run("Fail_cancel_retry_flow", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		raw := []byte(`{"name":"account-lifecycle"}`)
		postResp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		postBody := readBody(t, postResp)
		if postResp.StatusCode != http.StatusCreated {
			t.Fatalf("enqueue: %d %s", postResp.StatusCode, postBody)
		}
		var created metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(postBody), &created)
		markJobRunningForIntegrationTest(t, writer, created.JobID)

		failPayload := []byte(`{"errors":[{"error":"boom"}]}`)
		failResp, err := client.Post(apiJobs(baseURL, "/", created.JobID, "/fail"), "application/json", bytes.NewReader(failPayload))
		if err != nil {
			t.Fatal(err)
		}
		failBody := readBody(t, failResp)
		if failResp.StatusCode != http.StatusOK {
			t.Fatalf("fail: %d %s", failResp.StatusCode, failBody)
		}
		var failedJob metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(failBody), &failedJob)
		if len(failedJob.Errors) != 1 || failedJob.Errors[0].Error != "boom" {
			t.Fatalf("fail response errors=%v want one entry with boom", failedJob.Errors)
		}

		cancelResp, err := client.Post(apiJobs(baseURL, "/", created.JobID, "/cancel"), "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		cancelBody := readBody(t, cancelResp)
		if cancelResp.StatusCode != http.StatusBadRequest {
			t.Fatalf("cancel terminal want 400 got %d %s", cancelResp.StatusCode, cancelBody)
		}
		if !strings.Contains(ginErrFromBody(t, cancelBody), "cannot cancel") {
			t.Fatalf("unexpected cancel error: %s", cancelBody)
		}

		retryResp, err := client.Post(apiJobs(baseURL, "/", created.JobID, "/retry"), "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		retryBody := readBody(t, retryResp)
		if retryResp.StatusCode != http.StatusOK {
			t.Fatalf("retry: %d %s", retryResp.StatusCode, retryBody)
		}

		req, err := http.NewRequest(http.MethodGet, apiJobs(baseURL, "/", created.JobID), nil)
		if err != nil {
			t.Fatal(err)
		}
		getResp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		gotBody := readBody(t, getResp)
		if getResp.StatusCode != http.StatusOK {
			t.Fatalf("get after retry: %d %s", getResp.StatusCode, gotBody)
		}
		var got metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(gotBody), &got)
		if got.Status != metadata.JobStatusPendingDispatch {
			t.Fatalf("after retry status=%s want pending_dispatch", got.Status)
		}
		if len(got.Errors) != 1 || got.Errors[0].Error != "boom" {
			t.Fatalf("after retry errors=%v want preserved boom entry", got.Errors)
		}
	})

	t.Run("ErrorHistoryAcrossRetries", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		postResp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader([]byte(`{"name":"account-lifecycle"}`)))
		if err != nil {
			t.Fatal(err)
		}
		postBody := readBody(t, postResp)
		if postResp.StatusCode != http.StatusCreated {
			t.Fatalf("enqueue: %d %s", postResp.StatusCode, postBody)
		}
		var created metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(postBody), &created)

		markJobRunningForIntegrationTest(t, writer, created.JobID)

		firstFail := []byte(`{"errors":[{"error":"first failure"}]}`)
		failResp, err := client.Post(apiJobs(baseURL, "/", created.JobID, "/fail"), "application/json", bytes.NewReader(firstFail))
		if err != nil {
			t.Fatal(err)
		}
		failBody := readBody(t, failResp)
		if failResp.StatusCode != http.StatusOK {
			t.Fatalf("first fail: %d %s", failResp.StatusCode, failBody)
		}
		var afterFirstFail metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(failBody), &afterFirstFail)
		if len(afterFirstFail.Errors) != 1 {
			t.Fatalf("after first fail len(errors)=%d want 1", len(afterFirstFail.Errors))
		}
		if afterFirstFail.Errors[0].RetryAttempt != 0 {
			t.Fatalf("first error retryAttempt=%d want 0", afterFirstFail.Errors[0].RetryAttempt)
		}
		if afterFirstFail.Errors[0].Type != metadata.JobErrorTypeExecution {
			t.Fatalf("first error type=%q want execution", afterFirstFail.Errors[0].Type)
		}

		retryResp, err := client.Post(apiJobs(baseURL, "/", created.JobID, "/retry"), "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		retryBody := readBody(t, retryResp)
		if retryResp.StatusCode != http.StatusOK {
			t.Fatalf("retry: %d %s", retryResp.StatusCode, retryBody)
		}

		getAfterRetry := getJobViaHTTP(t, client, baseURL, created.JobID)
		if len(getAfterRetry.Errors) != 1 {
			t.Fatalf("after retry len(errors)=%d want 1 preserved", len(getAfterRetry.Errors))
		}
		if getAfterRetry.RetryCount != 1 {
			t.Fatalf("after retry retryCount=%d want 1", getAfterRetry.RetryCount)
		}

		markJobRunningForIntegrationTest(t, writer, created.JobID)

		secondFail := []byte(`{"errors":[{"error":"second failure"}]}`)
		failResp2, err := client.Post(apiJobs(baseURL, "/", created.JobID, "/fail"), "application/json", bytes.NewReader(secondFail))
		if err != nil {
			t.Fatal(err)
		}
		failBody2 := readBody(t, failResp2)
		if failResp2.StatusCode != http.StatusOK {
			t.Fatalf("second fail: %d %s", failResp2.StatusCode, failBody2)
		}
		var afterSecondFail metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(failBody2), &afterSecondFail)
		if len(afterSecondFail.Errors) != 2 {
			t.Fatalf("after second fail len(errors)=%d want 2", len(afterSecondFail.Errors))
		}
		if afterSecondFail.Errors[1].RetryAttempt != 1 {
			t.Fatalf("second error retryAttempt=%d want 1", afterSecondFail.Errors[1].RetryAttempt)
		}
		if afterSecondFail.GetLatestError() != "second failure" {
			t.Fatalf("GetLatestError()=%q want second failure", afterSecondFail.GetLatestError())
		}
	})

	t.Run("Cancel_pending_job", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		postResp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader([]byte(`{"name":"account-lifecycle"}`)))
		if err != nil {
			t.Fatal(err)
		}
		postBody := readBody(t, postResp)
		if postResp.StatusCode != http.StatusCreated {
			t.Fatalf("enqueue: %d %s", postResp.StatusCode, postBody)
		}
		var created metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(postBody), &created)

		cancelResp, err := client.Post(apiJobs(baseURL, "/", created.JobID, "/cancel"), "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		cancelBody := readBody(t, cancelResp)
		if cancelResp.StatusCode != http.StatusOK {
			t.Fatalf("cancel: %d %s", cancelResp.StatusCode, cancelBody)
		}
	})

	t.Run("Retry_non_failed_job", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		postResp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader([]byte(`{"name":"account-lifecycle"}`)))
		if err != nil {
			t.Fatal(err)
		}
		postBody := readBody(t, postResp)
		if postResp.StatusCode != http.StatusCreated {
			t.Fatalf("enqueue: %d %s", postResp.StatusCode, postBody)
		}
		var created metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(postBody), &created)

		retryResp, err := client.Post(apiJobs(baseURL, "/", created.JobID, "/retry"), "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		retryBody := readBody(t, retryResp)
		if retryResp.StatusCode != http.StatusBadRequest {
			t.Fatalf("retry want 400 got %d %s", retryResp.StatusCode, retryBody)
		}
	})

	t.Run("GetJobLogs_and_invalid_level_query", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		postResp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader([]byte(`{"name":"optimisation"}`)))
		if err != nil {
			t.Fatal(err)
		}
		postBody := readBody(t, postResp)
		if postResp.StatusCode != http.StatusCreated {
			t.Fatalf("enqueue: %d %s", postResp.StatusCode, postBody)
		}
		var created metadata.JobMetadataModel
		mustDecodeJSON(t, bytes.NewReader(postBody), &created)

		logsResp, err := client.Get(apiJobs(baseURL, "/", created.JobID, "/logs"))
		if err != nil {
			t.Fatal(err)
		}
		logsBody := readBody(t, logsResp)
		if logsResp.StatusCode != http.StatusOK {
			t.Fatalf("logs: %d %s", logsResp.StatusCode, logsBody)
		}
		var logsWrap struct {
			Logs []metadata.JobLog `json:"logs"`
		}
		mustDecodeJSON(t, bytes.NewReader(logsBody), &logsWrap)
		if len(logsWrap.Logs) < 1 {
			t.Fatalf("expected at least creation log, got %+v", logsWrap.Logs)
		}

		badResp, err := client.Get(apiJobs(baseURL, "/", created.JobID, "/logs") + "?levels=nope")
		if err != nil {
			t.Fatal(err)
		}
		badBody := readBody(t, badResp)
		if badResp.StatusCode != http.StatusBadRequest {
			t.Fatalf("bad levels query want 400 got %d %s", badResp.StatusCode, badBody)
		}
	})

	t.Run("FailJob_not_found", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		id := metadata.GenerateJobID()
		resp, err := client.Post(apiJobs(baseURL, "/", id, "/fail"), "application/json", bytes.NewReader([]byte(`{"errors":[{"error":"x"}]}`)))
		if err != nil {
			t.Fatal(err)
		}
		b := readBody(t, resp)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("want 404 got %d %s", resp.StatusCode, b)
		}
	})
}
