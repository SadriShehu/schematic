package schematic

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type StringAliasType string

type SimpleStruct struct {
	FieldString      string   `json:"field_string"`
	FieldStringPtr   *string  `json:"field_string_ptr"`
	FieldStringSlice []string `json:"field_string_slice"`
	FieldInt         int      `json:"field_int"`
	FieldInt32       int32    `json:"field_int32"`
	FieldInt64       int64    `json:"field_int64"`
	FieldFloat32     float32  `json:"field_float32"`
	FieldFloat64     float64  `json:"field_float64"`
	FieldOptional    string   `json:"field_optional,omitempty"`
}

type MainStruct struct {
	IntSlice        []int            `json:"int_slice"`
	IntSlicePtr1    []*int           `json:"int_slice_ptr1"`
	IntSlicePtr2    *[]int           `json:"int_slice_ptr2"`
	Alias           StringAliasType  `json:"alias"`
	AliasPtr        *StringAliasType `json:"alias_ptr"`
	StructSimple    SimpleStruct     `json:"struct_simple"`
	StructPtr       *SimpleStruct    `json:"struct_ptr"`
	SliceStruct     []SimpleStruct   `json:"slice_struct"`
	SliceStructPtr1 []*SimpleStruct  `json:"slice_struct_ptr1"`
	SliceStructPtr2 *[]SimpleStruct  `json:"slice_struct_ptr2"`
}

type EventTags struct {
	EventName    string `json:"event_name"`
	EventVersion string `json:"event_version"`
	EventID      string `json:"event_id"`
}

type EventToGenerate struct {
	Tags EventTags  `json:"tags"`
	Body MainStruct `json:"main_struct"`
}

type EdgeCaseStruct struct {
	MapField       map[string]any  `json:"map_field"`
	InterfaceField any             `json:"interface_field"`
	TimeField      time.Time       `json:"time_field"`
	ByteSlice      []byte          `json:"byte_slice"`
	JSONRaw        json.RawMessage `json:"json_raw"`
	SkippedField   string          `json:"-"`
	UintField      uint64          `json:"uint_field"`
}

type RecursiveStruct struct {
	Name     string            `json:"name"`
	Children []RecursiveStruct `json:"children,omitempty"`
}

func TestGenerateSchema(t *testing.T) {
	schema := GenerateSchema(EventToGenerate{}, "Test Event", "http://json-schema.org/draft-07/schema#")

	require.Equal(t, "http://json-schema.org/draft-07/schema#", schema.Schema)
	require.Equal(t, "Test Event", schema.Title)
	require.Equal(t, "object", schema.Type)
	require.Contains(t, schema.Properties, "tags")
	require.Contains(t, schema.Properties, "main_struct")
}

func TestEdgeCases(t *testing.T) {
	properties := GenerateProperties(EdgeCaseStruct{})

	// Test map field
	require.Contains(t, properties, "map_field")
	require.Equal(t, "object", properties["map_field"].Type)

	// Test interface{} field - should accept any type
	require.Contains(t, properties, "interface_field")
	require.Equal(t, "", properties["interface_field"].Type) // No type constraint

	// Test time field
	require.Contains(t, properties, "time_field")
	require.Equal(t, "string", properties["time_field"].Type)
	require.Equal(t, "date-time", properties["time_field"].Format)

	// Test byte slice
	require.Contains(t, properties, "byte_slice")
	require.Equal(t, "string", properties["byte_slice"].Type)
	require.Equal(t, "byte", properties["byte_slice"].Format)

	// Test skipped field should not be present
	require.NotContains(t, properties, "skipped_field")

	// Test uint field
	require.Contains(t, properties, "uint_field")
	require.Equal(t, "integer", properties["uint_field"].Type)
}

func TestRecursiveStructs(t *testing.T) {
	properties := GenerateProperties(RecursiveStruct{})

	require.Contains(t, properties, "name")
	require.Contains(t, properties, "children")

	// Children should be an array
	require.Equal(t, "array", properties["children"].Type)
	require.NotNil(t, properties["children"].Items)
}

func TestSnakeCaseConversion(t *testing.T) {
	type TestStruct struct {
		CamelCaseField  string `json:"explicit_json_tag"`
		PascalCaseField string // Should become pascal_case_field
		SimpleField     string // Should become simple_field
		XMLHttpRequest  string // Should become x_m_l_http_request
		IOHandler       string // Should become i_o_handler
	}

	properties := GenerateProperties(TestStruct{})

	// Field with explicit JSON tag should use the tag
	require.Contains(t, properties, "explicit_json_tag")

	// Fields without JSON tags should be converted to snake_case
	require.Contains(t, properties, "pascal_case_field")
	require.Contains(t, properties, "simple_field")
	require.Contains(t, properties, "x_m_l_http_request")
	require.Contains(t, properties, "i_o_handler")

	// Make sure the original PascalCase names are NOT present
	require.NotContains(t, properties, "PascalCaseField")
	require.NotContains(t, properties, "SimpleField")
	require.NotContains(t, properties, "XMLHttpRequest")
	require.NotContains(t, properties, "IOHandler")
}

func TestGenerateRequired(t *testing.T) {
	// Test with MainStruct
	required := GenerateRequired(MainStruct{}, nil)

	// Only non-pointer fields without omitempty should be required
	expectedRequired := []string{"alias", "struct_simple"}
	require.ElementsMatch(t, expectedRequired, required)

	// Test with EventTags
	tagsRequired := GenerateRequired(EventTags{}, nil)
	expectedTagsRequired := []string{"event_name", "event_version", "event_id"}
	require.ElementsMatch(t, expectedTagsRequired, tagsRequired)
}
