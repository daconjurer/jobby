package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/daconjurer/jobby/internal/jobs/appruntime"
	jobshttp "github.com/daconjurer/jobby/internal/jobs/http"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("cannot start jobs-server: %v", err)
	}

	rt, cleanup, err := appruntime.Bootstrap(ctx, appruntime.Config{
		Mongo:            cfg.Mongo,
		TopicsConfigPath: cfg.Topics.ConfigPath,
	})
	if err != nil {
		log.Fatalf("Failed to bootstrap jobs runtime: %v", err)
	}
	defer cleanup()

	log.Println("Connected to MongoDB (jobby database)")

	jobsHandler := jobshttp.NewJobsHandler(rt.Metadata, rt.Enqueue)

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
