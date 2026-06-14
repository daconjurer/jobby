package mongodb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ResumeTokenStore persists MongoDB change stream resume tokens across restarts.
type ResumeTokenStore interface {
	Load() (bson.Raw, error)
	Save(token bson.Raw) error
}

// FileResumeTokenStore reads and writes a change stream resume token as JSON at path.
type FileResumeTokenStore struct {
	path string
}

// NewFileResumeTokenStore returns a file-backed change stream resume token store.
func NewFileResumeTokenStore(path string) *FileResumeTokenStore {
	return &FileResumeTokenStore{path: path}
}

// Load returns the saved token or nil when none exists.
func (s *FileResumeTokenStore) Load() (bson.Raw, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read mongodb resume token: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	var token bson.Raw
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("decode mongodb resume token: %w", err)
	}
	return token, nil
}

// Save persists token atomically to disk.
func (s *FileResumeTokenStore) Save(token bson.Raw) error {
	if token == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create mongodb resume token dir: %w", err)
	}
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("encode mongodb resume token: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write mongodb resume token temp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename mongodb resume token: %w", err)
	}
	return nil
}

// NopResumeTokenStore discards change stream resume tokens (tests / ephemeral dev).
type NopResumeTokenStore struct{}

func (NopResumeTokenStore) Load() (bson.Raw, error) { return nil, nil }
func (NopResumeTokenStore) Save(bson.Raw) error     { return nil }
