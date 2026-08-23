package api

import (
	"net/http"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
	"github.com/xavskye/gorag/internal/model"
)

func NewRouter(cfg *config.Config, eng *engine.Engine) http.Handler {
	mux := http.NewServeMux()
	h := &Handlers{Cfg: cfg, Eng: eng}

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		WriteOK(w, r, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !eng.Ready() {
			WriteError(w, r, model.NewError(model.CodeInternal, "not ready"))
			return
		}
		WriteOK(w, r, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("POST /api/v1/auth/login", loginHandler(cfg))
	mux.HandleFunc("GET /api/v1/assets/{hash}", h.Asset)

	mux.HandleFunc("GET /api/v1/collections", requireAuth(cfg, h.ListCollections))
	mux.HandleFunc("POST /api/v1/collections", requireAuth(cfg, h.CreateCollection))
	mux.HandleFunc("DELETE /api/v1/collections/{name}", requireAuth(cfg, h.DeleteCollection))

	mux.HandleFunc("POST /api/v1/documents", requireAuth(cfg, h.IngestDocument))
	mux.HandleFunc("POST /api/v1/images", requireAuth(cfg, h.IngestImage))

	mux.HandleFunc("POST /api/v1/search/text", requireAuth(cfg, h.SearchText))
	mux.HandleFunc("POST /api/v1/search/image", requireAuth(cfg, h.SearchImage))
	mux.HandleFunc("POST /api/v1/search/hybrid", requireAuth(cfg, h.SearchHybrid))
	mux.HandleFunc("POST /api/v1/rag/query", requireAuth(cfg, h.RAG))
	mux.HandleFunc("GET /api/v1/eval/recall", requireAuth(cfg, h.EvalRecall))
	mux.HandleFunc("GET /api/v1/stats", requireAuth(cfg, h.Stats))
	mux.HandleFunc("GET /api/v1/cost", requireAuth(cfg, h.Cost))
	mux.HandleFunc("POST /api/v1/admin/flush", requireAuth(cfg, h.Flush))
	mux.HandleFunc("POST /api/v1/admin/compact", requireAuth(cfg, h.Compact))
	mux.HandleFunc("GET /api/v1/meta", requireAuth(cfg, h.Meta))

	return withTrace(mux)
}

type Handlers struct {
	Cfg *config.Config
	Eng *engine.Engine
}
