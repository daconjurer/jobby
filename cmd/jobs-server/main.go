package main

import (
	"context"
	"log"
	"net/http"

	"github.com/daconjurer/jobby/internal/jobs/handler"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	mongoConfig, err := loadMongoMetadataConfig()
	if err != nil {
		log.Fatalf("Failed to load MongoDB configuration: %v", err)
	}

	serverCfg, err := loadServerListenConfig()
	if err != nil {
		log.Fatalf("Failed to load server configuration: %v", err)
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

	port := serverCfg.Port
	log.Printf("Starting jobs service on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
