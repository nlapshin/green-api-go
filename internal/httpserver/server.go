package httpserver

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	"green-api-test/internal/handler"
)

type Deps struct {
	Handler   *handler.Handler
	Logger    *slog.Logger
	StaticDir string
}

type Server struct {
	h         *handler.Handler
	logger    *slog.Logger
	staticDir string
}

func New(d Deps) *Server {
	log := d.Logger
	if log == nil {
		log = slog.Default()
	}
	sd := d.StaticDir
	if sd == "" {
		sd = "web/static"
	}
	return &Server{h: d.Handler, logger: log, staticDir: sd}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RequestLogger(s.logger))
	r.Use(RecovererJSON(s.logger))

	r.Get("/healthz", s.h.Healthz)

	r.Get("/", s.h.Index)

	r.Mount("/swagger", httpSwagger.WrapHandler)

	r.Route("/api", func(r chi.Router) {
		r.Get("/get-settings", s.h.APIGetSettings)
		r.Get("/get-state-instance", s.h.APIGetStateInstance)
		r.Post("/send-message", s.h.APISendMessage)
		r.Post("/send-file-by-url", s.h.APISendFileByURL)
	})

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(s.staticDir))))

	return r
}
