package schematic

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// BuildEvents generates JSON Schema files from the provided schema definitions
// It creates the directory structure if it doesn't exist and writes each schema to a separate file
func BuildEvents(path *string, genSchema map[string]Schema) error {
	endsWithSlash := regexp.MustCompile("/$")

	if !endsWithSlash.MatchString(*path) {
		*path += "/"
	}

	if _, err := os.Stat(*path); os.IsNotExist(err) {
		err := os.MkdirAll(*path, 0o744)
		if err != nil {
			return fmt.Errorf("error while creating path to save files: %w", err)
		}
	}

	for name, schema := range genSchema {
		marshal, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return fmt.Errorf("error while marshaling schema %s: %w", schema.Title, err)
		}
		filename := buildFileName(name)
		filename = *path + filename
		err = os.WriteFile(filename, marshal, 0o644)
		if err != nil {
			return fmt.Errorf("error while writing file %s: %w", filename, err)
		}
	}

	return nil
}

func buildFileName(name string) string {
	filename := strings.ReplaceAll(name, ".", "_") + ".json"
	return filename
}
