package configwriter

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func WriteConfig(path string, data interface{}, headerComment string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	// Open the file with read-write permissions
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Write header comment
	if headerComment != "" {
		if _, err := file.WriteString(fmt.Sprintf("%s\n\n", headerComment)); err != nil {
			return fmt.Errorf("failed to write header comment: %w", err)
		}
	}

	// Encode YAML data
	enc := yaml.NewEncoder(file)
	enc.SetIndent(2)
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}
