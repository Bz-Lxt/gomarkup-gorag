package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/pkg/logger"
	"github.com/xavskye/gorag/pkg/timeutil"
)

func withTrace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Trace-Id") == "" {
			r.Header.Set("X-Trace-Id", traceID(r))
		}
		w.Header().Set("X-Trace-Id", r.Header.Get("X-Trace-Id"))
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http", "method", r.Method, "path", r.URL.Path, "ms", time.Since(start).Milliseconds())
	})
}

func requireAuth(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !validToken(cfg, tok) {
			WriteError(w, r, model.NewError(model.CodeUnauthorized, "login required"))
			return
		}
		next(w, r)
	}
}

func issueToken(cfg *config.Config) string {
	mac := hmac.New(sha256.New, []byte(cfg.DemoPass+"|"+cfg.DemoUser))
	_, _ = mac.Write([]byte("gorag-session"))
	return hex.EncodeToString(mac.Sum(nil))
}

func validToken(cfg *config.Config, tok string) bool {
	return tok != "" && hmac.Equal([]byte(tok), []byte(issueToken(cfg)))
}

func loginHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := decodeJSON(r, &req); err != nil {
			WriteError(w, r, err)
			return
		}
		if req.Username != cfg.DemoUser || req.Password != cfg.DemoPass {
			WriteError(w, r, model.NewError(model.CodeUnauthorized, "invalid credentials"))
			return
		}
		WriteOK(w, r, map[string]any{
			"token":      issueToken(cfg),
			"username":   cfg.DemoUser,
			"issued_at":  timeutil.Format(timeutil.Now()),
			"llm_kind":   cfg.LLMProvider,
			"embed_kind": cfg.EmbeddingProvider,
		})
	}
}
