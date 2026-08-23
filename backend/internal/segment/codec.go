package segment

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"

	"github.com/xavskye/gorag/internal/model"
)

const (
	segMagic   = "GORG"
	segVersion = uint16(1)
)

type FileHeader struct {
	ID        uint64
	RowCount  uint32
	ByteSize  uint32
	IndexType model.IndexType
	MinTS     int64
	MaxTS     int64
}

type wireEntity struct {
	E model.Entity
}

func WriteFile(path string, hdr FileHeader, ents []model.Entity) (uint32, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, fmt.Errorf("mkdir segment: %w", err)
	}
	var payload bytes.Buffer
	enc := gob.NewEncoder(&payload)
	if err := enc.Encode(hdr); err != nil {
		return 0, fmt.Errorf("encode header: %w", err)
	}
	if err := enc.Encode(ents); err != nil {
		return 0, fmt.Errorf("encode entities: %w", err)
	}
	raw := payload.Bytes()
	crc := crc32.ChecksumIEEE(raw)
	buf := make([]byte, 4+2+4+len(raw)+4)
	copy(buf[:4], segMagic)
	binary.LittleEndian.PutUint16(buf[4:6], segVersion)
	binary.LittleEndian.PutUint32(buf[6:10], uint32(len(raw)))
	copy(buf[10:10+len(raw)], raw)
	binary.LittleEndian.PutUint32(buf[10+len(raw):], crc)
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return 0, fmt.Errorf("write segment: %w", err)
	}
	return crc, nil
}

func ReadFile(path string) (FileHeader, []model.Entity, uint32, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return FileHeader{}, nil, 0, fmt.Errorf("read segment: %w", err)
	}
	if len(buf) < 14 {
		return FileHeader{}, nil, 0, fmt.Errorf("segment too short")
	}
	if string(buf[:4]) != segMagic {
		return FileHeader{}, nil, 0, fmt.Errorf("bad segment magic")
	}
	ver := binary.LittleEndian.Uint16(buf[4:6])
	if ver != segVersion {
		return FileHeader{}, nil, 0, fmt.Errorf("unsupported segment version %d", ver)
	}
	n := binary.LittleEndian.Uint32(buf[6:10])
	if int(10+n+4) > len(buf) {
		return FileHeader{}, nil, 0, fmt.Errorf("segment truncated")
	}
	raw := buf[10 : 10+n]
	crc := binary.LittleEndian.Uint32(buf[10+n:])
	if crc32.ChecksumIEEE(raw) != crc {
		return FileHeader{}, nil, 0, fmt.Errorf("segment crc mismatch")
	}
	dec := gob.NewDecoder(bytes.NewReader(raw))
	var hdr FileHeader
	if err := dec.Decode(&hdr); err != nil {
		return FileHeader{}, nil, 0, fmt.Errorf("decode header: %w", err)
	}
	var ents []model.Entity
	if err := dec.Decode(&ents); err != nil {
		return FileHeader{}, nil, 0, fmt.Errorf("decode entities: %w", err)
	}
	return hdr, ents, crc, nil
}
