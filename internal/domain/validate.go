package domain

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "validation error" }

func (e *ValidationError) FieldsOrNil() map[string]string {
	if e == nil || len(e.Fields) == 0 {
		return nil
	}
	return e.Fields
}

func newValidationError(fields map[string]string) error {
	if len(fields) == 0 {
		return nil
	}
	cp := make(map[string]string, len(fields))
	for k, v := range fields {
		cp[k] = v
	}
	return &ValidationError{Fields: cp}
}

func mergeFieldErrors(parts ...map[string]string) map[string]string {
	out := make(map[string]string)
	for _, p := range parts {
		if p == nil {
			continue
		}
		for k, v := range p {
			out[k] = v
		}
	}
	return out
}

var idInstanceRe = regexp.MustCompile(`^[0-9]{4,32}$`)
var chatIDRe = regexp.MustCompile(`^[^@\s]+@(c\.us|g\.us)$`)

func (c *ConnectRequest) Normalize() {
	c.IDInstance = strings.TrimSpace(c.IDInstance)
	c.APITokenInstance = strings.TrimSpace(c.APITokenInstance)
}

func (r *SendMessageRequest) Normalize() {
	r.ConnectRequest.Normalize()
	r.ChatID = strings.TrimSpace(r.ChatID)
	r.Message = strings.TrimSpace(r.Message)
}

func (r *SendFileByURLRequest) Normalize() {
	r.ConnectRequest.Normalize()
	r.ChatID = strings.TrimSpace(r.ChatID)
	r.FileURL = strings.TrimSpace(r.FileURL)
	r.FileName = strings.TrimSpace(r.FileName)
	r.Caption = strings.TrimSpace(r.Caption)
}

func (c *ConnectRequest) Validate() error {
	return newValidationError(validateConnect(c.IDInstance, c.APITokenInstance))
}

func (r *SendMessageRequest) Validate() error {
	return newValidationError(mergeFieldErrors(
		validateConnect(r.IDInstance, r.APITokenInstance),
		validateChatID(r.ChatID),
		validateMessage(r.Message),
	))
}

func (r *SendFileByURLRequest) Validate() error {
	return newValidationError(mergeFieldErrors(
		validateConnect(r.IDInstance, r.APITokenInstance),
		validateChatID(r.ChatID),
		validateFileURL(r.FileURL),
		validateFileName(r.FileName),
		validateCaption(r.Caption),
	))
}

func validateConnect(idInstance, apiToken string) map[string]string {
	errs := make(map[string]string)
	if idInstance == "" {
		errs["idInstance"] = "idInstance is required"
	} else if len(idInstance) > MaxIDInstanceLen || !idInstanceRe.MatchString(idInstance) {
		errs["idInstance"] = "idInstance must be 4–32 digits"
	}
	if apiToken == "" {
		errs["apiTokenInstance"] = "apiTokenInstance is required"
	} else if len(apiToken) > MaxAPITokenLen {
		errs["apiTokenInstance"] = "apiTokenInstance is too long"
	}
	return errs
}

func validateChatID(chatID string) map[string]string {
	if chatID == "" {
		return map[string]string{"chatId": "chatId is required"}
	}
	if !chatIDRe.MatchString(chatID) {
		return map[string]string{"chatId": "chatId must end with @c.us or @g.us"}
	}
	return nil
}

func validateMessage(message string) map[string]string {
	if message == "" {
		return map[string]string{"message": "message is required"}
	}
	if utf8.RuneCountInString(message) > MaxMessageRunes {
		return map[string]string{"message": "message exceeds maximum length"}
	}
	return nil
}

func validateFileName(name string) map[string]string {
	if name == "" {
		return map[string]string{"fileName": "fileName is required"}
	}
	if utf8.RuneCountInString(name) > MaxFileNameRunes {
		return map[string]string{"fileName": "fileName is too long"}
	}
	return nil
}

func validateCaption(caption string) map[string]string {
	if caption == "" {
		return nil
	}
	if utf8.RuneCountInString(caption) > MaxCaptionRunes {
		return map[string]string{"caption": "caption exceeds maximum length"}
	}
	return nil
}

func validateFileURL(fileURL string) map[string]string {
	if fileURL == "" {
		return map[string]string{"fileUrl": "fileUrl is required"}
	}
	u, err := url.ParseRequestURI(fileURL)
	if err != nil {
		return map[string]string{"fileUrl": "fileUrl must be a valid URL"}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return map[string]string{"fileUrl": "fileUrl must use http or https scheme"}
	}
	if u.Host == "" {
		return map[string]string{"fileUrl": "fileUrl must be a valid URL"}
	}
	if u.User != nil {
		return map[string]string{"fileUrl": "fileUrl must not contain user info"}
	}
	return nil
}
