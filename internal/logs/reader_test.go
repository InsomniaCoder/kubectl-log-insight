package logs_test

import (
	"strings"
	"testing"

	"github.com/InsomniaCoder/kubectl-log-insight/internal/logs"
)

func TestRead_FromReader(t *testing.T) {
	input := "line1\nline2\nline3\n"
	content, truncated, err := logs.Read(strings.NewReader(input), 1000)
	if err != nil {
		t.Fatal(err)
	}
	if content != input {
		t.Errorf("expected %q, got %q", input, content)
	}
	if truncated {
		t.Error("expected no truncation")
	}
}

func TestRead_Truncates_WhenOverLimit(t *testing.T) {
	// 2 bytes per token heuristic: maxTokens=2 means 4 bytes max
	longInput := "hello world this is a long log line that exceeds the limit\n"
	content, truncated, err := logs.Read(strings.NewReader(longInput), 2)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Error("expected truncation")
	}
	if len(content) > 4 {
		t.Errorf("expected at most 4 bytes, got %d", len(content))
	}
}

func TestRead_Truncates_KeepsMostRecentLines(t *testing.T) {
	// 2 bytes/token, maxTokens=5 = 10 bytes max
	// Each line is 6 bytes ("lineN\n"), so only last line fits cleanly
	lines := "line1\nline2\nline3\nline4\nline5\n"
	content, truncated, err := logs.Read(strings.NewReader(lines), 5)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Error("expected truncation for 30-byte input with 10-byte limit")
	}
	if !strings.Contains(content, "line5") {
		t.Error("expected most recent line (line5) to be present")
	}
	if strings.Contains(content, "line1") {
		t.Error("expected oldest line (line1) to be truncated")
	}
}
