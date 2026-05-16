package main

import (
	"context"
	"log"
	"net/http"

	"github.com/daconjurer/jobby/internal/jobs/handler"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/daconjurer/jobby/internal/settings"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	mongoConfig := metadata.MongoConfig{
		URI:                settings.GetEnvOrPanic("MONGODB_URI"),
		Database:           settings.GetEnvOrPanic("MONGODB_DATABASE"),
		CollectionMetadata: settings.GetEnvOrPanic("MONGODB_COLLECTION_METADATA"),
		CollectionLogs:     settings.GetEnvOrPanic("MONGODB_COLLECTION_LOGS"),
		Timeout:            settings.ParseDuration(settings.GetEnv("MONGODB_TIMEOUT", "10s")),
		MaxPoolSize:        settings.ParseUint64(settings.GetEnv("MONGODB_MAX_POOL_SIZE", "100")),
		MinPoolSize:        settings.ParseUint64(settings.GetEnv("MONGODB_MIN_POOL_SIZE", "10")),
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
	jobsHandler := handler.NewJobsHandler(metadataSvc)

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
		jobs.GET("", jobsHandler.ListJobs)
		jobs.POST("", jobsHandler.EnqueueJob)
		jobs.GET("/stats", jobsHandler.GetJobStats)
		jobs.GET("/:id", jobsHandler.GetJob)
		jobs.POST("/:id/fail", jobsHandler.FailJob)
		jobs.POST("/:id/cancel", jobsHandler.CancelJob)
		jobs.POST("/:id/retry", jobsHandler.RetryJob)
		jobs.GET("/:id/logs", jobsHandler.GetJobLogs)
	}

	port := settings.GetEnvOrPanic("PORT")
	log.Printf("Starting jobs service on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
