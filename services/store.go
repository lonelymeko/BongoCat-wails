package services

import (
	"os"
	"path/filepath"
)

// StoreService replaces @tauri-store/pinia. Each pinia store persists its state
// as a JSON document under <dataDir>/stores/<id>.json. The frontend bridge owns
// serialisation, key filtering and debouncing; this service is pure storage.
type StoreService struct {
	dir string
}

func NewStoreService(dataDir string) *StoreService {
	return &StoreService{dir: filepath.Join(dataDir, "stores")}
}

func (s *StoreService) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

// Load returns the persisted JSON for a store id, or "" if none exists yet.
func (s *StoreService) Load(id string) (string, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// Save writes the JSON document for a store id, creating the directory if
// needed.
func (s *StoreService) Save(id, json string) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path(id), []byte(json), 0o644)
}
