package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeStrictJSON_ok(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":1}`))
	var v struct{ A int `json:"a"` }
	if err := DecodeStrictJSON(httptest.NewRecorder(), r, &v, 1024); err != nil {
		t.Fatal(err)
	}
	if v.A != 1 {
		t.Fatal(v.A)
	}
}

func TestDecodeStrictJSON_extraRejected(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":1}{}`))
	var v struct{ A int `json:"a"` }
	err := DecodeStrictJSON(httptest.NewRecorder(), r, &v, 1024)
	if err != ErrUnexpectedExtra {
		t.Fatalf("got %v", err)
	}
}

func TestDecodeStrictJSON_nilBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	var v struct{}
	err := DecodeStrictJSON(httptest.NewRecorder(), r, &v, 1024)
	if err != ErrBodyRequired {
		t.Fatalf("got %v", err)
	}
}

func TestDecodeStrictJSON_unknownField(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"x":1}`))
	var v struct{}
	err := DecodeStrictJSON(httptest.NewRecorder(), r, &v, 1024)
	if err != ErrInvalidJSON {
		t.Fatalf("got %v", err)
	}
}
