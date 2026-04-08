package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestConnectRequest_Validate(t *testing.T) {
	err := (&ConnectRequest{IDInstance: "1101", APITokenInstance: "tok"}).Validate()
	if err != nil {
		t.Fatal(err)
	}
	err = (&ConnectRequest{IDInstance: "110", APITokenInstance: "tok"}).Validate()
	var ve *ValidationError
	if err == nil || !errors.As(err, &ve) || ve.Fields["idInstance"] == "" {
		t.Fatalf("expected idInstance error, got %v", err)
	}
	err = (&ConnectRequest{IDInstance: "1101", APITokenInstance: ""}).Validate()
	if err == nil || !errors.As(err, &ve) || ve.Fields["apiTokenInstance"] == "" {
		t.Fatal("expected token error")
	}
}

func TestSendMessageRequest_Validate(t *testing.T) {
	r := SendMessageRequest{
		ConnectRequest: ConnectRequest{IDInstance: "1101", APITokenInstance: "t"},
		ChatID:         "79990001122@c.us",
		Message:        "hi",
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("а", MaxMessageRunes+1)
	r.Message = long
	var ve *ValidationError
	if err := r.Validate(); err == nil || !errors.As(err, &ve) || ve.Fields["message"] == "" {
		t.Fatal("expected message length error")
	}
}

func TestSendFileByURLRequest_Validate_chatId(t *testing.T) {
	r := SendFileByURLRequest{
		ConnectRequest: ConnectRequest{IDInstance: "1101", APITokenInstance: "t"},
		ChatID:         "bad",
		FileURL:        "https://example.com/f",
		FileName:       "x",
	}
	var ve *ValidationError
	if err := r.Validate(); err == nil || !errors.As(err, &ve) || ve.Fields["chatId"] == "" {
		t.Fatal("expected chatId error")
	}
}

func TestValidate_withoutNormalize_rejectsPaddedID(t *testing.T) {
	c := ConnectRequest{IDInstance: "  1101  ", APITokenInstance: "tok"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected validation error without Normalize")
	}
}

func TestNormalize_thenValidate(t *testing.T) {
	c := ConnectRequest{IDInstance: "  1101  ", APITokenInstance: "  tok  "}
	c.Normalize()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.IDInstance != "1101" || c.APITokenInstance != "tok" {
		t.Fatalf("normalize: %+v", c)
	}
}

func TestSendMessageRequest_Normalize_preservesInnerSpaces(t *testing.T) {
	r := SendMessageRequest{
		ConnectRequest: ConnectRequest{IDInstance: "1101", APITokenInstance: "t"},
		ChatID:         "  79990001122@c.us ",
		Message:        "  hello  world  ",
	}
	r.Normalize()
	if r.Message != "hello  world" {
		t.Fatalf("message: %q", r.Message)
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_fileURL_rejectsRelative(t *testing.T) {
	r := SendFileByURLRequest{
		ConnectRequest: ConnectRequest{IDInstance: "1101", APITokenInstance: "t"},
		ChatID:         "79990001122@c.us",
		FileURL:        "/files/doc.pdf",
		FileName:       "x",
	}
	r.Normalize()
	var ve *ValidationError
	if err := r.Validate(); err == nil || !errors.As(err, &ve) || ve.Fields["fileUrl"] == "" {
		t.Fatalf("expected fileUrl error, got %v", err)
	}
}

func TestValidate_fileURL_rejectsUserInfo(t *testing.T) {
	r := SendFileByURLRequest{
		ConnectRequest: ConnectRequest{IDInstance: "1101", APITokenInstance: "t"},
		ChatID:         "79990001122@c.us",
		FileURL:        "https://user:pass@example.com/file",
		FileName:       "x",
	}
	r.Normalize()
	var ve *ValidationError
	if err := r.Validate(); err == nil || !errors.As(err, &ve) || ve.Fields["fileUrl"] == "" {
		t.Fatalf("expected fileUrl error, got %v", err)
	}
}
