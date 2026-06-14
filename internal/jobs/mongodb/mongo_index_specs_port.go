package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// verifyRequiredIndexesPresent reports whether ListSpecifications exposes every name in required.
//
// Only listing indexes yields a non-nil error (e.g. network or permission failure).
// Missing names set present=false with err=nil — callers decide whether to log or degrade gracefully.
func verifyRequiredIndexesPresent(ctx context.Context, coll *mongo.Collection, required []string) (present bool, err error) {
	label := coll.Name()
	specs, err := coll.Indexes().ListSpecifications(ctx)
	if err != nil {
		return false, fmt.Errorf("list indexes for %s: %w", label, err)
	}

	have := make(map[string]struct{}, len(specs))
	for _, s := range specs {
		have[s.Name] = struct{}{}
	}

	for _, name := range required {
		if _, ok := have[name]; !ok {
			return false, nil
		}
	}

	return true, nil
}
