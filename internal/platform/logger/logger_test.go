package logger_test

import (
	"bytes"
	"encoding/json/v2"
	"testing"

	"github.com/imamfzn/go-bootstrap/internal/platform/logger"
	"github.com/imamfzn/go-bootstrap/internal/platform/requestid"
)

func TestNew_CloudLoggingShape(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, "info")

	ctx := requestid.With(t.Context(), "req-1")
	log.WarnContext(ctx, "disk almost full", "free_mb", 12)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("log line is not JSON: %v\n%s", err, buf.String())
	}

	want := map[string]any{
		"severity":   "WARNING",
		"message":    "disk almost full",
		"request_id": "req-1",
		"free_mb":    float64(12),
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
	for _, k := range []string{"level", "msg"} {
		if _, ok := got[k]; ok {
			t.Errorf("unexpected slog default key %q in output", k)
		}
	}
}

func TestNew_LevelFilters(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(&buf, "error")

	log.InfoContext(t.Context(), "ignored")

	if buf.Len() != 0 {
		t.Fatalf("info record was written at error level: %s", buf.String())
	}
}
