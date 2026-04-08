package domain

type APIResponse struct {
	OK     bool      `json:"ok"`
	Pretty string    `json:"pretty,omitempty"`
	Error  *APIError `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}
