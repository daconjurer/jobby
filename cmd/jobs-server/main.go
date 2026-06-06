package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	jobshttp "github.com/daconjurer/jobby/internal/jobs/http"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	jobpulsar "github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("cannot start jobs-server: %v", err)
	}

	reader, writer, mongoClient, err := mongodb.OpenMongoJobs(ctx, cfg.Mongo)
	if err != nil {
		log.Fatalf("Failed to connect MongoDB jobs persistence: %v", err)
	}
	defer func() { _ = mongoClient.Disconnect(context.Background()) }()

	log.Println("Connected to MongoDB (jobby database)")
	if !reader.IndexesPresent {
		log.Println("warning: one or more expected indexes are missing (make sure the migrations are applied)")
	}

	topicResolver, err := jobpulsar.NewFileTopicResolver(cfg.Topics.ConfigPath)
	if err != nil {
		log.Fatalf("Failed to load job topics config: %v", err)
	}

	metadataSvc := service.NewMetadataService(reader, writer)
	enqueueSvc := service.NewEnqueueService(metadataSvc, topicResolver)
	jobsHandler := jobshttp.NewJobsHandler(metadataSvc, enqueueSvc)

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

	port := cfg.Server.Port
	log.Printf("Starting jobs service on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
