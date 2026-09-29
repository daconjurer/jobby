package mongodb

import (
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// statusToSubFields maps a display status to the underlying CompositeStatus.
func statusToSubFields(status metadata.JobStatus) metadata.CompositeStatus {
	switch status {
	case metadata.JobStatusPendingDispatch:
		return metadata.CompositeStatus{Dispatch: metadata.DispatchStatusPending, Execution: metadata.ExecutionStatusNotStarted}
	case metadata.JobStatusDispatched:
		return metadata.CompositeStatus{Dispatch: metadata.DispatchStatusDispatched, Execution: metadata.ExecutionStatusNotStarted}
	case metadata.JobStatusDispatchFailed:
		return metadata.CompositeStatus{Dispatch: metadata.DispatchStatusFailed, Execution: metadata.ExecutionStatusNotStarted}
	case metadata.JobStatusRunning:
		return metadata.CompositeStatus{Dispatch: metadata.DispatchStatusDispatched, Execution: metadata.ExecutionStatusRunning}
	case metadata.JobStatusCompleted:
		return metadata.CompositeStatus{Dispatch: metadata.DispatchStatusDispatched, Execution: metadata.ExecutionStatusCompleted}
	case metadata.JobStatusFailed:
		return metadata.CompositeStatus{Dispatch: metadata.DispatchStatusDispatched, Execution: metadata.ExecutionStatusFailed}
	case metadata.JobStatusCancelled:
		return metadata.CompositeStatus{Dispatch: metadata.DispatchStatusDispatched, Execution: metadata.ExecutionStatusCancelled}
	default:
		return metadata.CompositeStatus{}
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
// its CompositeStatus. Multiple statuses that resolve to the same pair are deduplicated.
func buildStatusOrQuery(statuses []metadata.JobStatus) bson.A {
	seen := make(map[string]struct{}, len(statuses))
	var conditions bson.A
	for _, status := range statuses {
		pair := statusToSubFields(status)
		if !pair.IsValid() {
			continue
		}
		key := string(pair.Dispatch) + "\x00" + string(pair.Execution)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		conditions = append(conditions, bson.M{
			"dispatchStatus":  pair.Dispatch,
			"executionStatus": pair.Execution,
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
