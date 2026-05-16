// MongoDB Initialization Script
// This script runs automatically on first container start
// It creates the jobby database with validated collections

// Switch to the jobby database (single database for all collections)
db = db.getSiblingDB("jobby");

print("==> Initializing jobby database");

// ============================================================================
// Collection: job_metadata
// Purpose: Track job execution metadata with consistent structure
// ============================================================================

print("==> Creating job_metadata collection with schema validation");

db.createCollection("job_metadata", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["jobId", "name", "status", "createdAt"],
      properties: {
        jobId: {
          bsonType: "string",
          description: "Unique job identifier (UUID) - REQUIRED",
          pattern:
            "^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$",
        },
        name: {
          bsonType: "string",
          description: "Job type name - REQUIRED",
          minLength: 1,
          maxLength: 100,
        },
        status: {
          enum: ["pending", "running", "completed", "failed", "cancelled"],
          description: "Current job execution status - REQUIRED",
        },
        priority: {
          bsonType: "int",
          minimum: 0,
          maximum: 10,
          description: "Job priority (0=lowest, 10=highest, default=5)",
        },
        createdAt: {
          bsonType: "date",
          description: "Job creation timestamp - REQUIRED",
        },
        startedAt: {
          bsonType: ["date", "null"],
          description: "Job start timestamp (null if not started)",
        },
        completedAt: {
          bsonType: ["date", "null"],
          description: "Job completion timestamp (null if not completed)",
        },
        payload: {
          bsonType: "object",
          description: "Job-specific data with dynamic schema",
        },
        metadata: {
          bsonType: "object",
          description: "Additional metadata fields (key-value pairs)",
        },
        error: {
          bsonType: ["string", "null"],
          description: "Error message if job failed (null if successful)",
        },
        retryCount: {
          bsonType: "int",
          minimum: 0,
          description: "Number of retry attempts (default=0)",
        },
        tags: {
          bsonType: "array",
          items: {
            bsonType: "string",
          },
          uniqueItems: true,
          description: "Job tags for filtering and categorization",
        },
      },
    },
  },
  validationLevel: "strict",
  validationAction: "error",
});

print("==> Creating indexes for job_metadata collection");

// Unique index on jobId (primary lookup field)
db.job_metadata.createIndex(
  { jobId: 1 },
  { unique: true, name: "idx_jobId_unique" },
);

// Index on name for filtering by job type
db.job_metadata.createIndex({ name: 1 }, { name: "idx_name" });

// Index on status for filtering by execution state
db.job_metadata.createIndex({ status: 1 }, { name: "idx_status" });

// Index on createdAt for time-based queries and sorting (descending)
db.job_metadata.createIndex({ createdAt: -1 }, { name: "idx_createdAt_desc" });

// Index on tags for filtering (supports queries like "find jobs with tag X")
db.job_metadata.createIndex({ tags: 1 }, { name: "idx_tags" });

// Compound index on name + status for common query pattern
db.job_metadata.createIndex(
  { name: 1, status: 1 },
  { name: "idx_name_status" },
);

// Compound index on status + priority for job queue queries
db.job_metadata.createIndex(
  { status: 1, priority: -1, createdAt: 1 },
  { name: "idx_status_priority_created" },
);

print("==> job_metadata collection initialized successfully");

// ============================================================================
// Collection: job_logs
// Purpose: Store structured logs for job execution events
// ============================================================================

print("==> Creating job_logs collection with schema validation");

db.createCollection("job_logs", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["jobId", "timestamp", "level", "message"],
      properties: {
        jobId: {
          bsonType: "string",
          description: "Associated job ID (links to job_metadata) - REQUIRED",
          pattern:
            "^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$",
        },
        timestamp: {
          bsonType: "date",
          description: "Log entry timestamp - REQUIRED",
        },
        level: {
          enum: ["debug", "info", "warn", "error", "fatal"],
          description: "Log severity level - REQUIRED",
        },
        message: {
          bsonType: "string",
          description: "Log message text - REQUIRED",
          minLength: 1,
          maxLength: 10000,
        },
        context: {
          bsonType: "object",
          description: "Additional context data (structured logging)",
        },
        source: {
          bsonType: "string",
          description: 'Source of the log entry (e.g., "executor", "handler")',
          maxLength: 100,
        },
        stackTrace: {
          bsonType: ["string", "null"],
          description: "Stack trace for error logs (null for non-errors)",
        },
      },
    },
  },
  validationLevel: "strict",
  validationAction: "error",
});

print("==> Creating indexes for job_logs collection");

// Index on jobId for fetching all logs for a specific job
db.job_logs.createIndex(
  { jobId: 1, timestamp: -1 },
  { name: "idx_jobId_timestamp_desc" },
);

// Index on timestamp for time-range queries
db.job_logs.createIndex({ timestamp: -1 }, { name: "idx_timestamp_desc" });

// Index on level for filtering by severity
db.job_logs.createIndex({ level: 1 }, { name: "idx_level" });

// Compound index for common query: logs for a job at a specific level
db.job_logs.createIndex(
  { jobId: 1, level: 1, timestamp: -1 },
  { name: "idx_jobId_level_timestamp" },
);

print("==> job_logs collection initialized successfully");

// ============================================================================
// Create Application User (Non-Root)
// ============================================================================

print("==> Creating application user");

db.createUser({
  user: "jobby_app",
  pwd: "jobby_app_pass",
  roles: [
    {
      role: "readWrite",
      db: "jobby",
    },
  ],
});

print("==> Application user created successfully");

// ============================================================================
// Verification
// ============================================================================

print("==> Verifying database setup");

const collections = db.getCollectionNames();
print("Collections created: " + collections.join(", "));

const metadataIndexes = db.job_metadata.getIndexes();
print("job_metadata indexes: " + metadataIndexes.length);

const logsIndexes = db.job_logs.getIndexes();
print("job_logs indexes: " + logsIndexes.length);

print("==> MongoDB initialization complete!");
print("");
print("Database: jobby");
print("Collections: job_metadata, job_logs");
print("Root user: jobby_admin");
print("App user: jobby_app");
print("");
