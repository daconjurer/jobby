package metadata

import (
	"fmt"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// bsonPartialSet walks a patch struct whose exported fields are pointers. For each field:
//   - bson tag selects the MongoDB document key (first segment only; ignores options like omitempty).
//   - bson:"-" means the field is skipped (disallowed for nullable patch semantics here).
//   - nil pointer: omit key from $set (unchanged in DB).
//   - non-nil pointer: dereference and assign to $set.
//
// Intended for structs like UpdateJob; all exported fields must be pointers and carry a non-dash bson name.
func bsonPartialSet[T any](p *T) (bson.M, error) {
	if p == nil {
		return nil, fmt.Errorf("bsonPartialSet: nil *T")
	}
	rv := reflect.ValueOf(p)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return nil, fmt.Errorf("bsonPartialSet: require non-nil pointer to struct, got %v", rv.Kind())
	}
	rv = rv.Elem()
	rt := rv.Type()
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("bsonPartialSet: T must be a struct, got %s", rv.Kind())
	}

	out := bson.M{}
	seenKeys := map[string]string{}

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if !sf.IsExported() {
			continue
		}
		tag := sf.Tag.Get("bson")
		key, ok := bsonNameFromTag(tag)
		if !ok {
			return nil, fmt.Errorf("bsonPartialSet: exported field %s.%s requires a bson tag with a document name (not '-')", rt.Name(), sf.Name)
		}
		first := sf.Name
		if prevField, dup := seenKeys[key]; dup {
			return nil, fmt.Errorf("bsonPartialSet: duplicate bson key %q on fields %q and %q", key, prevField, first)
		}
		seenKeys[key] = first

		fv := rv.Field(i)
		if fv.Kind() != reflect.Pointer {
			return nil, fmt.Errorf("bsonPartialSet: field %s.%s must be a pointer type, got %s", rt.Name(), sf.Name, fv.Kind())
		}
		if fv.IsNil() {
			continue
		}
		out[key] = fv.Elem().Interface()
	}

	return out, nil
}

// bsonNameFromTag returns the document field name from a bson struct tag string.
func bsonNameFromTag(tag string) (name string, ok bool) {
	if tag == "" {
		return "", false
	}
	parts := strings.Split(tag, ",")
	key := strings.TrimSpace(parts[0])
	if key == "" || key == "-" {
		return "", false
	}
	return key, true
}
