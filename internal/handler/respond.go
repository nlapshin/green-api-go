package handler

import (
	"errors"
	"net/http"

	"green-api-test/internal/domain"
	"green-api-test/internal/httpx"
	"green-api-test/internal/jsonfmt"
)

func writeAPIError(w http.ResponseWriter, status int, code, message string, details any) {
	httpx.WriteJSON(w, status, domain.APIResponse{
		OK: false,
		Error: &domain.APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func writeDecodeError(w http.ResponseWriter, err error) {
	msg := "Invalid JSON"
	if errors.Is(err, httpx.ErrBodyRequired) {
		msg = "Request body is required"
	}
	writeAPIError(w, http.StatusBadRequest, "invalid_json", msg, nil)
}

func writeValidationError(w http.ResponseWriter, err error) {
	var ve *domain.ValidationError
	if errors.As(err, &ve) {
		writeAPIError(w, http.StatusBadRequest, "validation_error", "Validation error", ve.FieldsOrNil())
		return
	}
	writeAPIError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
}

func writeProxySuccess(w http.ResponseWriter, raw []byte) {
	httpx.WriteJSON(w, http.StatusOK, domain.APIResponse{
		OK:     true,
		Pretty: jsonfmt.PrettyJSON(raw),
	})
}
