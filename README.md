# Schematic
This tool generates JSON schemas from Go structs which can be used as event definition produced by a service.

## Usage
Create a new directory and in a main.go file add/define your events like follows and use the below `func main()` snippet
```
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
```
By default the schemas will be generate to `/tmp/schemas/` but this can be redefined as needed when the go program is executed by using the `-path` parameter.

# Contributors
[@endrit101](https://github.com/endrit101) - Endrit Toplica
