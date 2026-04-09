package handler

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"

	"green-api-test/internal/domain"
	"green-api-test/internal/greenapi"
	"green-api-test/internal/httpx"
)

type Proxy interface {
	GetSettings(ctx context.Context, cred domain.ConnectRequest) ([]byte, error)
	GetStateInstance(ctx context.Context, cred domain.ConnectRequest) ([]byte, error)
	SendMessage(ctx context.Context, cred domain.ConnectRequest, msg domain.OutboundTextMessage) ([]byte, error)
	SendFileByURL(ctx context.Context, cred domain.ConnectRequest, file domain.OutboundFileMessage) ([]byte, error)
}

type Deps struct {
	Proxy        Proxy
	Logger       *slog.Logger
	TemplatePath string
}

type Handler struct {
	proxy  Proxy
	logger *slog.Logger
	tmpl   *template.Template
}

func New(d Deps) (*Handler, error) {
	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}
	if d.Proxy == nil {
		return nil, fmt.Errorf("handler: Proxy is required")
	}
	tplPath := d.TemplatePath
	if tplPath == "" {
		tplPath = filepath.Join("web", "templates", "index.html")
	}
	tmpl, err := template.ParseFiles(tplPath)
	if err != nil {
		return nil, fmt.Errorf("handler: parse template %q: %w", tplPath, err)
	}
	return &Handler{
		proxy:  d.Proxy,
		logger: logger,
		tmpl:   tmpl,
	}, nil
}

// Livez answers whether the process should be restarted (liveness).
//
//	@Summary		Liveness probe
//	@Description	Returns plain text `ok`.
//	@Tags			system
//	@Produce		plain
//	@Success		200	{string}	string	"ok"
//	@Router			/livez [get]
func (h *Handler) Livez(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Readyz answers whether the process should receive traffic (readiness). Extend with dependency checks when the service grows.
//
//	@Summary		Readiness probe
//	@Description	Returns plain text `ok` when startup wiring completed successfully.
//	@Tags			system
//	@Produce		plain
//	@Success		200	{string}	string	"ok"
//	@Router			/readyz [get]
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	h.Livez(w, r)
}

// Index serves the bundled HTML demo page (SSR template).
//
//	@Summary		Demo UI
//	@Description	Simple browser UI for trying the proxy (not required for API clients).
//	@Tags			ui
//	@Produce		html
//	@Success		200	{string}	string	"HTML page"
//	@Router			/ [get]
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.Execute(w, nil)
}

// APIGetSettings proxies Green-API `getSettings` (waInstanceSettings).
//
//	@Summary		Get instance settings
//	@Description	Forwards to Green-API using `idInstance` / `apiTokenInstance` from headers.
//	@Tags			api
//	@Produce		json
//	@Param			X-Instance-Id	header		string	true	"Green-API idInstance (digits)"
//	@Param			X-Api-Token		header		string	true	"Green-API apiTokenInstance"
//	@Success		200	{object}	domain.APIResponse	"ok=true, pretty=formatted upstream JSON"
//	@Failure		400	{object}	domain.APIResponse	"validation_error or invalid_json"
//	@Failure		502	{object}	domain.APIResponse	"upstream integration error"
//	@Router			/api/v1/get-settings [get]
func (h *Handler) APIGetSettings(w http.ResponseWriter, r *http.Request) {
	h.withConnect(w, r, h.proxy.GetSettings)
}

// APIGetStateInstance proxies Green-API `getStateInstance`.
//
//	@Summary		Get instance state
//	@Description	Returns authorization / instance state from Green-API.
//	@Tags			api
//	@Produce		json
//	@Param			X-Instance-Id	header		string	true	"Green-API idInstance (digits)"
//	@Param			X-Api-Token		header		string	true	"Green-API apiTokenInstance"
//	@Success		200	{object}	domain.APIResponse
//	@Failure		400	{object}	domain.APIResponse
//	@Failure		502	{object}	domain.APIResponse
//	@Router			/api/v1/get-state-instance [get]
func (h *Handler) APIGetStateInstance(w http.ResponseWriter, r *http.Request) {
	h.withConnect(w, r, h.proxy.GetStateInstance)
}

// APISendMessage proxies Green-API outbound text message send.
//
//	@Summary		Send text message
//	@Description	JSON body must include chatId and message. Credentials in headers override body fields when both are present.
//	@Tags			api
//	@Accept			json
//	@Produce		json
//	@Param			X-Instance-Id	header		string					false	"Green-API idInstance (optional if present in body)"
//	@Param			X-Api-Token		header		string					false	"Green-API apiTokenInstance (optional if present in body)"
//	@Param			body			body		domain.SendMessageRequest	true	"Payload"
//	@Success		200	{object}	domain.APIResponse
//	@Failure		400	{object}	domain.APIResponse
//	@Failure		502	{object}	domain.APIResponse
//	@Router			/api/v1/send-message [post]
func (h *Handler) APISendMessage(w http.ResponseWriter, r *http.Request) {
	var req domain.SendMessageRequest
	if err := httpx.DecodeStrictJSON(w, r, &req, domain.MaxJSONRequestBody); err != nil {
		writeDecodeError(w, r, err)
		return
	}
	httpx.MergeConnectHeaders(r, &req.ConnectRequest)
	req.Normalize()
	if err := req.Validate(); err != nil {
		writeValidationError(w, r, err)
		return
	}

	raw, err := h.proxy.SendMessage(r.Context(), req.ConnectRequest, domain.OutboundTextMessage{
		ChatID:  req.ChatID,
		Message: req.Message,
	})
	h.finishProxy(w, r, raw, err)
}

// APISendFileByURL proxies Green-API file send by URL.
//
//	@Summary		Send file by URL
//	@Description	JSON body: chatId, fileUrl, fileName; optional caption. Headers override embedded credentials.
//	@Tags			api
//	@Accept			json
//	@Produce		json
//	@Param			X-Instance-Id	header		string						false	"Green-API idInstance"
//	@Param			X-Api-Token		header		string						false	"Green-API apiTokenInstance"
//	@Param			body			body		domain.SendFileByURLRequest	true	"Payload"
//	@Success		200	{object}	domain.APIResponse
//	@Failure		400	{object}	domain.APIResponse
//	@Failure		502	{object}	domain.APIResponse
//	@Router			/api/v1/send-file-by-url [post]
func (h *Handler) APISendFileByURL(w http.ResponseWriter, r *http.Request) {
	var req domain.SendFileByURLRequest
	if err := httpx.DecodeStrictJSON(w, r, &req, domain.MaxJSONRequestBody); err != nil {
		writeDecodeError(w, r, err)
		return
	}
	httpx.MergeConnectHeaders(r, &req.ConnectRequest)
	req.Normalize()
	if err := req.Validate(); err != nil {
		writeValidationError(w, r, err)
		return
	}

	raw, err := h.proxy.SendFileByURL(r.Context(), req.ConnectRequest, domain.OutboundFileMessage{
		ChatID:   req.ChatID,
		URLFile:  req.FileURL,
		FileName: req.FileName,
		Caption:  req.Caption,
	})
	h.finishProxy(w, r, raw, err)
}

func (h *Handler) withConnect(w http.ResponseWriter, r *http.Request, fn func(context.Context, domain.ConnectRequest) ([]byte, error)) {
	cred := httpx.ConnectFromHeaders(r)
	cred.Normalize()
	if err := cred.Validate(); err != nil {
		writeValidationError(w, r, err)
		return
	}
	raw, err := fn(r.Context(), cred)
	h.finishProxy(w, r, raw, err)
}

func (h *Handler) finishProxy(w http.ResponseWriter, r *http.Request, raw []byte, err error) {
	if err == nil {
		writeProxySuccess(w, raw)
		return
	}
	m := greenapi.MapIntegrationError(err)
	if m.LogMsg != "" {
		h.logger.LogAttrs(r.Context(), m.LogLevel, m.LogMsg, m.LogAttrs...)
	}
	writeAPIError(w, r, m.HTTPStatus, m.APICode, m.APIMessage, m.Details)
}
