package uiutils_test

import (
	"testing"

	"fyne.io/fyne/v2/data/binding"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/ui/uiutils"
)

type testStruct struct {
	Value string
}

func Test_GetUntypedListOrPanic_ShouldReturnListWithStructValue(t *testing.T) {
	// Given
	fyne_test.NewTempApp(t)

	data := binding.NewUntypedList()
	data.Append(testStruct{Value: "test1"})
	data.Append(testStruct{Value: "test2"})

	expected := []testStruct{
		{Value: "test1"},
		{Value: "test2"},
	}

	// When
	result := uiutils.GetUntypedListOrPanic[testStruct](data)

	// Then
	assert.Len(t, result, 2, "Expected 2 items in the list")
	assert.EqualValues(t, expected, result, "Expected the list to match the expected values")
}

func Test_GetUntypedListOrPanic_ShouldReturnListWithStructPointer(t *testing.T) {
	// Given
	fyne_test.NewTempApp(t)

	d1 := testStruct{Value: "test1"}
	d2 := testStruct{Value: "test2"}

	data := binding.NewUntypedList()
	data.Append(&d1)
	data.Append(&d2)

	expected := []*testStruct{
		&d1, &d2,
	}

	// When
	result := uiutils.GetUntypedListOrPanic[*testStruct](data)

	// Then
	assert.Len(t, result, 2, "Expected 2 items in the list")
	assert.EqualValues(t, expected, result, "Expected the list to match the expected values")
}

func Test_GetUntypedListOrPanic_ShouldPanicOnInvalidType(t *testing.T) {
	// Given
	fyne_test.NewTempApp(t)

	data := binding.NewUntypedList()
	data.Append("test1")
	data.Append("test2")

	// When/Then
	assert.PanicsWithValue(t, "Invalid casting type for binding.UntypedList", func() {
		uiutils.GetUntypedListOrPanic[testStruct](data)
	}, "Expected panic due to invalid type conversion")
}
