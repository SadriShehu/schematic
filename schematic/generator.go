package schematic

import (
	"reflect"
	"strings"
)

const typeArray string = "array"

type Schema struct {
	Schemas    string                        `json:"$schema"`
	Title      string                        `json:"title"`
	Type       string                        `json:"type"`
	Required   []string                      `json:"required,omitempty"`
	Properties map[string]PropertyDefinition `json:"properties"`
}

type PropertyDefinition struct {
	Type        string                        `json:"type"`
	Description string                        `json:"description,omitempty"`
	Format      string                        `json:"format,omitempty"`
	Required    []string                      `json:"required,omitempty"`
	Items       *PropertyDefinition           `json:"items,omitempty"`
	Properties  map[string]PropertyDefinition `json:"properties,omitempty"`
}

func GenerateProperties[T any](object T) map[string]PropertyDefinition {
	// ignore required field since it is being build separately for the main object
	properties, _ := buildProperties(reflect.TypeOf(object), make(map[reflect.Type]bool), 0)
	return properties
}

func buildProperties(t reflect.Type, visited map[reflect.Type]bool, nestedCounter int) (map[string]PropertyDefinition, []string) {
	properties := map[string]PropertyDefinition{}
	var required []string
	switch t.Kind() {
	case reflect.Slice:
		t := t.Elem()

		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		if t.Kind() != reflect.Struct {
			break
		}
		if visited[t] && nestedCounter > 1 {
			// Don't blow up on recursive type definition.
			break
		}
		visited[t] = true
		properties = doReflection(t, visited, nestedCounter)
	case reflect.Struct:
		if visited[t] && nestedCounter > 1 {
			// Don't blow up on recursive type definition.
			break
		}
		visited[t] = true
		properties = doReflection(t, visited, nestedCounter)
	case reflect.Ptr:
		if t.Elem().Kind() == reflect.Slice &&
			t.Elem().Elem().Kind() == reflect.Struct {
			t = t.Elem()
		}
		if t.Elem().Kind() != reflect.Struct {
			break
		}
		// get the underlying pointer element and reflect it as struct
		t := t.Elem()
		if visited[t] && nestedCounter > 1 {
			// Don't blow up on recursive type definition.
			break
		}
		visited[t] = true
		properties = doReflection(t, visited, nestedCounter)
	}

	// build required field for nested struct
	required = GenerateRequired(nil, t)

	return properties, required
}

func doReflection(t reflect.Type, visited map[reflect.Type]bool, nestedCounter int) map[string]PropertyDefinition {
	properties := map[string]PropertyDefinition{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		// extract tag from json
		tag := t.Field(i).Tag.Get("json")
		args := strings.Split(tag, ",")
		tagName := args[0]
		// if field has no json tag, use the variable name
		if len(tagName) == 0 {
			tagName = f.Name
		}

		var (
			typeName, format, sliceFormat, sliceTypeName string
			skipNested                                   bool
			required                                     []string
		)

		typeName = f.Type.String()

		var items *PropertyDefinition
		switch f.Type.Kind() {

		case reflect.Slice:
			// extract the slice type
			sliceTypeName = f.Type.Elem().Kind().String()
			if sliceTypeName == "ptr" {
				sliceTypeName = f.Type.Elem().String()
			}
			// a slice in JSON is array
			typeName = typeArray
			// convert slice type to event type
			sliceTypeName, sliceFormat, skipNested = convertToEventName(sliceTypeName, nil)
		case reflect.Ptr:
			sliceTypeName = f.Type.Elem().Kind().String()
			if sliceTypeName == "slice" {
				typeName = typeArray
				sliceTypeName = f.Type.Elem().Elem().String()
				// convert slice type to event type
				sliceTypeName, sliceFormat, skipNested = convertToEventName(sliceTypeName, &f.Type)
			} else {
				typeName, format, skipNested = convertToEventName(typeName, &f.Type)
			}
		default:
			// convert to event type
			typeName, format, skipNested = convertToEventName(typeName, &f.Type)
		}

		nested := map[string]PropertyDefinition{}
		if !skipNested {
			nestedCounter++
			nested, required = buildProperties(f.Type, visited, nestedCounter)
		}
		// reset the nested counter if we are only one recursion deep for that type
		// this is needed because we can have multiple elements of type e.g. DateTime and we want all of them to have their properties added
		nestedCounter = 0

		if typeName == typeArray {
			// define slice items
			// if there are nested properties this type is object in event schema, overwrite it
			if len(nested) > 0 {
				sliceTypeName = "object"
			}
			items = &PropertyDefinition{
				Type:        sliceTypeName,
				Description: f.Name,
				Properties:  nested,
				Format:      sliceFormat,
				Required:    required,
			}
		}

		// if there are nested properties this type is object in event schema, overwrite it
		if len(nested) > 0 && typeName != typeArray {
			typeName = "object"
		}

		if typeName == typeArray {
			properties[tagName] = PropertyDefinition{
				Type:        typeName,
				Description: f.Name,
				Format:      format,
				Items:       items,
				Required:    required,
			}
			continue
		}

		properties[tagName] = PropertyDefinition{
			Type:        typeName,
			Description: f.Name,
			Properties:  nested,
			Format:      format,
			Items:       items,
			Required:    required,
		}
	}
	return properties
}

func GenerateRequired(object interface{}, nestedObject reflect.Type) []string {
	t := reflect.TypeOf(object)

	if nestedObject != nil {
		t = nestedObject
	}

	var require []string

	switch t.Kind() {
	case reflect.Ptr:
		t = t.Elem()
	// if not struct skip
	case reflect.String:
		return require
	case reflect.Slice:
		return require
	case reflect.Int:
		return require
	case reflect.Bool:
		return require
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")

		if field.Type.Kind() == reflect.Slice {
			continue
		}

		if field.Type.Kind() != reflect.Ptr {
			args := strings.Split(tag, ",")
			tagName := args[0]
			// if field has no json tag, use the variable name
			if len(tagName) == 0 {
				tagName = field.Name
			}
			omitempty := strings.Join(args[1:], ",")

			// check if the field is required by checking if it has omitempty
			if len(omitempty) != 0 {
				require = append(require, tagName)
				continue
			}

			require = append(require, tagName)
		} else if tag == "tags" {
			// basic is pointer so we need to add it to required, skip the rest
			require = append(require, tag)
		}
	}

	return require
}

func convertToEventName(defType string, refType *reflect.Type) (string, string, bool) {
	// check if type is primitive type or from kit library (renamed built in type) that is not supported by event schema
	// if type is struct we do not handle it here
	switch defType {
	case "uuid.UUID", "EventName", "*string", "string":
		return "string", "", true
	case "float64", "*float64", "float32", "*float32":
		return "number", "", true
	case "int", "*int", "uint64", "int64", "*int64", "int32", "*int32":
		return "integer", "", true
	case "bool", "*bool":
		return "boolean", "", true
	case "time.Time":
		return "string", "date-time", true
	default:
		if refType != nil {
			f := *refType
			if f.Kind().String() != "ptr" {
				return convertToEventName(f.Kind().String(), nil)
			}
			if f.Elem().Kind() != reflect.Struct {
				return convertToEventName(f.Elem().Kind().String(), nil)
			}
		}
		return defType, "", false
	}
}
