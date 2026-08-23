package segment

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/pkg/logger"
)

// CompactSmall 合并已落盘的小 Segment，清理 tombstone。
func CompactSmall(dir string, segs []model.SegmentInfo, live func(model.Entity) bool, newID uint64) (model.SegmentInfo, []model.Entity, error) {
	var merged []model.Entity
	var ids []uint64
	var bytes int64
	var minTS, maxTS int64
	var readErr error
	for _, s := range segs {
		if s.FilePath == "" {
			continue
		}
		_, ents, _, err := ReadFile(s.FilePath)
		if err != nil {
			if readErr == nil {
				readErr = err
			}
			continue
		}
		for _, e := range ents {
			if e.Deleted || (live != nil && !live(e)) {
				continue
			}
			merged = append(merged, e)
		}
		bytes += s.ByteSize
		ids = append(ids, s.ID)
		if minTS == 0 || s.MinTS < minTS {
			minTS = s.MinTS
		}
		if s.MaxTS > maxTS {
			maxTS = s.MaxTS
		}
	}
	if len(merged) == 0 {
		if readErr != nil {
			return model.SegmentInfo{}, nil, readErr
		}
		return model.SegmentInfo{}, nil, fmt.Errorf("nothing to compact")
	}
	path := filepath.Join(dir, fmt.Sprintf("seg_%d.bin", newID))
	info := model.SegmentInfo{
		ID: newID, State: model.SegPersisted, RowCount: len(merged), ByteSize: bytes,
		IndexType: model.IndexHNSW, FilePath: path, MinTS: minTS, MaxTS: maxTS,
	}
	crc, err := WriteFile(path, FileHeader{
		ID: newID, RowCount: uint32(len(merged)), ByteSize: uint32(bytes),
		IndexType: model.IndexHNSW, MinTS: minTS, MaxTS: maxTS,
	}, merged)
	if err != nil {
		return model.SegmentInfo{}, nil, err
	}
	info.CRC32 = crc
	info.State = model.SegCompacted
	for _, s := range segs {
		if s.FilePath != "" && s.FilePath != path {
			_ = os.Remove(s.FilePath)
		}
	}
	logger.Info("segment.compacted", "from", ids, "to", newID, "rows", len(merged))
	return info, merged, nil
}
