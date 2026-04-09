package domain

const (
	MaxMessageRunes  = 4096
	MaxCaptionRunes  = 1024
	MaxFileNameRunes = 255
	MaxIDInstanceLen = 32
	MaxAPITokenLen   = 128

	// MaxJSONRequestBody limits JSON bodies on POST API handlers (pretty-printed error if exceeded).
	MaxJSONRequestBody = 1 << 20 // 1 MiB
)
