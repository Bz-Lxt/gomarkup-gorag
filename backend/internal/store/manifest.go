package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/pkg/timeutil"
)

type Manifest struct {
	path        string
	mu          sync.Mutex
	Collections []model.Collection `json:"collections"`
	Segments    []model.SegmentInfo `json:"segments"`
	NextEntity  uint64             `json:"next_entity"`
	NextSegment uint64             `json:"next_segment"`
	UpdatedAt   string             `json:"updated_at"`
}

func LoadManifest(dir string) (*Manifest, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("manifest mkdir: %w", err)
	}
	path := filepath.Join(dir, "manifest.json")
	m := &Manifest{path: path, NextEntity: 1, NextSegment: 1}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return m, m.Save()
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	if err := json.Unmarshal(b, m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	m.path = path
	if m.NextEntity == 0 {
		m.NextEntity = 1
	}
	if m.NextSegment == 0 {
		m.NextSegment = 1
	}
	return m, nil
}

func (m *Manifest) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdatedAt = timeutil.Format(timeutil.Now())
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("write manifest tmp: %w", err)
	}
	return os.Rename(tmp, m.path)
}

func (m *Manifest) AllocEntity() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.NextEntity
	m.NextEntity++
	return id
}

func (m *Manifest) AllocSegment() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.NextSegment
	m.NextSegment++
	return id
}
