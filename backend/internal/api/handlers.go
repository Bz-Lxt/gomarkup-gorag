package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xavskye/gorag/internal/engine"
	imagefeat "github.com/xavskye/gorag/internal/feature/image"
	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/internal/rag"
	"github.com/xavskye/gorag/pkg/logger"
)

func (h *Handlers) ListCollections(w http.ResponseWriter, r *http.Request) {
	WriteOK(w, r, h.Eng.ListCollections())
}

func (h *Handlers) CreateCollection(w http.ResponseWriter, r *http.Request) {
	var c model.Collection
	if err := decodeJSON(r, &c); err != nil {
		WriteError(w, r, err)
		return
	}
	if err := h.Eng.CreateCollection(c); err != nil {
		WriteError(w, r, err)
		return
	}
	WriteCreated(w, r, c)
}

func (h *Handlers) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	if err := h.Eng.DeleteCollection(r.PathValue("name")); err != nil {
		WriteError(w, r, fmt.Errorf("delete collection: %w", err))
		return
	}
	WriteOK(w, r, map[string]bool{"deleted": true})
}

func (h *Handlers) IngestDocument(w http.ResponseWriter, r *http.Request) {
	var req engine.IngestDocReq
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, r, err)
		return
	}
	resp, err := h.Eng.IngestDocument(req)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteCreated(w, r, resp)
}

func (h *Handlers) IngestImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.Cfg.MaxUploadBytes + 1<<20); err != nil {
		WriteError(w, r, model.Wrap(model.CodeUploadInvalid, "multipart", err))
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		WriteError(w, r, model.NewError(model.CodeUploadInvalid, "file field required"))
		return
	}
	defer file.Close()
	resp, err := h.Eng.IngestImage(engine.IngestImageReq{
		Collection: r.FormValue("collection"),
		Caption:    r.FormValue("caption"),
		Tags:       splitCSV(r.FormValue("tags")),
		SourceRef:  r.FormValue("source_ref"),
		Reader:     file,
	})
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteCreated(w, r, resp)
}

func (h *Handlers) SearchText(w http.ResponseWriter, r *http.Request) {
	var req model.SearchRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		WriteError(w, r, model.NewError(model.CodeValidation, "query required"))
		return
	}
	resp, err := h.Eng.SearchText(req)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteOK(w, r, resp)
}

func (h *Handlers) SearchImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.Cfg.MaxUploadBytes + 1<<20); err != nil {
		WriteError(w, r, model.Wrap(model.CodeUploadInvalid, "multipart", err))
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		WriteError(w, r, model.NewError(model.CodeUploadInvalid, "file field required"))
		return
	}
	defer file.Close()
	feat, err := imagefeat.Extract(file, imagefeat.Options{
		Dim: model.DefaultDim, Grid: h.Cfg.PatchGrid,
		MaxBytes: h.Cfg.MaxUploadBytes, MaxPixels: h.Cfg.MaxImagePixels,
	})
	if err != nil {
		WriteError(w, r, err)
		return
	}
	req := model.SearchRequest{
		Collection:  r.FormValue("collection"),
		TopK:        atoi(r.FormValue("top_k")),
		Metric:      model.Metric(r.FormValue("metric")),
		IndexType:   model.IndexType(r.FormValue("index_type")),
		CompareFLAT: r.FormValue("compare_flat") == "1",
		Modality:    model.ModalityImage,
	}
	resp, err := h.Eng.SearchImage(feat.Vector, req)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteOK(w, r, resp)
}

func (h *Handlers) SearchHybrid(w http.ResponseWriter, r *http.Request) {
	var req model.SearchRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, r, err)
		return
	}
	resp, err := h.Eng.SearchHybrid(req, nil)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteOK(w, r, resp)
}

func (h *Handlers) RAG(w http.ResponseWriter, r *http.Request) {
	var q rag.Query
	if err := decodeJSON(r, &q); err != nil {
		WriteError(w, r, err)
		return
	}
	ch, meta, err := rag.Run(r.Context(), h.Eng, q)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)
	mb, _ := json.Marshal(meta)
	_, _ = w.Write([]byte("event: meta\ndata: " + string(mb) + "\n\n"))
	if flusher != nil {
		flusher.Flush()
	}
	for tok := range ch {
		if tok.Err != nil {
			eb, _ := json.Marshal(map[string]string{"error": tok.Err.Error()})
			_, _ = w.Write([]byte("event: error\ndata: " + string(eb) + "\n\n"))
			break
		}
		if tok.Text != "" {
			tb, _ := json.Marshal(map[string]any{"text": tok.Text, "mock": tok.Mock})
			_, _ = w.Write([]byte("event: token\ndata: " + string(tb) + "\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
		if tok.Done {
			_, _ = w.Write([]byte("event: done\ndata: {}\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

func (h *Handlers) EvalRecall(w http.ResponseWriter, r *http.Request) {
	n := atoi(r.URL.Query().Get("n"))
	q := atoi(r.URL.Query().Get("queries"))
	k := atoi(r.URL.Query().Get("k"))
	res, err := h.Eng.EvalRecall(n, q, k)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	WriteOK(w, r, res)
}

func (h *Handlers) Stats(w http.ResponseWriter, r *http.Request) {
	WriteOK(w, r, h.Eng.Stats())
}

func (h *Handlers) Cost(w http.ResponseWriter, r *http.Request) {
	WriteOK(w, r, map[string]any{
		"spent_cny":  h.Eng.Ledger.Spent(),
		"budget_cny": h.Eng.Ledger.Limit(),
		"records":    h.Eng.Ledger.List(),
	})
}

func (h *Handlers) Flush(w http.ResponseWriter, r *http.Request) {
	if err := h.Eng.Flush("manual"); err != nil {
		WriteError(w, r, err)
		return
	}
	WriteOK(w, r, map[string]string{"status": "flushing"})
}

func (h *Handlers) Compact(w http.ResponseWriter, r *http.Request) {
	if err := h.Eng.Compact(); err != nil {
		WriteError(w, r, err)
		return
	}
	WriteOK(w, r, map[string]string{"status": "compacted"})
}

func (h *Handlers) Meta(w http.ResponseWriter, r *http.Request) {
	WriteOK(w, r, map[string]any{
		"providers": map[string]string{
			"embedding": h.Eng.Embed.Kind(),
			"vision":    h.Eng.CLIP.Name(),
			"llm":       h.Eng.LLM.Kind(),
		},
		"cross_modal": h.Eng.CLIP.Enabled(),
		"estimate_rag_cny": func() float64 {
			if h.Eng.LLM.Kind() == "mock" {
				return 0
			}
			return 0.02
		}(),
		"dim": model.DefaultDim,
	})
}

func (h *Handlers) Asset(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	// allow hash.ext
	hash = strings.TrimSuffix(hash, filepath.Ext(hash))
	p, err := h.Eng.Assets.Path(hash)
	if err != nil {
		WriteError(w, r, model.NewError(model.CodeNotFound, "asset not found"))
		return
	}
	http.ServeFile(w, r, p)
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func init() { logger.Debug("api handlers registered") }
