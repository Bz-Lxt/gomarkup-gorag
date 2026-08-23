package wal

import (
	"encoding/binary"
	"hash/crc32"
	"io"
)

const (
	RecUpsert     byte = 1
	RecDelete     byte = 2
	RecCollection byte = 3
	RecFlush      byte = 4
)

type Record struct {
	Type    byte
	TS      int64
	Payload []byte
}

func encodeRecord(r Record) []byte {
	buf := make([]byte, 1+8+4+len(r.Payload)+4)
	buf[0] = r.Type
	binary.LittleEndian.PutUint64(buf[1:9], uint64(r.TS))
	binary.LittleEndian.PutUint32(buf[9:13], uint32(len(r.Payload)))
	copy(buf[13:13+len(r.Payload)], r.Payload)
	crc := crc32.ChecksumIEEE(buf[:13+len(r.Payload)])
	binary.LittleEndian.PutUint32(buf[13+len(r.Payload):], crc)
	return buf
}

func decodeRecord(r io.Reader) (Record, error) {
	hdr := make([]byte, 13)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return Record{}, err
	}
	rec := Record{
		Type: hdr[0],
		TS:   int64(binary.LittleEndian.Uint64(hdr[1:9])),
	}
	n := binary.LittleEndian.Uint32(hdr[9:13])
	if n > 32*1024*1024 {
		return Record{}, io.ErrUnexpectedEOF
	}
	rec.Payload = make([]byte, n)
	if n > 0 {
		if _, err := io.ReadFull(r, rec.Payload); err != nil {
			return Record{}, err
		}
	}
	crcBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, crcBuf); err != nil {
		return Record{}, err
	}
	got := binary.LittleEndian.Uint32(crcBuf)
	all := append(hdr, rec.Payload...)
	want := crc32.ChecksumIEEE(all)
	if got != want {
		return Record{}, ErrCorrupt
	}
	return rec, nil
}
