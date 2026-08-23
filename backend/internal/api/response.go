package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/xavskye/gorag/internal/model"
	"github.com/xavskye/gorag/pkg/logger"
)

type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	TraceID string `json:"trace_id"`
}

func WriteOK(w http.ResponseWriter, r *http.Request, data any) {
	write(w, r, http.StatusOK, Envelope{Code: model.CodeOK, Message: "ok", Data: data})
}

func WriteCreated(w http.ResponseWriter, r *http.Request, data any) {
	write(w, r, http.StatusCreated, Envelope{Code: model.CodeOK, Message: "created", Data: data})
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	code := model.CodeInternal
	msg := "internal error"
	httpStatus := http.StatusInternalServerError
	var me *model.Error
	if errors.As(err, &me) {
		code = me.Code
		msg = me.Message
		switch {
		case code == model.CodeUnauthorized:
			httpStatus = http.StatusUnauthorized
		case code == model.CodeNotFound:
			httpStatus = http.StatusNotFound
		case code == model.CodeConflict:
			httpStatus = http.StatusConflict
		case code >= 40000 && code < 50000:
			httpStatus = http.StatusBadRequest
		default:
			httpStatus = http.StatusInternalServerError
		}
	} else if err != nil {
		msg = err.Error()
	}
	// Log the full error string (including wrapping context) so operation
	// context like "delete collection: collection not found" is preserved.
	logger.Warn("api.error", "code", code, "msg", msg, "err", err, "path", r.URL.Path)
	write(w, r, httpStatus, Envelope{Code: code, Message: msg})
}

func write(w http.ResponseWriter, r *http.Request, status int, env Envelope) {
	if env.TraceID == "" {
		env.TraceID = traceID(r)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}

func traceID(r *http.Request) string {
	if v := r.Header.Get("X-Trace-Id"); v != "" {
		return v
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func decodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return model.NewError(model.CodeBadRequest, "empty body")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return model.Wrap(model.CodeBadRequest, "invalid json", err)
	}
	return nil
}
