package logger

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestParseLevel_ValidAndInvalid(t *testing.T) {
	cases := map[string]zapcore.Level{
		"info":  zapcore.InfoLevel,
		"WARN":  zapcore.WarnLevel,
		"error": zapcore.ErrorLevel,
		"":      zapcore.InfoLevel,  // zap's UnmarshalText treats "" as info
		"bogus": zapcore.DebugLevel, // genuinely invalid text falls back to debug
	}
	for input, want := range cases {
		if got := parseLevel(input); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestLogFilePath_UsesConfiguredDir(t *testing.T) {
	got := LogFilePath(Config{Dir: "custom-logs"})
	if filepath.Dir(got) != "custom-logs" {
		t.Errorf("LogFilePath dir = %q, want custom-logs", filepath.Dir(got))
	}
}

func TestNew_WritesToConfiguredDir(t *testing.T) {
	dir := t.TempDir()
	log := New("debug", Config{Dir: dir})
	log.Info("hello")
	_ = log.Sync()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected a log file to be created, found none")
	}
}
