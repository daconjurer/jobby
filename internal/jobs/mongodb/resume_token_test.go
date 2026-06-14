package mongodb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileResumeTokenStore_LoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	store := NewFileResumeTokenStore(path)

	token, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if token != nil {
		t.Fatalf("token=%v want nil", token)
	}
}

func TestFileResumeTokenStore_LoadEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewFileResumeTokenStore(path)

	token, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if token != nil {
		t.Fatalf("token=%v want nil", token)
	}
}

func TestFileResumeTokenStore_SaveNilIsNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	store := NewFileResumeTokenStore(path)

	if err := store.Save(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stat path: %v want not exist", err)
	}
}

func TestFileResumeTokenStore_SaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	store := NewFileResumeTokenStore(path)
	raw := []byte(`{"_data":"resume-test-token"}`)

	if err := store.Save(raw); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if string(loaded) != string(raw) {
		t.Fatalf("token=%q want %q", loaded, raw)
	}
}

func TestFileResumeTokenStore_SaveCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "token.json")
	store := NewFileResumeTokenStore(path)
	raw := []byte(`{"_data":"nested-token"}`)

	if err := store.Save(raw); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if string(loaded) != string(raw) {
		t.Fatalf("token=%q want %q", loaded, raw)
	}
}

func TestFileResumeTokenStore_SaveDoesNotLeaveTempFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	store := NewFileResumeTokenStore(path)

	if err := store.Save([]byte(`{"_data":"x"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temp file stat: %v want not exist", err)
	}
}

func TestFileResumeTokenStore_LoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewFileResumeTokenStore(path)

	_, err := store.Load()
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestNopResumeTokenStore_LoadSave(t *testing.T) {
	store := NopResumeTokenStore{}

	token, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if token != nil {
		t.Fatalf("token=%v want nil", token)
	}
	if err := store.Save([]byte(`{"_data":"ignored"}`)); err != nil {
		t.Fatal(err)
	}
}
