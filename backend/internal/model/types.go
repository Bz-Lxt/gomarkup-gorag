package model

import "time"

const DefaultDim = 1024

type Modality string

const (
	ModalityText  Modality = "text"
	ModalityImage Modality = "image"
	ModalityAudio Modality = "audio" // reserved, not implemented
	ModalityVideo Modality = "video" // reserved, not implemented
)

func (m Modality) Valid() bool {
	return m == ModalityText || m == ModalityImage
}

type Metric string

const (
	MetricCosine Metric = "cosine"
	MetricL2     Metric = "l2"
)

func (m Metric) Valid() bool {
	return m == MetricCosine || m == MetricL2
}

type IndexType string

const (
	IndexHNSW IndexType = "hnsw"
	IndexFLAT IndexType = "flat"
)

func (t IndexType) Valid() bool {
	return t == IndexHNSW || t == IndexFLAT
}

type SegmentState string

const (
	SegGrowing    SegmentState = "growing"
	SegSealed     SegmentState = "sealed"
	SegPersisted  SegmentState = "persisted"
	SegCompacted  SegmentState = "compacted"
)

type EntityID uint64

type Collection struct {
	Name      string    `json:"name"`
	Dim       int       `json:"dim"`
	Metric    Metric    `json:"metric"`
	IndexType IndexType `json:"index_type"`
	CreatedAt time.Time `json:"created_at"`
}

type TermPos struct {
	Term  string `json:"term"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type Sentence struct {
	Start  int       `json:"start"`
	End    int       `json:"end"`
	Vector []float32 `json:"-"`
}

type Patch struct {
	GridRow int       `json:"grid_row"`
	GridCol int       `json:"grid_col"`
	BBox    [4]float64 `json:"bbox"` // x,y,w,h normalized
	Vector  []float32 `json:"-"`
}

// Entity 是统一的图文记录。向量始终 L2 归一化、长度为 Dim。
type Entity struct {
	ID         EntityID       `json:"id"`
	Collection string         `json:"collection"`
	Modality   Modality       `json:"modality"`
	Vector     []float32      `json:"-"`
	Scalar     map[string]any `json:"scalar,omitempty"`
	SourceRef  string         `json:"source_ref,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	Deleted    bool           `json:"deleted,omitempty"`

	// text
	Content    string     `json:"content,omitempty"`
	DocID      string     `json:"doc_id,omitempty"`
	ChunkIndex int        `json:"chunk_index,omitempty"`
	CharOffset int        `json:"char_offset,omitempty"`
	Terms      []TermPos  `json:"terms,omitempty"`
	Sentences  []Sentence `json:"-"`

	// image
	ContentHash string   `json:"content_hash,omitempty"`
	Width       int      `json:"width,omitempty"`
	Height      int      `json:"height,omitempty"`
	Caption     string   `json:"caption,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Patches     []Patch  `json:"patches,omitempty"`
	MIME        string   `json:"mime,omitempty"`
}

type ChannelScores struct {
	Vector  int     `json:"vector"`
	Keyword int     `json:"keyword"`
	RRF     float64 `json:"rrf"`
}

type BBoxEvidence struct {
	Box   [4]float64 `json:"box"`
	Score float64    `json:"score"`
}

type CharRange struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Kind  string `json:"kind"`
}

type Evidence struct {
	BBox       []BBoxEvidence `json:"bbox"`
	CharRanges []CharRange    `json:"char_ranges"`
}

type SearchHit struct {
	ID         EntityID      `json:"id"`
	Score      float64       `json:"score"`
	Modality   Modality      `json:"modality"`
	Channels   ChannelScores `json:"channels"`
	CrossModal bool          `json:"cross_modal"`
	Evidence   Evidence      `json:"evidence"`
	Content    string        `json:"content,omitempty"`
	Caption    string        `json:"caption,omitempty"`
	AssetURL   string        `json:"asset_url,omitempty"`
	Tags       []string      `json:"tags,omitempty"`
	Collection string        `json:"collection"`
	SourceRef  string        `json:"source_ref,omitempty"`
	Title      string        `json:"title,omitempty"`
}

type SearchRequest struct {
	Collection  string            `json:"collection"`
	Query       string            `json:"query"`
	TopK        int               `json:"top_k"`
	Metric      Metric            `json:"metric"`
	IndexType   IndexType         `json:"index_type"`
	RRFK        int               `json:"rrf_k"`
	VectorW     float64           `json:"vector_weight"`
	KeywordW    float64           `json:"keyword_weight"`
	Filter      string            `json:"filter"`
	Modality    Modality          `json:"modality"`
	EfSearch    int               `json:"ef_search"`
	CompareFLAT bool              `json:"compare_flat"`
	ExtraScalar map[string]string `json:"-"`
}

type SearchResponse struct {
	Hits        []SearchHit `json:"hits"`
	FLATHits    []SearchHit `json:"flat_hits,omitempty"`
	RecallAtK   float64     `json:"recall_at_k,omitempty"`
	CrossModal  bool        `json:"cross_modal"`
	DegradeNote string      `json:"degrade_note,omitempty"`
	TookMS      int64       `json:"took_ms"`
	Channels    []string    `json:"channels"`
}

type SegmentInfo struct {
	ID        uint64       `json:"id"`
	State     SegmentState `json:"state"`
	RowCount  int          `json:"row_count"`
	ByteSize  int64        `json:"byte_size"`
	IndexType IndexType    `json:"index_type"`
	FilePath  string       `json:"file_path,omitempty"`
	CRC32     uint32       `json:"crc32,omitempty"`
	MinTS     int64        `json:"min_ts"`
	MaxTS     int64        `json:"max_ts"`
}

type Stats struct {
	Collections   int            `json:"collections"`
	Entities      int            `json:"entities"`
	Vectors       int            `json:"vectors"`
	Patches       int            `json:"patches"`
	Segments      []SegmentInfo  `json:"segments"`
	MemBytes      int64          `json:"mem_bytes"`
	WALBytes      int64          `json:"wal_bytes"`
	FlushHistory  []FlushEvent   `json:"flush_history"`
	QPS           float64        `json:"qps"`
	LatencyP50MS  float64        `json:"latency_p50_ms"`
	LatencyP99MS  float64        `json:"latency_p99_ms"`
	RecoverMS     int64          `json:"recover_ms"`
	CostCNY       float64        `json:"cost_cny"`
	BudgetCNY     float64        `json:"budget_cny"`
	Providers     map[string]string `json:"providers"`
	HNSWParams    map[string]int `json:"hnsw_params"`
}

type FlushEvent struct {
	At       time.Time `json:"at"`
	Segment  uint64    `json:"segment"`
	Rows     int       `json:"rows"`
	Bytes    int64     `json:"bytes"`
	Reason   string    `json:"reason"`
	Duration string    `json:"duration"`
}

type CostRecord struct {
	At       time.Time `json:"at"`
	Provider string    `json:"provider"`
	Model    string    `json:"model"`
	Tokens   int       `json:"tokens"`
	CNY      float64   `json:"cny"`
	OK       bool      `json:"ok"`
	Reason   string    `json:"reason,omitempty"`
}
