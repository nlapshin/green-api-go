package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"green-api-test/internal/domain"
)

// WriteAPIErrorResponse writes the unified JSON error envelope. request_id is taken from chi when r is non-nil.
func WriteAPIErrorResponse(w http.ResponseWriter, r *http.Request, status int, code, message string, details any) {
	rid := ""
	if r != nil {
		rid = middleware.GetReqID(r.Context())
	}
	WriteJSON(w, status, domain.APIResponse{
		OK: false,
		Error: &domain.APIError{
			Code:      code,
			Message:   message,
			RequestID: rid,
			Details:   details,
		},
	})
}
