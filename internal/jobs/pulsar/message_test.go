package pulsar

import (
	"encoding/json"
	"strings"
	"testing"
)

type samplePayload struct {
	StoreID string `json:"storeId"`
	Limit   int    `json:"limit"`
}

func TestJobMessage_JSONRoundTrip(t *testing.T) {
	original, err := NewJobMessage("550e8400-e29b-41d4-a716-446655440000", "load-products", samplePayload{
		StoreID: "store-1",
		Limit:   50,
	})
	if err != nil {
		t.Fatalf("NewJobMessage: %v", err)
	}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	decoded, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.JobID != original.JobID || decoded.Name != original.Name {
		t.Fatalf("identity fields: got %+v want %+v", decoded, original)
	}
	if string(decoded.Payload) != string(original.Payload) {
		t.Fatalf("payload: got %s want %s", decoded.Payload, original.Payload)
	}

	got, err := DecodePayload[samplePayload](decoded)
	if err != nil {
		t.Fatalf("DecodePayload: %v", err)
	}
	want := samplePayload{StoreID: "store-1", Limit: 50}
	if got != want {
		t.Fatalf("typed payload: got %+v want %+v", got, want)
	}
}

func TestJobMessage_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		msg  JobMessage
		want string
	}{
		{
			name: "missing jobId",
			msg:  JobMessage{Name: "load-products", Payload: json.RawMessage(`{}`)},
			want: "jobId",
		},
		{
			name: "missing name",
			msg:  JobMessage{JobID: "id-1", Payload: json.RawMessage(`{}`)},
			want: "name",
		},
		{
			name: "whitespace jobId",
			msg:  JobMessage{JobID: "   ", Name: "load-products", Payload: json.RawMessage(`{}`)},
			want: "jobId",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Marshal(tt.msg)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Marshal: got %v want error containing %q", err, tt.want)
			}
			_, err = json.Marshal(tt.msg)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("json.Marshal: got %v want error containing %q", err, tt.want)
			}
		})
	}
}

func TestJobMessage_UnmarshalInvalidJSON(t *testing.T) {
	_, err := Unmarshal([]byte(`{"jobId":"x"`))
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestJobMessage_DecodePayload_EmptyPayload(t *testing.T) {
	msg := JobMessage{JobID: "id-1", Name: "load-products"}
	_, err := DecodePayload[samplePayload](msg)
	if err == nil || !strings.Contains(err.Error(), "payload is empty") {
		t.Fatalf("got %v want empty payload error", err)
	}
}

func TestNewJobMessage_InvalidIdentity(t *testing.T) {
	_, err := NewJobMessage("", "load-products", samplePayload{StoreID: "s"})
	if err == nil || !strings.Contains(err.Error(), "jobId") {
		t.Fatalf("got %v want jobId validation error", err)
	}
}
