package metadata

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

func buildListQuery(filter ListFilter) bson.M {
	query := bson.M{}

	if len(filter.Names) > 0 {
		query["name"] = bson.M{"$in": filter.Names}
	}

	if len(filter.Statuses) > 0 {
		query["status"] = bson.M{"$in": filter.Statuses}
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

func buildLogsQuery(jobID string, filter LogFilter) bson.M {
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
