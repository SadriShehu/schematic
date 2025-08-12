package schematic

import (
	"reflect"
	"strings"
	"unicode"
)

const typeArray string = "array"

// toSnakeCase converts PascalCase or camelCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// Schema represents a JSON Schema definition
type Schema struct {
	Schema      string                        `json:"$schema"`
	Title       string                        `json:"title"`
	Type        string                        `json:"type"`
	Required    []string                      `json:"required,omitempty"`
	Properties  map[string]PropertyDefinition `json:"properties"`
	Definitions map[string]PropertyDefinition `json:"$defs,omitempty"`
}

// PropertyDefinition represents a property within a JSON Schema
type PropertyDefinition struct {
	Type        string                        `json:"type,omitempty"`
	Description string                        `json:"description,omitempty"`
	Format      string                        `json:"format,omitempty"`
	Required    []string                      `json:"required,omitempty"`
	Items       *PropertyDefinition           `json:"items,omitempty"`
	Properties  map[string]PropertyDefinition `json:"properties,omitempty"`
	Ref         string                        `json:"$ref,omitempty"`
}

// fieldInfo contains information about a struct field for schema generation
type fieldInfo struct {
	Field       reflect.StructField
	TagName     string
	TypeName    string
	Format      string
	SliceFormat string
	SliceType   string
	SkipNested  bool
	IsArray     bool
}

// schemaContext tracks state during schema generation
type schemaContext struct {
	visited     map[reflect.Type]bool
	definitions map[string]PropertyDefinition
	counter     int
}

// GenerateProperties creates JSON Schema properties from a Go struct type
func GenerateProperties[T any](object T) map[string]PropertyDefinition {
	ctx := &schemaContext{
		visited:     make(map[reflect.Type]bool),
		definitions: make(map[string]PropertyDefinition),
	}
	properties, _ := ctx.buildProperties(reflect.TypeOf(object), 0)
	return properties
}

// GenerateSchema creates a complete JSON Schema with definitions from a Go struct type
func GenerateSchema[T any](object T, title, schemaURL string) Schema {
	ctx := &schemaContext{
		visited:     make(map[reflect.Type]bool),
		definitions: make(map[string]PropertyDefinition),
	}
	properties, _ := ctx.buildProperties(reflect.TypeOf(object), 0)

	schema := Schema{
		Schema:     schemaURL,
		Title:      title,
		Type:       "object",
		Required:   GenerateRequired(object, nil),
		Properties: properties,
	}

	if len(ctx.definitions) > 0 {
		schema.Definitions = ctx.definitions
	}

	return schema
}

func (ctx *schemaContext) buildProperties(t reflect.Type, nestedCounter int) (map[string]PropertyDefinition, []string) {
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
		if ctx.visited[t] && nestedCounter > 1 {
			// Don't blow up on recursive type definition.
			break
		}
		ctx.visited[t] = true
		properties = ctx.reflectStruct(t, nestedCounter)
	case reflect.Struct:
		if ctx.visited[t] && nestedCounter > 1 {
			// Don't blow up on recursive type definition.
			break
		}
		ctx.visited[t] = true
		properties = ctx.reflectStruct(t, nestedCounter)
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
		if ctx.visited[t] && nestedCounter > 1 {
			// Don't blow up on recursive type definition.
			break
		}
		ctx.visited[t] = true
		properties = ctx.reflectStruct(t, nestedCounter)
	}

	// build required field for nested struct
	required = GenerateRequired(nil, t)

	return properties, required
}

// reflectStruct processes a struct type and generates properties for all its fields
func (ctx *schemaContext) reflectStruct(t reflect.Type, nestedCounter int) map[string]PropertyDefinition {
	properties := map[string]PropertyDefinition{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldInfo := ctx.extractFieldInfo(field)

		if fieldInfo.TagName == "-" || fieldInfo.TagName == "" {
			continue // Skip fields with json:"-" or invalid tag names
		}

		property := ctx.buildFieldProperty(fieldInfo, nestedCounter)
		properties[fieldInfo.TagName] = property
	}

	return properties
}

// extractFieldInfo extracts field information needed for schema generation
func (ctx *schemaContext) extractFieldInfo(field reflect.StructField) fieldInfo {
	tag := field.Tag.Get("json")
	args := strings.Split(tag, ",")
	tagName := args[0]

	// Skip fields with json:"-"
	if tagName == "-" {
		return fieldInfo{Field: field, TagName: "-"}
	}

	// If field has no json tag, use the variable name converted to snake_case
	if len(tagName) == 0 {
		tagName = toSnakeCase(field.Name)
	}

	info := fieldInfo{
		Field:   field,
		TagName: tagName,
	}

	// Determine the type information based on field type
	ctx.analyzeFieldType(&info)

	return info
}

// analyzeFieldType determines the JSON Schema type information for a field
func (ctx *schemaContext) analyzeFieldType(info *fieldInfo) {
	fieldType := info.Field.Type
	info.TypeName = fieldType.String()

	// Handle interface{} specially
	if info.TypeName == "interface{}" {
		info.TypeName = ""
		info.Format = ""
		info.SkipNested = true
		return
	}

	// Handle []byte specially - should be string with byte format, not array
	if info.TypeName == "[]byte" || info.TypeName == "[]uint8" {
		info.TypeName = "string"
		info.Format = "byte"
		info.SkipNested = true
		info.IsArray = false
		return
	}

	switch fieldType.Kind() {
	case reflect.Slice:
		info.IsArray = true
		info.TypeName = typeArray
		ctx.handleSliceType(info, fieldType)
	case reflect.Ptr:
		ctx.handlePointerType(info, fieldType)
	default:
		info.TypeName, info.Format, info.SkipNested = convertToEventName(info.TypeName, &fieldType)
	}
}

// handleSliceType processes slice/array field types
func (ctx *schemaContext) handleSliceType(info *fieldInfo, fieldType reflect.Type) {
	elemType := fieldType.Elem()
	info.SliceType = elemType.Kind().String()

	if info.SliceType == "ptr" {
		info.SliceType = elemType.String()
	}

	info.SliceType, info.SliceFormat, info.SkipNested = convertToEventName(info.SliceType, nil)
}

// handlePointerType processes pointer field types
func (ctx *schemaContext) handlePointerType(info *fieldInfo, fieldType reflect.Type) {
	elemType := fieldType.Elem()

	if elemType.Kind() == reflect.Slice {
		info.IsArray = true
		info.TypeName = typeArray
		info.SliceType = elemType.Elem().String()
		info.SliceType, info.SliceFormat, info.SkipNested = convertToEventName(info.SliceType, &fieldType)
	} else {
		info.TypeName, info.Format, info.SkipNested = convertToEventName(info.TypeName, &fieldType)
	}
}

// buildFieldProperty creates a PropertyDefinition for a single field
func (ctx *schemaContext) buildFieldProperty(info fieldInfo, nestedCounter int) PropertyDefinition {
	var nested map[string]PropertyDefinition
	var required []string

	// Handle nested structures
	if !info.SkipNested {
		nestedCounter++
		nested, required = ctx.buildProperties(info.Field.Type, nestedCounter)
		nestedCounter = 0 // Reset counter for next field
	}

	// Check if this is a reusable type that should be in definitions
	if ctx.shouldUseDefinition(info.Field.Type, nested) {
		return ctx.createDefinitionReference(info, nested, required)
	}

	if info.IsArray {
		return ctx.buildArrayProperty(info, nested, required)
	}

	return ctx.buildObjectProperty(info, nested, required)
}

// shouldUseDefinition determines if a type should be moved to $defs for reuse
func (ctx *schemaContext) shouldUseDefinition(fieldType reflect.Type, nested map[string]PropertyDefinition) bool {
	// Only create definitions for complex structs with multiple properties
	return len(nested) > 2 && fieldType.Kind() == reflect.Struct
}

// createDefinitionReference creates a $ref to a definition and stores the definition
func (ctx *schemaContext) createDefinitionReference(info fieldInfo, nested map[string]PropertyDefinition, required []string) PropertyDefinition {
	defName := info.Field.Type.Name()
	if defName == "" {
		defName = "AnonymousStruct" + strconv.Itoa(ctx.counter)
		ctx.counter++
	}

	// Store in definitions if not already present
	if _, exists := ctx.definitions[defName]; !exists {
		typeName := "object"
		if len(nested) > 0 {
			typeName = "object"
		}

		ctx.definitions[defName] = PropertyDefinition{
			Type:        typeName,
			Properties:  nested,
			Required:    required,
			Description: info.Field.Name,
		}
	}

	return PropertyDefinition{
		Ref:         "#/$defs/" + defName,
		Description: info.Field.Name,
	}
}

// buildArrayProperty creates a PropertyDefinition for array/slice fields
func (ctx *schemaContext) buildArrayProperty(info fieldInfo, nested map[string]PropertyDefinition, required []string) PropertyDefinition {
	sliceTypeName := info.SliceType
	if len(nested) > 0 {
		sliceTypeName = "object"
	}

	items := &PropertyDefinition{
		Type:        sliceTypeName,
		Description: info.Field.Name,
		Properties:  nested,
		Format:      info.SliceFormat,
		Required:    required,
	}

	return PropertyDefinition{
		Type:        typeArray,
		Description: info.Field.Name,
		Format:      info.Format,
		Items:       items,
		Required:    required,
	}
}

// buildObjectProperty creates a PropertyDefinition for object/struct fields
func (ctx *schemaContext) buildObjectProperty(info fieldInfo, nested map[string]PropertyDefinition, required []string) PropertyDefinition {
	typeName := info.TypeName
	if len(nested) > 0 && typeName != typeArray {
		typeName = "object"
	}

	return PropertyDefinition{
		Type:        typeName,
		Description: info.Field.Name,
		Properties:  nested,
		Format:      info.Format,
		Required:    required,
	}
}

// GenerateRequired determines which fields are required in a JSON Schema based on Go struct tags
// Fields are considered required if they don't have the "omitempty" tag and are not pointer types
func GenerateRequired(object interface{}, nestedObject reflect.Type) []string {
	t := reflect.TypeOf(object)

	if nestedObject != nil {
		t = nestedObject
	}

	var require []string

	// Handle nil type
	if t == nil {
		return require
	}

	switch t.Kind() {
	case reflect.Ptr:
		t = t.Elem()
		// Check again after dereferencing
		if t == nil {
			return require
		}
	// if not struct skip
	case reflect.String, reflect.Slice, reflect.Int, reflect.Bool, reflect.Interface,
		reflect.Map, reflect.Chan, reflect.Func, reflect.Float32, reflect.Float64,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return require
	}

	// Only process struct types
	if t.Kind() != reflect.Struct {
		return require
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")

		// Skip fields with json:"-"
		if tag == "-" {
			continue
		}

		if field.Type.Kind() == reflect.Slice {
			continue
		}

		if field.Type.Kind() != reflect.Ptr {
			args := strings.Split(tag, ",")
			tagName := args[0]
			// if field has no json tag, use the variable name converted to snake_case
			if len(tagName) == 0 {
				tagName = toSnakeCase(field.Name)
			}
			omitempty := strings.Join(args[1:], ",")

			// Field is required if it doesn't have omitempty tag
			if !strings.Contains(omitempty, "omitempty") {
				require = append(require, tagName)
			}
		} else {
			// Handle special case for tags field
			args := strings.Split(tag, ",")
			tagName := args[0]
			if tagName == "tags" {
				require = append(require, tagName)
			}
		}
	}

	return require
}

var typeMapping = map[string]struct {
	jsonType   string
	format     string
	skipNested bool
}{
	"uuid.UUID":        {"string", "uuid", true},
	"EventName":        {"string", "", true},
	"*string":          {"string", "", true},
	"string":           {"string", "", true},
	"float64":          {"number", "", true},
	"*float64":         {"number", "", true},
	"float32":          {"number", "", true},
	"*float32":         {"number", "", true},
	"int":              {"integer", "", true},
	"*int":             {"integer", "", true},
	"uint":             {"integer", "", true},
	"*uint":            {"integer", "", true},
	"uint8":            {"integer", "", true},
	"*uint8":           {"integer", "", true},
	"uint16":           {"integer", "", true},
	"*uint16":          {"integer", "", true},
	"uint32":           {"integer", "", true},
	"*uint32":          {"integer", "", true},
	"uint64":           {"integer", "", true},
	"*uint64":          {"integer", "", true},
	"int8":             {"integer", "", true},
	"*int8":            {"integer", "", true},
	"int16":            {"integer", "", true},
	"*int16":           {"integer", "", true},
	"int32":            {"integer", "", true},
	"*int32":           {"integer", "", true},
	"int64":            {"integer", "", true},
	"*int64":           {"integer", "", true},
	"bool":             {"boolean", "", true},
	"*bool":            {"boolean", "", true},
	"time.Time":        {"string", "date-time", true},
	"*time.Time":       {"string", "date-time", true},
	"interface{}":      {}, // Will be handled as any type
	"*interface{}":     {}, // Will be handled as any type
	"any":              {}, // Will be handled as any type
	"json.RawMessage":  {"string", "", true},
	"*json.RawMessage": {"string", "", true},
	"[]byte":           {"string", "byte", true},
	"*[]byte":          {"string", "byte", true},
}

func convertToEventName(defType string, refType *reflect.Type) (string, string, bool) {
	// Handle interface{} and any type - should accept any value
	if defType == "interface{}" || defType == "*interface{}" {
		return "", "", true // No type constraint - accepts anything
	}

	// Check if type exists in our mapping
	if mapping, exists := typeMapping[defType]; exists {
		return mapping.jsonType, mapping.format, mapping.skipNested
	}

	// Handle maps - represent as objects
	if strings.HasPrefix(defType, "map[") {
		return "object", "", true
	}

	// Handle channels - not directly representable in JSON Schema
	if strings.HasPrefix(defType, "chan ") || strings.HasPrefix(defType, "<-chan ") || strings.HasPrefix(defType, "chan<- ") {
		return "string", "", true // Represent as string with channel info
	}

	// Handle functions - not directly representable in JSON Schema
	if strings.HasPrefix(defType, "func(") {
		return "string", "", true // Could represent as string description
	}

	// Handle complex types
	if refType != nil {
		f := *refType
		kind := f.Kind()

		// Handle interface type
		if kind == reflect.Interface {
			return "", "", true
		}

		if kind.String() != "ptr" {
			return convertToEventName(kind.String(), nil)
		}
		if f.Elem().Kind() != reflect.Struct {
			return convertToEventName(f.Elem().Kind().String(), nil)
		}
	}

	return defType, "", false
}
