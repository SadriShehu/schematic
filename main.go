package main

import (
	"flag"
	"log"

	"github.com/sadrishehu/schematic/schematic"
)

type YourEventTags struct {
	EventName    string `json:"event_name"`
	EventVersion string `json:"event_version"`
	EventID      string `json:"event_id"`
}

type YourFieldStruct struct {
	YourFieldString string `json:"your_field_string"`
	YourFieldInt    int    `json:"your_field_int"`
}

type YourEventStruct struct {
	Tags               YourEventTags    `json:"tags"`
	YourFieldString    string           `json:"your_field_string"`
	YourFieldInt       int              `json:"your_field_int"`
	YourFieldFloat     float64          `json:"your_field_float"`
	YourFieldBool      bool             `json:"your_field_bool"`
	YourFieldSlice     []string         `json:"your_field_slice"`
	YourFieldStruct    YourFieldStruct  `json:"your_field_struct"`
	YourFieldPtr       *string          `json:"your_field_ptr"`
	YourFieldPtrSlice  []*string        `json:"your_field_ptr_slice"`
	YourFieldPtrStruct *YourFieldStruct `json:"your_field_ptr_struct"`
}

var genSchema map[string]schematic.Schema = map[string]schematic.Schema{
	"event.name": schematic.GenerateSchema(
		YourEventStruct{},
		"Cute Event Name",
		"http://json-schema.org/draft-07/schema#",
	),
}

func main() {
	path := flag.String("path", "/tmp/schemas/", "enter full path where to save schemas")
	help := flag.Bool("help", false, "print help/usage information")

	flag.Parse()

	if *help {
		flag.PrintDefaults()
		return
	}

	if err := schematic.BuildEvents(path, genSchema); err != nil {
		log.Fatalf("there was an error during file writing. Error: %s", err)
	}

	log.Printf("Schemas generated succssfully, located at: %s", *path)
}
