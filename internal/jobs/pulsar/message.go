package pulsar

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// JobMessage is the JSON envelope published to Pulsar and consumed by the executor.
type JobMessage struct {
	JobID   string          `json:"jobId"`
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload"`
}

// MarshalJSON validates required fields then encodes the envelope.
func (m JobMessage) MarshalJSON() ([]byte, error) {
	if err := m.validate(); err != nil {
		return nil, err
	}
	type alias JobMessage
	return json.Marshal(alias(m))
}

// UnmarshalJSON decodes the envelope and validates required fields.
func (m *JobMessage) UnmarshalJSON(data []byte) error {
	type alias JobMessage
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*m = JobMessage(decoded)
	return m.validate()
}

func (m JobMessage) validate() error {
	if strings.TrimSpace(m.JobID) == "" {
		return errors.New("job message: jobId is required")
	}
	if strings.TrimSpace(m.Name) == "" {
		return errors.New("job message: name is required")
	}
	return nil
}

// Marshal encodes a validated JobMessage to JSON bytes.
func Marshal(msg JobMessage) ([]byte, error) {
	return json.Marshal(msg)
}

// Unmarshal decodes JSON bytes into JobMessage with validation.
func Unmarshal(data []byte) (JobMessage, error) {
	var msg JobMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return JobMessage{}, err
	}
	return msg, nil
}

// NewJobMessage builds an envelope with a typed payload marshaled into Payload.
func NewJobMessage[T any](jobID, name string, payload T) (JobMessage, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return JobMessage{}, fmt.Errorf("marshal job payload: %w", err)
	}
	msg := JobMessage{
		JobID:   jobID,
		Name:    name,
		Payload: raw,
	}
	if err := msg.validate(); err != nil {
		return JobMessage{}, err
	}
	return msg, nil
}

// DecodePayload unmarshals msg.Payload into a typed value.
func DecodePayload[T any](msg JobMessage) (T, error) {
	var out T
	if err := msg.validate(); err != nil {
		return out, err
	}
	if len(msg.Payload) == 0 {
		return out, errors.New("job message: payload is empty")
	}
	if err := json.Unmarshal(msg.Payload, &out); err != nil {
		return out, fmt.Errorf("unmarshal job payload: %w", err)
	}
	return out, nil
}
