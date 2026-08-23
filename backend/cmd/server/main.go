package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xavskye/gorag/internal/api"
	"github.com/xavskye/gorag/internal/config"
	"github.com/xavskye/gorag/internal/engine"
	"github.com/xavskye/gorag/pkg/logger"
	"github.com/xavskye/gorag/pkg/timeutil"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logger.Error("config", "err", err)
		os.Exit(1)
	}
	logger.Init(cfg.LogLevel, cfg.LogEnv)
	logger.Info("boot", "addr", cfg.HTTPAddr, "data", cfg.DataDir, "tz", timeutil.Format(timeutil.Now()))

	eng, err := engine.Open(cfg)
	if err != nil {
		logger.Error("engine.open", "err", err)
		os.Exit(1)
	}
	eng.SeedDemo()
	defer eng.Close()

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewRouter(cfg, eng),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
	}
	go func() {
		logger.Info("http.listen", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http.serve", "err", err)
			os.Exit(1)
		}
	}()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	logger.Info("shutdown")
}
