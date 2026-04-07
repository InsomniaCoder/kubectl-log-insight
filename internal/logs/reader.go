package logs

import (
	"io"
	"strings"
)

// bytesPerToken is a conservative estimate for Kubernetes logs.
// K8s logs contain timestamps, JSON, Go errors, and stack traces which tokenize
// at ~2 bytes/token — much denser than prose (4 bytes/token).
const bytesPerToken = 2

// Read reads all content from r and truncates to maxTokens if needed.
// Truncation removes from the top (oldest lines) to keep most recent content.
// Returns (content, wasTruncated, error).
func Read(r io.Reader, maxTokens int) (string, bool, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return "", false, err
	}

	content := string(raw)
	maxBytes := maxTokens * bytesPerToken

	if len(content) <= maxBytes {
		return content, false, nil
	}

	// Keep the tail (most recent lines)
	truncated := content[len(content)-maxBytes:]
	// Trim to a clean line boundary
	if idx := strings.Index(truncated, "\n"); idx >= 0 {
		truncated = truncated[idx+1:]
	}

	return truncated, true, nil
}
