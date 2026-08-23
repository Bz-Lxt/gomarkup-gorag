package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Assets struct {
	dir string
}

func NewAssets(dir string) (*Assets, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("assets mkdir: %w", err)
	}
	return &Assets{dir: dir}, nil
}

func (a *Assets) Put(hash string, mime string, raw []byte) (string, error) {
	if err := safeHash(hash); err != nil {
		return "", err
	}
	ext := extOf(mime)
	name := hash + ext
	path := filepath.Join(a.dir, name)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", fmt.Errorf("write asset: %w", err)
	}
	return name, nil
}

func (a *Assets) Path(hash string) (string, error) {
	if err := safeHash(hash); err != nil {
		return "", err
	}
	matches, err := filepath.Glob(filepath.Join(a.dir, hash+".*"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		// try exact
		p := filepath.Join(a.dir, hash)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		return "", modelNotFound()
	}
	return matches[0], nil
}

func modelNotFound() error {
	return fmt.Errorf("asset not found")
}

func safeHash(h string) error {
	if len(h) < 16 || strings.Contains(h, "/") || strings.Contains(h, "..") {
		return fmt.Errorf("invalid content hash")
	}
	for _, r := range h {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return fmt.Errorf("invalid content hash")
		}
	}
	return nil
}

func extOf(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}
