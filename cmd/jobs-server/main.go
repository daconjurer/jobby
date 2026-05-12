package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/daconjurer/jobby/internal/jobs/handler"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

func main() {
	ctx := context.Background()

	mongoConfig := metadata.MongoConfig{
		URI:                getEnv("MONGODB_URI", "mongodb://jobby_app:jobby_app_pass@localhost:27018/jobby?authSource=jobby"),
		Database:           getEnv("MONGODB_DATABASE", "jobby"),
		CollectionMetadata: getEnv("MONGODB_COLLECTION_METADATA", "job_metadata"),
		CollectionLogs:     getEnv("MONGODB_COLLECTION_LOGS", "job_logs"),
		Timeout:            parseDuration(getEnv("MONGODB_TIMEOUT", "10s")),
		MaxPoolSize:        parseUint64(getEnv("MONGODB_MAX_POOL_SIZE", "100")),
		MinPoolSize:        parseUint64(getEnv("MONGODB_MIN_POOL_SIZE", "10")),
	}

	reader, writer, mongoClient, err := metadata.OpenMongoJobs(ctx, mongoConfig)
	if err != nil {
		log.Fatalf("Failed to connect MongoDB jobs persistence: %v", err)
	}
	defer func() { _ = mongoClient.Disconnect(context.Background()) }()

	log.Println("Connected to MongoDB (jobby database)")
	if !reader.IndexesPresent {
		log.Println("warning: one or more expected indexes are missing (see mongo-init)")
	}

	metadataSvc := service.NewMetadataService(reader, writer)
	metaHandler := handler.NewMetadataHandler(metadataSvc)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"database": "connected",
		})
	})

	apiRoutes := r.Group("/api")
	jobs := apiRoutes.Group("/jobs")
	{
		jobs.GET("", metaHandler.ListJobs)
		jobs.POST("", metaHandler.CreateJob)
		jobs.GET("/stats", metaHandler.GetJobStats)
		jobs.GET("/:id", metaHandler.GetJob)
		jobs.POST("/:id/start", metaHandler.StartJob)
		jobs.POST("/:id/complete", metaHandler.CompleteJob)
		jobs.POST("/:id/fail", metaHandler.FailJob)
		jobs.POST("/:id/cancel", metaHandler.CancelJob)
		jobs.POST("/:id/retry", metaHandler.RetryJob)
		jobs.GET("/:id/logs", metaHandler.GetJobLogs)
	}

	port := getEnv("PORT", "3001")
	log.Printf("Starting jobs service on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

func parseUint64(s string) uint64 {
	var v uint64
	_, _ = fmt.Sscanf(s, "%d", &v)
	return v
}
