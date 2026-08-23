package engine

import (
	"fmt"
	"io"
	"strings"
	"github.com/xavskye/gorag/internal/feature/image"
	"github.com/xavskye/gorag/internal/feature/text"
	"github.com/xavskye/gorag/internal/metric"
	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/internal/tokenize"
	"github.com/xavskye/gorag/pkg/timeutil"
)

type IngestDocReq struct {
	Collection string   `json:"collection"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
	SourceRef  string   `json:"source_ref"`
}

type IngestDocResp struct {
	DocID     string           `json:"doc_id"`
	Chunks    int              `json:"chunks"`
	EntityIDs []model.EntityID `json:"entity_ids"`
}

func (e *Engine) IngestDocument(req IngestDocReq) (*IngestDocResp, error) {
	if strings.TrimSpace(req.Content) == "" {
		return nil, model.NewError(model.CodeValidation, "content required")
	}
	if req.Collection == "" {
		req.Collection = "default"
	}
	e.mu.RLock()
	if _, ok := e.cols[req.Collection]; !ok {
		e.mu.RUnlock()
		return nil, model.NewError(model.CodeNotFound, "collection not found")
	}
	e.mu.RUnlock()
	chunks := chunkText(req.Content, 480, 80)
	docID := fmt.Sprintf("doc-%d", timeutil.UnixMilli())
	ids := make([]model.EntityID, 0, len(chunks))
	offset := 0
	for i, ch := range chunks {
		vec, err := e.Embed.Embed(ch)
		if err != nil {
			return nil, err
		}
		if err := metric.ValidateDim(vec, model.DefaultDim); err != nil {
			return nil, model.Wrap(model.CodeDimMismatch, "embed dim", err)
		}
		toks := tokenize.Tokenize(ch)
		ent := model.Entity{
			ID:         model.EntityID(e.Man.AllocEntity()),
			Collection: req.Collection,
			Modality:   model.ModalityText,
			Vector:     vec,
			SourceRef:  req.SourceRef,
			CreatedAt:  timeutil.NowNaive(),
			Content:    ch,
			DocID:      docID,
			ChunkIndex: i,
			CharOffset: offset,
			Terms:      tokenize.ToTermPos(toks),
			Sentences:  text.EmbedSentences(ch, model.DefaultDim),
			Tags:       req.Tags,
			Scalar:     map[string]any{"title": req.Title, "tag": firstTag(req.Tags)},
		}
		e.mu.Lock()
		e.indexLocked(&ent, true)
		e.mu.Unlock()
		ids = append(ids, ent.ID)
		offset += len(ch)
	}
	_ = e.Man.Save()
	return &IngestDocResp{DocID: docID, Chunks: len(ids), EntityIDs: ids}, nil
}

type IngestImageReq struct {
	Collection string
	Caption    string
	Tags       []string
	SourceRef  string
	Reader     io.Reader
}

type IngestImageResp struct {
	EntityID    model.EntityID `json:"entity_id"`
	ContentHash string         `json:"content_hash"`
	Patches     int            `json:"patches"`
	Width       int            `json:"width"`
	Height      int            `json:"height"`
	AssetURL    string         `json:"asset_url"`
}

func (e *Engine) IngestImage(req IngestImageReq) (*IngestImageResp, error) {
	if req.Collection == "" {
		req.Collection = "default"
	}
	e.mu.RLock()
	if _, ok := e.cols[req.Collection]; !ok {
		e.mu.RUnlock()
		return nil, model.NewError(model.CodeNotFound, "collection not found")
	}
	e.mu.RUnlock()
	feat, err := imagefeat.Extract(req.Reader, imagefeat.Options{
		Dim: model.DefaultDim, Grid: e.Cfg.PatchGrid,
		MaxBytes: e.Cfg.MaxUploadBytes, MaxPixels: e.Cfg.MaxImagePixels,
	})
	if err != nil {
		return nil, err
	}
	name, err := e.Assets.Put(feat.ContentHash, feat.MIME, feat.Bytes)
	if err != nil {
		return nil, model.Wrap(model.CodeInternal, "store asset", err)
	}
	textBlob := strings.TrimSpace(req.Caption + " " + strings.Join(req.Tags, " "))
	ent := model.Entity{
		ID:          model.EntityID(e.Man.AllocEntity()),
		Collection:  req.Collection,
		Modality:    model.ModalityImage,
		Vector:      feat.Vector,
		SourceRef:   req.SourceRef,
		CreatedAt:   timeutil.NowNaive(),
		ContentHash: feat.ContentHash,
		Width:       feat.Width,
		Height:      feat.Height,
		Caption:     req.Caption,
		Tags:        req.Tags,
		Patches:     feat.Patches,
		MIME:        feat.MIME,
		Content:     textBlob,
		Terms:       tokenize.ToTermPos(tokenize.Tokenize(textBlob)),
		Scalar:      map[string]any{"tag": firstTag(req.Tags), "caption": req.Caption},
	}
	e.mu.Lock()
	e.indexLocked(&ent, true)
	e.mu.Unlock()
	_ = e.Man.Save()
	return &IngestImageResp{
		EntityID: ent.ID, ContentHash: feat.ContentHash, Patches: len(feat.Patches),
		Width: feat.Width, Height: feat.Height, AssetURL: "/api/v1/assets/" + name,
	}, nil
}

func chunkText(s string, size, overlap int) []string {
	if size < 64 {
		size = 64
	}
	rs := []rune(s)
	if len(rs) <= size {
		return []string{s}
	}
	var out []string
	for i := 0; i < len(rs); {
		j := i + size
		if j > len(rs) {
			j = len(rs)
		}
		out = append(out, string(rs[i:j]))
		if j == len(rs) {
			break
		}
		i = j - overlap
		if i < 0 {
			i = 0
		}
	}
	return out
}

func firstTag(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return tags[0]
}
