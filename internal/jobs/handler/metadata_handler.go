package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/gin-gonic/gin"
)

// MetadataHandler handles HTTP requests for job metadata.
type MetadataHandler struct {
	svc *service.MetadataService
}

// NewMetadataHandler creates a new metadata handler.
func NewMetadataHandler(svc *service.MetadataService) *MetadataHandler {
	return &MetadataHandler{svc: svc}
}

// CreateJobRequest represents the request body for creating a job.
type CreateJobRequest struct {
	Name     string         `json:"name" binding:"required"`
	Payload  map[string]any `json:"payload"`
	Priority *int           `json:"priority"`
	Tags     []string       `json:"tags"`
	Metadata map[string]any `json:"metadata"`
}

// CreateJob handles POST /api/jobs.
func (h *MetadataHandler) CreateJob(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Priority != nil && (*req.Priority < 0 || *req.Priority > 10) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "priority must be between 0 and 10"})
		return
	}

	options := service.CreateJobOptions{
		Priority: req.Priority,
		Tags:     req.Tags,
		Metadata: req.Metadata,
	}

	job, err := h.svc.CreateJob(c.Request.Context(), req.Name, req.Payload, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, job)
}

// GetJob handles GET /api/jobs/:id.
func (h *MetadataHandler) GetJob(c *gin.Context) {
	jobID := c.Param("id")

	job, err := h.svc.GetJob(c.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, metadata.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}

// ListJobs handles GET /api/jobs.
func (h *MetadataHandler) ListJobs(c *gin.Context) {
	filter := metadata.ListFilter{
		Limit:    parseIntQuery(c, "limit", 50),
		Skip:     parseIntQuery(c, "skip", 0),
		SortBy:   c.DefaultQuery("sortBy", "createdAt"),
		SortDesc: c.DefaultQuery("sortDesc", "true") == "true",
	}

	if statusStr := c.Query("status"); statusStr != "" {
		st := metadata.JobStatus(statusStr)
		if !st.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid status: %q", statusStr)})
			return
		}
		filter.Statuses = []metadata.JobStatus{st}
	}

	if tags := c.QueryArray("tags"); len(tags) > 0 {
		filter.Tags = tags
	}
	if names := c.QueryArray("names"); len(names) > 0 {
		filter.Names = names
	}

	jobs, err := h.svc.ListJobs(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// StartJob handles POST /api/jobs/:id/start.
func (h *MetadataHandler) StartJob(c *gin.Context) {
	jobID := c.Param("id")

	if err := h.svc.StartJob(c.Request.Context(), jobID); err != nil {
		if errors.Is(err, metadata.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job started"})
}

// CompleteJobRequest represents the request body for completing a job.
type CompleteJobRequest struct {
	Result map[string]any `json:"result"`
}

// CompleteJob handles POST /api/jobs/:id/complete.
func (h *MetadataHandler) CompleteJob(c *gin.Context) {
	jobID := c.Param("id")

	var req CompleteJobRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.CompleteJob(c.Request.Context(), jobID, req.Result); err != nil {
		if errors.Is(err, metadata.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job completed"})
}

// FailJobRequest represents the request body for failing a job.
type FailJobRequest struct {
	Error string `json:"error" binding:"required"`
}

// FailJob handles POST /api/jobs/:id/fail.
func (h *MetadataHandler) FailJob(c *gin.Context) {
	jobID := c.Param("id")

	var req FailJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.FailJob(c.Request.Context(), jobID, fmt.Errorf("%s", req.Error)); err != nil {
		if errors.Is(err, metadata.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job marked as failed"})
}

// CancelJobRequest represents the request body for cancelling a job.
type CancelJobRequest struct {
	Reason string `json:"reason"`
}

// CancelJob handles POST /api/jobs/:id/cancel.
func (h *MetadataHandler) CancelJob(c *gin.Context) {
	jobID := c.Param("id")

	var req CancelJobRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.CancelJob(c.Request.Context(), jobID, req.Reason); err != nil {
		if errors.Is(err, metadata.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job cancelled"})
}

// RetryJob handles POST /api/jobs/:id/retry.
func (h *MetadataHandler) RetryJob(c *gin.Context) {
	jobID := c.Param("id")

	if err := h.svc.RetryJob(c.Request.Context(), jobID); err != nil {
		if errors.Is(err, metadata.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job retry initiated"})
}

// GetJobLogs handles GET /api/jobs/:id/logs.
func (h *MetadataHandler) GetJobLogs(c *gin.Context) {
	jobID := c.Param("id")

	filter := metadata.LogFilter{
		Limit: parseIntQuery(c, "limit", 100),
		Skip:  parseIntQuery(c, "skip", 0),
	}

	if levels := c.QueryArray("levels"); len(levels) > 0 {
		filter.Levels = make([]metadata.LogLevel, len(levels))
		for i, l := range levels {
			lvl := metadata.LogLevel(l)
			if !lvl.IsValid() {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid log level: %q", l)})
				return
			}
			filter.Levels[i] = lvl
		}
	}

	logs, err := h.svc.GetJobLogs(c.Request.Context(), jobID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"count": len(logs),
	})
}

// GetJobStats handles GET /api/jobs/stats.
func (h *MetadataHandler) GetJobStats(c *gin.Context) {
	stats, err := h.svc.GetJobStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	if value := c.Query(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func bindOptionalJSON(c *gin.Context, v any) error {
	if c.Request.ContentLength == 0 {
		return nil
	}
	return c.ShouldBindJSON(v)
}
