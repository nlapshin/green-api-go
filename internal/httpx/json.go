package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

var (
	ErrBodyRequired    = errors.New("request body is required")
	ErrInvalidJSON     = errors.New("invalid json")
	ErrUnexpectedExtra = errors.New("unexpected extra json data")
	ErrReadBody        = errors.New("failed to read body")
	ErrBodyTooLarge    = errors.New("request body too large")
)

func DecodeStrictJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	if r.Body == nil {
		return ErrBodyRequired
	}
	defer func() { _ = r.Body.Close() }()

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrBodyRequired
		}
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return ErrBodyTooLarge
		}
		return ErrInvalidJSON
	}
	switch err := dec.Decode(&struct{}{}); {
	case errors.Is(err, io.EOF):
		return nil
	case err == nil:
		return ErrUnexpectedExtra
	default:
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return ErrBodyTooLarge
		}
		return ErrInvalidJSON
	}
}
