package handler

import (
	"errors"
	"net/http"

	"green-api-test/internal/domain"
	"green-api-test/internal/httpx"
	"green-api-test/internal/jsonfmt"
)

func writeAPIError(w http.ResponseWriter, r *http.Request, status int, code, message string, details any) {
	httpx.WriteAPIErrorResponse(w, r, status, code, message, details)
}

func writeDecodeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, httpx.ErrBodyTooLarge):
		writeAPIError(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body is too large", nil)
		return
	case errors.Is(err, httpx.ErrBodyRequired):
		writeAPIError(w, r, http.StatusBadRequest, "invalid_json", "Request body is required", nil)
		return
	default:
		writeAPIError(w, r, http.StatusBadRequest, "invalid_json", "Invalid JSON", nil)
	}
}

func writeValidationError(w http.ResponseWriter, r *http.Request, err error) {
	var ve *domain.ValidationError
	if errors.As(err, &ve) {
		writeAPIError(w, r, http.StatusBadRequest, "validation_error", "Validation error", ve.FieldsOrNil())
		return
	}
	writeAPIError(w, r, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
}

func writeProxySuccess(w http.ResponseWriter, raw []byte) {
	httpx.WriteJSON(w, http.StatusOK, domain.APIResponse{
		OK:     true,
		Pretty: jsonfmt.PrettyJSON(raw),
	})
}
