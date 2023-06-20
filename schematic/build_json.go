package schematic

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strings"
)

func BuildEvents(path *string, genSchema map[string]Schema) error {
	endsWithSlash := regexp.MustCompile("/$")

	if !endsWithSlash.MatchString(*path) {
		*path += "/"
	}

	if _, err := os.Stat(*path); os.IsNotExist(err) {
		err := os.MkdirAll(*path, 0o744)
		if err != nil {
			log.Fatalf("error while creating path to save files: %s", err)
		}
	}

	for name, schema := range genSchema {
		marshal, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			log.Fatalf("error while marshaling schema %s: %s", schema.Title, err)
		}
		filename := buildFileName(name)
		filename = *path + filename
		err = os.WriteFile(filename, marshal, 0o644)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildFileName(name string) string {
	filename := strings.ReplaceAll(name, ".", "_") + ".json"
	return filename
}
