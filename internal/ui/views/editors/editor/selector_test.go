package editor_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	mock_editor "github.com/thomas-marquis/s3-box/mocks/editor"
	"go.uber.org/mock/gomock"
)

func TestSelector_RegisterEditor(t *testing.T) {
	t.Run("should register a single editor factory", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()

		// When
		result := selector.RegisterEditor(mockFactory)

		// Then
		assert.Same(t, selector, result)
		assert.Len(t, selector.RegisteredEditors(), 1)
	})

	t.Run("should register multiple editor factories", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory1 := mock_editor.NewMockFactory(ctrl)
		mockFactory2 := mock_editor.NewMockFactory(ctrl)
		mockFactory1.EXPECT().Name().Return("text").AnyTimes()
		mockFactory2.EXPECT().Name().Return("csv").AnyTimes()

		selector := editor.NewSelector()

		// When
		selector.RegisterEditor(mockFactory1)
		selector.RegisterEditor(mockFactory2)

		// Then
		assert.Len(t, selector.RegisteredEditors(), 2)
	})

	t.Run("should retrieve registered editors via RegisteredEditors", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()

		// When
		selector.RegisterEditor(mockFactory)
		editors := selector.RegisteredEditors()

		// Then
		assert.Len(t, editors, 1)
		assert.Equal(t, mockFactory, editors[0])
	})

	t.Run("should allow duplicate registration without issues", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()

		// When
		selector.RegisterEditor(mockFactory)
		selector.RegisterEditor(mockFactory)

		// Then
		assert.Len(t, selector.RegisteredEditors(), 1)
	})
}

func TestSelector_RegisterMapping(t *testing.T) {
	t.Run("should register a valid mapping when editor exists", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)

		// When
		err := selector.RegisterMapping("text", `\.txt$`)

		// Then
		assert.NoError(t, err)
		assert.Len(t, selector.Mappings(), 1)
	})

	t.Run("should return ErrEditorNotRegistered when editor does not exist", func(t *testing.T) {
		// Given
		selector := editor.NewSelector()

		// When
		err := selector.RegisterMapping("nonexistent", `\.txt$`)

		// Then
		assert.ErrorIs(t, err, editor.ErrEditorNotRegistered)
	})

	t.Run("should register multiple mappings", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory1 := mock_editor.NewMockFactory(ctrl)
		mockFactory2 := mock_editor.NewMockFactory(ctrl)
		mockFactory1.EXPECT().Name().Return("text").AnyTimes()
		mockFactory2.EXPECT().Name().Return("csv").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory1)
		selector.RegisterEditor(mockFactory2)

		// When
		u.Skip(selector.RegisterMapping("text", `\.txt$`))
		u.Skip(selector.RegisterMapping("csv", `\.csv$`))

		// Then
		assert.Len(t, selector.Mappings(), 2)
	})

	t.Run("should store mappings correctly", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)

		// When
		u.Skip(selector.RegisterMapping("text", `\.txt$`))
		mappings := selector.Mappings()

		// Then
		assert.Len(t, mappings, 1)
		assert.Equal(t, `\.txt$`, mappings[0].RegexpPattern)
		assert.Equal(t, "text", mappings[0].EditorName)
	})
}

func TestSelector_UpdateMapping(t *testing.T) {
	t.Run("should update an existing mapping", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)
		u.Skip(selector.RegisterMapping("text", `\.txt$`))

		// When
		err := selector.UpdateMapping("text", `\.txt$`, `\.text$`)

		// Then
		assert.NoError(t, err)
		mappings := selector.Mappings()
		assert.Len(t, mappings, 1)
		assert.Equal(t, `\.text$`, mappings[0].RegexpPattern)
		assert.Equal(t, "text", mappings[0].EditorName)
	})

	t.Run("should add new mapping when updating non-existent mapping", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)

		// When
		err := selector.UpdateMapping("text", `\.txt$`, `\.text$`)

		// Then
		assert.NoError(t, err)
		assert.Len(t, selector.Mappings(), 1)
		mappings := selector.Mappings()
		assert.Equal(t, `\.text$`, mappings[0].RegexpPattern)
	})

	t.Run("should return ErrEditorNotRegistered when updating with invalid editor name", func(t *testing.T) {
		// Given
		selector := editor.NewSelector()

		// When
		err := selector.UpdateMapping("nonexistent", `\.txt$`, `\.text$`)

		// Then
		assert.ErrorIs(t, err, editor.ErrEditorNotRegistered)
	})
}

func TestSelector_DeleteMapping(t *testing.T) {
	t.Run("should delete an existing mapping", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)
		u.Skip(selector.RegisterMapping("text", `\.txt$`))

		// When
		err := selector.DeleteMapping(editor.Mapping{RegexpPattern: `\.txt$`, EditorName: "text"})

		// Then
		assert.NoError(t, err)
		assert.Len(t, selector.Mappings(), 0)
	})

	t.Run("should succeed silently when deleting non-existent mapping", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)

		// When
		err := selector.DeleteMapping(editor.Mapping{RegexpPattern: `\.txt$`, EditorName: "text"})

		// Then
		assert.NoError(t, err)
		assert.Len(t, selector.Mappings(), 0)
	})

	t.Run("should return ErrEditorNotRegistered when editor does not exist", func(t *testing.T) {
		// Given
		selector := editor.NewSelector()

		// When
		err := selector.DeleteMapping(editor.Mapping{RegexpPattern: `\.txt$`, EditorName: "nonexistent"})

		// Then
		assert.ErrorIs(t, err, editor.ErrEditorNotRegistered)
	})

	t.Run("should remove deleted mappings from the list", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)
		u.Skip(selector.RegisterMapping("text", `\.txt$`))
		u.Skip(selector.RegisterMapping("text", `\.md$`))

		// When
		u.Skip(selector.DeleteMapping(editor.Mapping{RegexpPattern: `\.txt$`, EditorName: "text"}))

		// Then
		mappings := selector.Mappings()
		assert.Len(t, mappings, 1)
		assert.Equal(t, `\.md$`, mappings[0].RegexpPattern)
	})
}

func TestSelector_Mappings(t *testing.T) {
	t.Run("should return mappings in correct order by pattern length ascending", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)

		// When
		u.Skip(selector.RegisterMapping("text", `\.txt$`))
		u.Skip(selector.RegisterMapping("text", `\.txt\.backup$`))
		u.Skip(selector.RegisterMapping("text", `\.t$`))

		// Then
		mappings := selector.Mappings()
		assert.Len(t, mappings, 3)
		assert.Equal(t, `\.t$`, mappings[0].RegexpPattern)
		assert.Equal(t, `\.txt$`, mappings[1].RegexpPattern)
		assert.Equal(t, `\.txt\.backup$`, mappings[2].RegexpPattern)
	})

	t.Run("should return empty slice when no mappings exist", func(t *testing.T) {
		// Given
		selector := editor.NewSelector()

		// When & Then
		assert.Empty(t, selector.Mappings())
	})
}

func TestSelector_Select(t *testing.T) {
	t.Run("should select editor for file path that matches pattern", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)
		u.Skip(selector.RegisterMapping("text", `\.txt$`))

		// When
		factory, err := selector.Select("test.txt")

		// Then
		assert.NoError(t, err)
		assert.Equal(t, mockFactory, factory)
	})

	t.Run("should return ErrNoMatchingEditor when no pattern matches", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)
		u.Skip(selector.RegisterMapping("text", `\.txt$`))

		// When
		_, err := selector.Select("test.csv")

		// Then
		assert.ErrorIs(t, err, editor.ErrNoMatchingEditor)
	})

	t.Run("should use most specific pattern when multiple patterns match", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory1 := mock_editor.NewMockFactory(ctrl)
		mockFactory2 := mock_editor.NewMockFactory(ctrl)
		mockFactory1.EXPECT().Name().Return("text").AnyTimes()
		mockFactory2.EXPECT().Name().Return("csv").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory1)
		selector.RegisterEditor(mockFactory2)

		// Register mappings - csv is more specific
		u.Skip(selector.RegisterMapping("csv", `\.csv$`))
		u.Skip(selector.RegisterMapping("text", `.*`))

		// When
		factory, err := selector.Select("test.csv")

		// Then
		assert.NoError(t, err)
		assert.Equal(t, mockFactory1, factory)
	})

	t.Run("should return ErrNoMatchingEditor when editor was unregistered after mapping was created", func(t *testing.T) {
		// Given
		selector := editor.NewSelector()
		u.Skip(selector.RegisterMapping("nonexistent", `\.txt$`))

		// When
		_, err := selector.Select("test.txt")

		// Then
		assert.ErrorIs(t, err, editor.ErrNoMatchingEditor)
	})
}

func TestSelector_GetRegisteredEditorByName(t *testing.T) {
	t.Run("should get registered editor by name", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)

		// When
		factory, err := selector.GetRegisteredEditorByName("text")

		// Then
		assert.NoError(t, err)
		assert.Equal(t, mockFactory, factory)
	})

	t.Run("should return ErrEditorNotRegistered when editor does not exist", func(t *testing.T) {
		// Given
		selector := editor.NewSelector()

		// When
		_, err := selector.GetRegisteredEditorByName("nonexistent")

		// Then
		assert.ErrorIs(t, err, editor.ErrEditorNotRegistered)
	})
}

func TestSelector_MappingsObservable(t *testing.T) {
	t.Run("should update observable value when mappings change", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)

		observable := selector.MappingsObservable()
		initialMappings := observable.Get()
		require.Empty(t, initialMappings)

		// When
		u.Skip(selector.RegisterMapping("text", `\.txt$`))

		// Then
		updatedMappings := observable.Get()
		assert.Len(t, updatedMappings, 1)
		assert.Equal(t, `\.txt$`, updatedMappings[0].RegexpPattern)
	})

	t.Run("should notify listeners on mapping changes", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockFactory := mock_editor.NewMockFactory(ctrl)
		mockFactory.EXPECT().Name().Return("text").AnyTimes()

		selector := editor.NewSelector()
		selector.RegisterEditor(mockFactory)

		observable := selector.MappingsObservable()
		var receivedMappings []editor.Mapping
		observeFunc := func(mappings []editor.Mapping) {
			receivedMappings = mappings
		}

		// Subscribe to changes
		observable.Observe(observeFunc)

		// When
		u.Skip(selector.RegisterMapping("text", `\.txt$`))

		// Then
		assert.Len(t, receivedMappings, 1)
		assert.Equal(t, `\.txt$`, receivedMappings[0].RegexpPattern)
	})
}

func TestSelector_NewSelector(t *testing.T) {
	t.Run("should create new selector with initialized fields", func(t *testing.T) {
		// Given & When
		selector := editor.NewSelector()

		// Then
		assert.NotNil(t, selector)
		assert.Empty(t, selector.RegisteredEditors())
		assert.Empty(t, selector.Mappings())
		assert.NotNil(t, selector.MappingsObservable())
	})
}

func TestCompareMappings(t *testing.T) {
	t.Run("should return true for equal mappings", func(t *testing.T) {
		// Given & When & Then
		mapping1 := editor.Mapping{RegexpPattern: `\.txt$`, EditorName: "text"}
		mapping2 := editor.Mapping{RegexpPattern: `\.txt$`, EditorName: "text"}

		assert.True(t, editor.CompareMappings(mapping1, mapping2))
	})

	t.Run("should return false for different mappings", func(t *testing.T) {
		// Given & When & Then
		mapping1 := editor.Mapping{RegexpPattern: `\.txt$`, EditorName: "text"}
		mapping2 := editor.Mapping{RegexpPattern: `\.csv$`, EditorName: "csv"}

		assert.False(t, editor.CompareMappings(mapping1, mapping2))
	})

	t.Run("should return false when only pattern differs", func(t *testing.T) {
		// Given & When & Then
		mapping1 := editor.Mapping{RegexpPattern: `\.txt$`, EditorName: "text"}
		mapping2 := editor.Mapping{RegexpPattern: `\.md$`, EditorName: "text"}

		assert.False(t, editor.CompareMappings(mapping1, mapping2))
	})

	t.Run("should return false when only editor name differs", func(t *testing.T) {
		// Given & When & Then
		mapping1 := editor.Mapping{RegexpPattern: `\.txt$`, EditorName: "text"}
		mapping2 := editor.Mapping{RegexpPattern: `\.txt$`, EditorName: "csv"}

		assert.False(t, editor.CompareMappings(mapping1, mapping2))
	})
}
