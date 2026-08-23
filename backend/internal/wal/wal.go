package wal

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/xavskye/gorag/pkg/timeutil"
)

var ErrCorrupt = errors.New("wal: corrupt record")

const magic = "GRWL"
const version uint16 = 1

type Log struct {
	path string
	f    *os.File
	mu   sync.Mutex
	once sync.Once
}

func Open(dir string) (*Log, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("wal mkdir: %w", err)
	}
	path := filepath.Join(dir, "wal.bin")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("wal open: %w", err)
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if st.Size() == 0 {
		hdr := make([]byte, 6)
		copy(hdr[:4], magic)
		hdr[4] = byte(version)
		hdr[5] = byte(version >> 8)
		if _, err := f.Write(hdr); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("wal header: %w", err)
		}
		if err := f.Sync(); err != nil {
			_ = f.Close()
			return nil, err
		}
	}
	return &Log{path: path, f: f}, nil
}

func (l *Log) Append(typ byte, payload []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return fmt.Errorf("wal closed")
	}
	raw := encodeRecord(Record{Type: typ, TS: timeutil.UnixMilli(), Payload: payload})
	if _, err := l.f.Write(raw); err != nil {
		return fmt.Errorf("wal write: %w", err)
	}
	if err := l.f.Sync(); err != nil {
		return fmt.Errorf("wal sync: %w", err)
	}
	return nil
}

func (l *Log) Replay() ([]Record, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	hdr := make([]byte, 6)
	if _, err := io.ReadFull(l.f, hdr); err != nil {
		return nil, fmt.Errorf("wal read header: %w", err)
	}
	if string(hdr[:4]) != magic {
		return nil, fmt.Errorf("wal bad magic")
	}
	ver := uint16(hdr[4]) | uint16(hdr[5])<<8
	if ver != version {
		return nil, fmt.Errorf("wal unsupported version %d", ver)
	}
	var recs []Record
	for {
		rec, err := decodeRecord(l.f)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return recs, fmt.Errorf("wal replay: %w", err)
		}
		recs = append(recs, rec)
	}
	if _, err := l.f.Seek(0, io.SeekEnd); err != nil {
		return recs, err
	}
	return recs, nil
}

func (l *Log) Size() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return 0
	}
	st, err := l.f.Stat()
	if err != nil {
		return 0
	}
	return st.Size()
}

func (l *Log) Truncate() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f == nil {
		return fmt.Errorf("wal closed")
	}
	if err := l.f.Truncate(6); err != nil {
		return err
	}
	if _, err := l.f.Seek(6, io.SeekStart); err != nil {
		return err
	}
	return l.f.Sync()
}

// Close 必须用 Once，测试里 defer + 显式 Close 不会 panic。
func (l *Log) Close() error {
	var err error
	l.once.Do(func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if l.f != nil {
			err = l.f.Close()
			l.f = nil
		}
	})
	return err
}
