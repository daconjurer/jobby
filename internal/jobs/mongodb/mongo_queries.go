package mongodb

import (
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// statusToSubFields maps a display status to the underlying (dispatchStatus, executionStatus) pair.
func statusToSubFields(status metadata.JobStatus) (metadata.DispatchStatus, metadata.ExecutionStatus) {
	switch status {
	case metadata.JobStatusPendingDispatch:
		return metadata.DispatchStatusPending, metadata.ExecutionStatusNotStarted
	case metadata.JobStatusDispatched:
		return metadata.DispatchStatusDispatched, metadata.ExecutionStatusNotStarted
	case metadata.JobStatusDispatchFailed:
		return metadata.DispatchStatusFailed, metadata.ExecutionStatusNotStarted
	case metadata.JobStatusRunning:
		return metadata.DispatchStatusDispatched, metadata.ExecutionStatusRunning
	case metadata.JobStatusCompleted:
		return metadata.DispatchStatusDispatched, metadata.ExecutionStatusCompleted
	case metadata.JobStatusFailed:
		return metadata.DispatchStatusDispatched, metadata.ExecutionStatusFailed
	case metadata.JobStatusCancelled:
		return metadata.DispatchStatusDispatched, metadata.ExecutionStatusCancelled
	default:
		return "", ""
	}
}

func buildListQuery(filter metadata.ListFilter) bson.M {
	query := bson.M{}

	if len(filter.Names) > 0 {
		query["name"] = bson.M{"$in": filter.Names}
	}

	if len(filter.Statuses) > 0 {
		query["$or"] = buildStatusOrQuery(filter.Statuses)
	}

	if len(filter.Tags) > 0 {
		query["tags"] = bson.M{"$in": filter.Tags}
	}

	if filter.MinPriority != nil || filter.MaxPriority != nil {
		priorityQuery := bson.M{}
		if filter.MinPriority != nil {
			priorityQuery["$gte"] = *filter.MinPriority
		}
		if filter.MaxPriority != nil {
			priorityQuery["$lte"] = *filter.MaxPriority
		}
		query["priority"] = priorityQuery
	}

	if filter.CreatedAfter != nil || filter.CreatedBefore != nil {
		createdQuery := bson.M{}
		if filter.CreatedAfter != nil {
			createdQuery["$gt"] = *filter.CreatedAfter
		}
		if filter.CreatedBefore != nil {
			createdQuery["$lt"] = *filter.CreatedBefore
		}
		query["createdAt"] = createdQuery
	}

	return query
}

// buildStatusOrQuery returns an $or array where each display status is translated to
// its (dispatchStatus, executionStatus) pair. Multiple statuses that resolve to the
// same pair are deduplicated.
func buildStatusOrQuery(statuses []metadata.JobStatus) bson.A {
	seen := make(map[string]struct{}, len(statuses))
	var conditions bson.A
	for _, status := range statuses {
		ds, es := statusToSubFields(status)
		if ds == "" && es == "" {
			continue
		}
		key := string(ds) + "\x00" + string(es)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		conditions = append(conditions, bson.M{
			"dispatchStatus":  ds,
			"executionStatus": es,
		})
	}
	return conditions
}

func buildLogsQuery(jobID string, filter metadata.LogFilter) bson.M {
	query := bson.M{"jobId": jobID}

	if len(filter.Levels) > 0 {
		query["level"] = bson.M{"$in": filter.Levels}
	}

	if filter.Since != nil || filter.Until != nil {
		timestampQuery := bson.M{}
		if filter.Since != nil {
			timestampQuery["$gte"] = *filter.Since
		}
		if filter.Until != nil {
			timestampQuery["$lte"] = *filter.Until
		}
		query["timestamp"] = timestampQuery
	}

	return query
}
