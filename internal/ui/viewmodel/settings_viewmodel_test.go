package viewmodel_test

import (
	"context"
	"errors"
	"testing"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/it-happened/inmemory"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/state"
	"github.com/thomas-marquis/s3-box/internal/ui/viewmodel"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	mock_editor "github.com/thomas-marquis/s3-box/mocks/editor"
	mocks_notification "github.com/thomas-marquis/s3-box/mocks/notification"
	"go.uber.org/mock/gomock"
)

func TestSettingsViewModel_LoadEditorMappings(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should load default editors when no mappings exist", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)

		mockEditorMappingsRepo := mock_editor.NewMockMappingRepository(ctrl)
		mockEditorMappingsRepo.EXPECT().GetAll().Return([]editor.Mapping{}, nil)
		mockEditorMappingsRepo.EXPECT().SaveAll(gomock.Any()).Return(nil)

		appState := state.New()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)

		// When
		mockNotifier := mocks_notification.NewMockRepository(ctrl)
		_ = viewmodel.NewSettingsViewModel(
			fyne.CurrentApp().Settings(),
			fyne.CurrentApp().Preferences(),
			mockNotifier,
			appState,
			bus,
			mockEditorMappingsRepo)

		// Then - default editors should be registered
		selector := appState.Editors().Selector()
		registeredEditors := selector.RegisteredEditors()
		assert.Len(t, registeredEditors, 3) // text, csv, and imgviewer

		// Check that default mappings were created
		mappings := selector.Mappings()
		assert.Len(t, mappings, 3)
	})

	t.Run("should load saved mappings from repository", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)

		savedMappings := []editor.Mapping{
			{RegexpPattern: `\.(txt|md)$`, EditorName: "text"},
			{RegexpPattern: `\.csv$`, EditorName: "csv"},
		}

		mockEditorMappingsRepo := mock_editor.NewMockMappingRepository(ctrl)
		mockEditorMappingsRepo.EXPECT().GetAll().Return(savedMappings, nil)

		appState := state.New()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)

		// When
		mockNotifier := mocks_notification.NewMockRepository(ctrl)
		_ = viewmodel.NewSettingsViewModel(
			fyne.CurrentApp().Settings(),
			fyne.CurrentApp().Preferences(),
			mockNotifier,
			appState,
			bus,
			mockEditorMappingsRepo)

		// Then - saved mappings should be loaded
		selector := appState.Editors().Selector()
		mappings := selector.Mappings()
		assert.Len(t, mappings, 2)
		assert.True(t, editor.CompareMappings(mappings[0], savedMappings[0]))
		assert.True(t, editor.CompareMappings(mappings[1], savedMappings[1]))
	})

	t.Run("should handle error from repository", func(t *testing.T) {
		// Given
		expectedErr := errors.New("load failed")
		ctrl := gomock.NewController(t)

		mockEditorMappingsRepo := mock_editor.NewMockMappingRepository(ctrl)
		mockEditorMappingsRepo.EXPECT().GetAll().Return(nil, expectedErr)

		appState := state.New()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)

		// When
		mockNotifier := mocks_notification.NewMockRepository(ctrl)
		_ = viewmodel.NewSettingsViewModel(
			fyne.CurrentApp().Settings(),
			fyne.CurrentApp().Preferences(),
			mockNotifier,
			appState,
			bus,
			mockEditorMappingsRepo)

		// Then - default editors should still be registered even if loading mappings fails
		selector := appState.Editors().Selector()
		registeredEditors := selector.RegisteredEditors()
		assert.Len(t, registeredEditors, 3) // text, csv, and imgviewer
	})
}

func TestSettingsViewModel_SaveEditorSelectors(t *testing.T) {
	fyne_test.NewApp()

	t.Run("should save editor mappings successfully", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)

		mockEditorMappingsRepo := mock_editor.NewMockMappingRepository(ctrl)
		mockEditorMappingsRepo.EXPECT().GetAll().Return([]editor.Mapping{}, nil)
		mockEditorMappingsRepo.EXPECT().SaveAll(gomock.Any()).Return(nil).Times(2)

		appState := state.New()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)

		mockNotifier := mocks_notification.NewMockRepository(ctrl)
		vm := viewmodel.NewSettingsViewModel(
			fyne.CurrentApp().Settings(),
			fyne.CurrentApp().Preferences(),
			mockNotifier,
			appState,
			bus,
			mockEditorMappingsRepo)

		// Add some mappings
		selector := appState.Editors().Selector()
		u.Skip(selector.RegisterMapping("text", `\.txt$`))

		// When & Then
		assert.NotPanics(t, func() {
			vm.SaveEditorMappings()
		})
	})

	t.Run("should notify error when repository fails", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)

		expectedErr := errors.New("save failed")
		mockEditorMappingsRepo := mock_editor.NewMockMappingRepository(ctrl)
		mockEditorMappingsRepo.EXPECT().GetAll().Return([]editor.Mapping{}, nil)
		mockEditorMappingsRepo.EXPECT().SaveAll(gomock.Any()).Return(expectedErr).
			After(mockEditorMappingsRepo.EXPECT().SaveAll(gomock.Any()).Return(nil))

		appState := state.New()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)

		mockNotifier := mocks_notification.NewMockRepository(ctrl)
		mockNotifier.EXPECT().NotifyError(expectedErr)
		vm := viewmodel.NewSettingsViewModel(
			fyne.CurrentApp().Settings(),
			fyne.CurrentApp().Preferences(),
			mockNotifier,
			appState,
			bus,
			mockEditorMappingsRepo)

		// When
		vm.SaveEditorMappings()
	})
}
