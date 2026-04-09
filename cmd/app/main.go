// Package main starts the HTTP server for the Green-API proxy facade.
//
// @title						Green-API HTTP proxy
// @version					1.0
// @description				Minimal HTTP API that forwards selected Green-API calls. Instance credentials are accepted via `X-Instance-Id` and `X-Api-Token` headers (and may be merged from JSON on POST bodies). Success responses wrap upstream JSON as pretty-printed text in `pretty`.
// @BasePath					/
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/prometheus/client_golang/prometheus"

	"green-api-test/internal/config"
	"green-api-test/internal/greenapi"
	"green-api-test/internal/handler"
	"green-api-test/internal/httpserver"
	"green-api-test/internal/metrics"

	_ "green-api-test/docs" // OpenAPI (swag)
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if err := cfg.EnsureIndexTemplate(); err != nil {
		log.Error("startup check failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	metrics.Register(prometheus.DefaultRegisterer)

	ga, err := greenapi.NewClient(greenapi.Config{
		BaseURL: cfg.GreenAPIBaseURL,
		Timeout: cfg.GreenAPITimeout,
		Logger:  log,
		RequestIDFromContext: func(ctx context.Context) string {
			return middleware.GetReqID(ctx)
		},
	})
	if err != nil {
		log.Error("greenapi client init failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	h, err := handler.New(handler.Deps{
		Proxy:        ga,
		Logger:       log,
		TemplatePath: cfg.IndexTemplatePath(),
	})
	if err != nil {
		log.Error("handler init failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	srv := httpserver.New(httpserver.Deps{
		Handler:   h,
		Logger:    log,
		StaticDir: cfg.StaticDir(),
	})

	httpSrv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second, // limit request-line + headers phase (slowloris)
		// Full request body read is bounded per handler (MaxBytesReader); ReadTimeout caps total read time.
		ReadTimeout: 30 * time.Second,
		// Response write time; keep above upstream timeout so we can still serialize errors.
		WriteTimeout: 45 * time.Second,
		// IdleTimeout recycles keep-alive connections; slightly above typical client reuse interval.
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MiB — same order as body cap; enough for cookies + custom headers
	}

	go func() {
		log.Info("listening", slog.String("addr", httpSrv.Addr))
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Warn("shutdown", slog.String("error", err.Error()))
	}
}
