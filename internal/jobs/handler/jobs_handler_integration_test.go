//go:build integration

package handler

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
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Integration tests require MongoDB (for example: make mongo-up).
// Run: make test-integration
//
// Required: MONGODB_URI.
// Optional (defaults match cmd/jobs-server and .env.example): MONGODB_DATABASE, MONGODB_COLLECTION_METADATA,
// MONGODB_COLLECTION_LOGS.
//
// Database lifecycle matches internal/jobs/metadata/mongo_integration_test.go: collections are cleared
// before each subtest and again on teardown so runs stay idempotent.

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func integrationMongoEnv(tb testing.TB) metadata.MongoConfig {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test (-short)")
	}
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		tb.Fatalf("MONGODB_URI is not set (required for integration tests; see .env and compose.yml files)")
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
	return metadata.MongoConfig{
		URI:                uri,
		Database:           db,
		CollectionMetadata: metaColl,
		CollectionLogs:     logsColl,
		Timeout:            30 * time.Second,
		MaxPoolSize:        50,
		MinPoolSize:        0,
	}
}

func clearJobCollections(ctx context.Context, db *mongo.Database, cfg metadata.MongoConfig) error {
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

func setupIntegrationCollections(ctx context.Context, db *mongo.Database, cfg metadata.MongoConfig) error {
	return clearJobCollections(ctx, db, cfg)
}

func teardownIntegrationCollections(ctx context.Context, db *mongo.Database, cfg metadata.MongoConfig) {
	_ = clearJobCollections(ctx, db, cfg)
}

// prepareIntegrationMongoPersistence opens reader/writer against MongoDB, clears both collections before the test,
// and registers teardown cleanup so writes do not leak across runs (same pattern as metadata integration tests).
func prepareIntegrationMongoPersistence(t *testing.T) (*metadata.MongoJobsReader, *metadata.MongoJobsWriter) {
	t.Helper()
	cfg := integrationMongoEnv(t)
	ctx := context.Background()

	reader, writer, client, err := metadata.OpenMongoJobs(ctx, cfg)
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

func startJobsIntegrationServer(tb testing.TB, reader *metadata.MongoJobsReader, writer *metadata.MongoJobsWriter) (baseURL string, httpClient *http.Client) {
	tb.Helper()
	svc := service.NewMetadataService(reader, writer)
	h := NewJobsHandler(svc)

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
			"name":     "integration-http-job",
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
		if created.JobID == "" || created.Name != "integration-http-job" {
			t.Fatalf("unexpected created job: %+v", created)
		}
		if created.Status != metadata.JobStatusPending {
			t.Fatalf("status=%s want pending", created.Status)
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

		raw := []byte(`{"name":"x","priority":11}`)
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

		raw := []byte(`{"name":"listed-job"}`)
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

		listResp, err := client.Get(apiJobs(baseURL) + "?status=pending")
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
		if stats1.Total < 1 || stats1.Pending < 1 {
			t.Fatalf("stats after enqueue: %+v", stats1)
		}
	})

	t.Run("Fail_cancel_retry_flow", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		raw := []byte(`{"name":"flow-job"}`)
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

		failPayload := []byte(`{"error":"boom"}`)
		failResp, err := client.Post(apiJobs(baseURL, "/", created.JobID, "/fail"), "application/json", bytes.NewReader(failPayload))
		if err != nil {
			t.Fatal(err)
		}
		failBody := readBody(t, failResp)
		if failResp.StatusCode != http.StatusOK {
			t.Fatalf("fail: %d %s", failResp.StatusCode, failBody)
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
		if got.Status != metadata.JobStatusPending {
			t.Fatalf("after retry status=%s want pending", got.Status)
		}
	})

	t.Run("Cancel_pending_job", func(t *testing.T) {
		reader, writer := prepareIntegrationMongoPersistence(t)
		baseURL, client := startJobsIntegrationServer(t, reader, writer)

		postResp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader([]byte(`{"name":"cancel-me"}`)))
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

		postResp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader([]byte(`{"name":"still-pending"}`)))
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

		postResp, err := client.Post(apiJobs(baseURL), "application/json", bytes.NewReader([]byte(`{"name":"log-job"}`)))
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
		resp, err := client.Post(apiJobs(baseURL, "/", id, "/fail"), "application/json", bytes.NewReader([]byte(`{"error":"x"}`)))
		if err != nil {
			t.Fatal(err)
		}
		b := readBody(t, resp)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("want 404 got %d %s", resp.StatusCode, b)
		}
	})
}
