package schematic

import (
	"encoding/json"
	"log"
	"os"
	"testing"

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

func TestGenerateProperties(t *testing.T) {
	got := GenerateProperties(EventToGenerate{})

	gotMarshal, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		log.Fatalf("error while marshaling schema: %s", err)
	}

	want, err := os.ReadFile("schema.json")
	if err != nil {
		log.Fatalf("error while reading schema.json: %s", err)
	}

	require.Equal(t, string(want), string(gotMarshal))
}
