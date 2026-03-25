package texteditor_test

//func TestFileEditor(t *testing.T) {
//	fyne_test.NewApp()
//
//	ctrl := gomock.NewController(t)
//	mockAppCtx := mocks_appcontext.NewMockAppContext(ctrl)
//	mockExplorerVM := mocks_viewmodel.NewMockExplorerViewModel(ctrl)
//	mockConnVM := mocks_viewmodel.NewMockConnectionViewModel(ctrl)
//	mockSettingsVM := mocks_viewmodel.NewMockSettingsViewModel(ctrl)
//
//	mockAppCtx.EXPECT().ExplorerViewModel().Return(mockExplorerVM).AnyTimes()
//	mockAppCtx.EXPECT().ConnectionViewModel().Return(mockConnVM).AnyTimes()
//	mockAppCtx.EXPECT().SettingsViewModel().Return(mockSettingsVM).AnyTimes()
//	mockAppCtx.EXPECT().Window().Return(fyne_test.NewWindow(nil)).AnyTimes()
//
//	lastModified := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
//	rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
//	file, _ := directory.NewFile("test.txt", rootDir,
//		directory.WithFileSize(1024),
//		directory.WithFileLastModified(lastModified),
//	)
//
//	t.Run("should display empty content when file is not loaded yet", func(t *testing.T) {
//		// Given
//		w := fyne_test.NewWindow(nil)
//		oe := &viewmodel.OpenedEditor{
//			Window:   w,
//			File:     file,
//			Content:  binding.NewString(),
//			IsLoaded: binding.NewBool(),
//			ErrorMsg: binding.NewString(),
//			OnSave:   func(string) error { return nil },
//		}
//
//		// When
//		res := widget.NewFileEditor(oe)
//		w.Canvas().SetContent(res)
//
//		// Then
//		fyne_test.AssertRendersToMarkup(t, "file_editor_not_loaded", w.Canvas())
//	})
//
//	t.Run("should display file content", func(t *testing.T) {
//		// Given
//		w := fyne_test.NewWindow(nil)
//		oe := &viewmodel.OpenedEditor{
//			Window:   w,
//			File:     file,
//			Content:  binding.NewString(),
//			IsLoaded: binding.NewBool(),
//			ErrorMsg: binding.NewString(),
//			OnSave:   func(string) error { return nil },
//		}
//
//		// When
//		res := widget.NewFileEditor(oe)
//		w.Canvas().SetContent(res)
//
//		oe.Content.Set("Hello world!") // nolint:errcheck
//		oe.IsLoaded.Set(true)          // nolint:errcheck
//
//		// Then
//		fyne_test.AssertRendersToMarkup(t, "file_editor_with_content", w.Canvas())
//	})
//
//	t.Run("should display error message", func(t *testing.T) {
//		// Given
//		w := fyne_test.NewWindow(nil)
//		oe := &viewmodel.OpenedEditor{
//			Window:   w,
//			File:     file,
//			Content:  binding.NewString(),
//			IsLoaded: binding.NewBool(),
//			ErrorMsg: binding.NewString(),
//			OnSave:   func(string) error { return nil },
//		}
//
//		// When
//		res := widget.NewFileEditor(oe)
//		w.Canvas().SetContent(res)
//
//		oe.ErrorMsg.Set("Error loading file") // nolint:errcheck
//		oe.IsLoaded.Set(true)                 // nolint:errcheck
//
//		// Then
//		fyne_test.AssertRendersToMarkup(t, "file_editor_with_error", w.Canvas())
//	})
//}
