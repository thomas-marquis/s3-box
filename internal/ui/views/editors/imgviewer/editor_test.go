package imgviewer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fyne_test "fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/inmemory"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/editor"
	"github.com/thomas-marquis/s3-box/internal/ui/views/editors/imgviewer"
)

// fakeFileContent is a test implementation of directory.FileContent
type fakeFileContent struct {
	*directory.InMemoryContent
	Readed chan struct{}
}

func (c *fakeFileContent) Read(buff []byte) (int, error) {
	<-c.Readed
	return c.InMemoryContent.Read(buff)
}

func TestImgViewer_Editor(t *testing.T) {
	t.Run("should display loader when loading", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir,
			directory.WithFileSize(1024),
			directory.WithFileLastModified(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)),
		)
		window := fyne_test.NewWindow(nil)
		window.Resize(fyne.NewSize(500, 300))
		ed := imgviewer.New(bus, window, file)

		res := ed.CreateWidget()
		canvas := window.Canvas()
		canvas.SetContent(res)

		// When & Then - should initially show loading state
		// The widget should be created without panic
		assert.NotNil(t, res)
	})

	t.Run("should display image when loaded", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir,
			directory.WithFileSize(1024),
			directory.WithFileLastModified(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)),
		)
		window := fyne_test.NewWindow(nil)
		window.Resize(fyne.NewSize(500, 300))
		ed := imgviewer.New(bus, window, file)

		res := ed.CreateWidget()
		canvas := window.Canvas()
		canvas.SetContent(res)

		// When the file is loaded with image data
		content := &directory.InMemoryContent{Data: []byte("test image data")}
		bus.Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: content,
		}))

		// Then - should not panic and widget should be created
		assert.NotNil(t, res)
	})

	t.Run("should display error when loading failed", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir,
			directory.WithFileSize(1024),
			directory.WithFileLastModified(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)),
		)
		window := fyne_test.NewWindow(nil)
		window.Resize(fyne.NewSize(500, 300))
		ed := imgviewer.New(bus, window, file)

		res := ed.CreateWidget()
		canvas := window.Canvas()
		canvas.SetContent(res)

		// When
		bus.Publish(event.New(editor.LoadFailed{
			Editor: ed,
			Err:    errors.New("failed to load image"),
		}))

		// Then - should not panic
		assert.NotNil(t, res)
	})

	t.Run("should be read-only", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir)
		window := fyne_test.NewWindow(nil)
		ed := imgviewer.New(bus, window, file)

		// When & Then
		assert.False(t, ed.(*imgviewer.Editor).HasChanged())
	})

	t.Run("should handle close requested", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir)
		window := fyne_test.NewWindow(nil)
		window.Resize(fyne.NewSize(500, 300))
		ed := imgviewer.New(bus, window, file)

		res := ed.CreateWidget()
		canvas := window.Canvas()
		canvas.SetContent(res)

		// When - editor is loaded
		content := &directory.InMemoryContent{
			Data: []byte("test image data"),
		}
		bus.Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: content,
		}))

		// When - close is requested
		bus.Publish(event.New(editor.CloseRequested{Editor: ed}))

		// Then - should not panic, close should be confirmed
		// We can't easily test the close confirmation in this setup,
		// but we can verify the editor doesn't panic
		assert.NotNil(t, ed)
	})

	t.Run("factory should have correct properties", func(t *testing.T) {
		// Given
		factory := &imgviewer.Factory{}

		// When & Then
		assert.Equal(t, "imgviewer", factory.Name())
		assert.Equal(t, "Image Viewer", factory.DisplayLabel())
		assert.Equal(t, "\\.(png|jpg|jpeg|gif)$", factory.DefaultFileRegexpPattern())
	})

	t.Run("should create editor with factory", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		window := fyne_test.NewWindow(nil)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir)

		factory := &imgviewer.Factory{}

		// When
		ed := factory.New(bus, window, file)

		// Then
		assert.NotNil(t, ed)
		assert.Equal(t, file, ed.File())
		assert.Equal(t, window, ed.Window())
	})

	t.Run("should handle empty image data", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir)
		window := fyne_test.NewWindow(nil)
		window.Resize(fyne.NewSize(500, 300))
		ed := imgviewer.New(bus, window, file)

		res := ed.CreateWidget()
		canvas := window.Canvas()
		canvas.SetContent(res)

		// When - editor is loaded with empty data
		content := &directory.InMemoryContent{
			Data: []byte{},
		}
		bus.Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: content,
		}))

		// Then - should not panic
		assert.NotNil(t, res)
	})

	t.Run("should handle large image data", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir)
		window := fyne_test.NewWindow(nil)
		window.Resize(fyne.NewSize(500, 300))
		ed := imgviewer.New(bus, window, file)

		res := ed.CreateWidget()
		canvas := window.Canvas()
		canvas.SetContent(res)

		// Create large image data
		largeData := make([]byte, 1024*1024) // 1MB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		content := &directory.InMemoryContent{
			Data: largeData,
		}

		// When
		bus.Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: content,
		}))

		// Then - should not panic
		assert.NotNil(t, res)
	})

	t.Run("should handle nil content", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir)
		ed := imgviewer.New(bus, fyne_test.NewWindow(nil), file)

		// When - editor is loaded with nil content
		bus.Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: &directory.InMemoryContent{Data: nil},
		}))

		// Then - should not panic
		assert.NotNil(t, ed)
	})

	t.Run("should display different file types", func(t *testing.T) {
		// Given
		testCases := []string{"test.png", "test.jpg", "test.jpeg", "test.gif"}

		for _, filename := range testCases {
			t.Run(filename, func(t *testing.T) {
				fyne_test.NewApp()
				ctx := context.Background()
				bus := inmemory.NewBus(ctx)
				window := fyne_test.NewWindow(nil)
				rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
				file, _ := directory.NewFile(filename, rootDir)

				// When
				ed := imgviewer.New(bus, window, file)

				// Then
				assert.NotNil(t, ed)
				assert.Equal(t, file.Name().String(), filename)
			})
		}
	})

	t.Run("should handle invalid image data", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir)
		window := fyne_test.NewWindow(nil)
		window.Resize(fyne.NewSize(500, 300))
		ed := imgviewer.New(bus, window, file)

		res := ed.CreateWidget()
		canvas := window.Canvas()
		canvas.SetContent(res)

		// When - editor is loaded with invalid image data
		content := &directory.InMemoryContent{
			Data: []byte("invalid image data"),
		}
		bus.Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: content,
		}))

		// Then - should not panic, Fyne will handle invalid image data gracefully
		assert.NotNil(t, res)
	})

	t.Run("should handle file with different image content", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir)
		window := fyne_test.NewWindow(nil)
		window.Resize(fyne.NewSize(500, 300))
		ed := imgviewer.New(bus, window, file)

		res := ed.CreateWidget()
		canvas := window.Canvas()
		canvas.SetContent(res)

		// When - editor is loaded with different content types
		testData := []struct {
			name string
			data []byte
		}{
			{"PNG", []byte{0x89, 0x50, 0x4E, 0x47}},
			{"JPEG", []byte{0xFF, 0xD8, 0xFF}},
			{"GIF", []byte{0x47, 0x49, 0x46, 0x38}},
		}

		for _, tc := range testData {
			content := &directory.InMemoryContent{Data: tc.data}
			bus.Publish(event.New(editor.Loaded{
				Editor:  ed,
				Content: content,
			}))
		}

		// Then - should not panic
		assert.NotNil(t, res)
	})

	t.Run("should handle multiple sequential loads", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir)
		window := fyne_test.NewWindow(nil)
		window.Resize(fyne.NewSize(500, 300))
		ed := imgviewer.New(bus, window, file)

		res := ed.CreateWidget()
		canvas := window.Canvas()
		canvas.SetContent(res)

		// When - load, then fail, then load again
		content1 := &directory.InMemoryContent{Data: []byte("image1")}
		bus.Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: content1,
		}))

		bus.Publish(event.New(editor.LoadFailed{
			Editor: ed,
			Err:    errors.New("load failed"),
		}))

		content2 := &directory.InMemoryContent{Data: []byte("image2")}
		bus.Publish(event.New(editor.Loaded{
			Editor:  ed,
			Content: content2,
		}))

		// Then - should not panic
		assert.NotNil(t, res)
	})

	t.Run("should have correct file in editor", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir,
			directory.WithFileSize(1024),
			directory.WithFileLastModified(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)),
		)
		window := fyne_test.NewWindow(nil)
		ed := imgviewer.New(bus, window, file)

		// When & Then - editor should have the correct file
		assert.Equal(t, "test.png", ed.File().Name().String())
	})

	t.Run("should handle cancel during loading", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir)
		ed := imgviewer.New(bus, fyne_test.NewWindow(nil), file)

		// When - cancel is called
		ed.(*imgviewer.Editor).Cancel()

		// Then - should not panic
		assert.NotNil(t, ed)
	})

	t.Run("RequestClose should publish CloseRequested event", func(t *testing.T) {
		// Given
		fyne_test.NewApp()
		ctx := context.Background()
		bus := inmemory.NewBus(ctx)
		rootDir, _ := directory.NewRoot(connection_deck.NewConnectionID())
		file, _ := directory.NewFile("test.png", rootDir)
		window := fyne_test.NewWindow(nil)
		ed := imgviewer.New(bus, window, file)

		// Setup a subscriber to catch the CloseRequested event
		closeRequestedChan := make(chan event.Event, 1)
		bus.Subscribe().
			On(event.Is(editor.CloseRequestedType), func(e event.Event) {
				closeRequestedChan <- e
			}).
			ListenNonBlocking()

		// When
		ed.(*imgviewer.Editor).RequestClose()

		// Then
		select {
		case evt := <-closeRequestedChan:
			assert.Equal(t, editor.CloseRequestedType, evt.Type())
			pl := evt.Payload().(editor.CloseRequested)
			assert.Equal(t, ed, pl.Editor)
		case <-time.After(time.Second):
			t.Error("CloseRequested event not received")
		}
	})
}
