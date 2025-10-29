package configwriter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper to read file content
func readFile(t *testing.T, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	return string(data)
}

func TestWriteConfig_Success(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "asena.yaml")

	// create the file manually (since WriteConfig expects it to exist)
	err := os.WriteFile(path, []byte("old: config"), 0640)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := map[string]interface{}{
		"port":      8080,
		"log_level": "info",
	}

	err = WriteConfig(path, cfg, "# Normalized by Asena")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := readFile(t, path)

	if !strings.HasPrefix(content, "# Normalized by Asena") {
		t.Errorf("expected header comment, got:\n%s", content)
	}

	if !strings.Contains(content, "port: 8080") || !strings.Contains(content, "log_level: info") {
		t.Errorf("expected YAML content not found, got:\n%s", content)
	}
}

func TestWriteConfig_DirectoryNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent", "asena.yaml")

	cfg := map[string]interface{}{"port": 8080}
	err := WriteConfig(path, cfg, "# Test comment")

	if err == nil {
		t.Fatalf("expected error for nonexistent directory, got nil")
	}

	if !strings.Contains(err.Error(), "directory does not exist") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestWriteConfig_EmptyHeader(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "asena.yaml")

	err := os.WriteFile(path, []byte("foo: bar"), 0640)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cfg := map[string]interface{}{
		"debug": true,
	}

	err = WriteConfig(path, cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := readFile(t, path)

	if strings.HasPrefix(strings.TrimSpace(content), "#") {
		t.Errorf("expected no comment header, got:\n%s", content)
	}
	if !strings.Contains(content, "debug: true") {
		t.Errorf("expected YAML data, got:\n%s", content)
	}
}
