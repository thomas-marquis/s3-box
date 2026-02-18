package directory_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func TestInMemoryFileObject_Read(t *testing.T) {
	t.Run("Read from empty buffer", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{Data: []byte("")}
		buf := make([]byte, 4)

		// When
		n, err := f.Read(buf)

		// Then
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
	})

	t.Run("Read from buffer with data", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{Data: []byte("test data")}
		buf := make([]byte, 4)

		// When
		n, err := f.Read(buf)

		// Then
		assert.Equal(t, 4, n)
		assert.Equal(t, "test", string(buf[:4]))
		assert.NoError(t, err)
	})

	t.Run("Read at end of buffer", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{Data: []byte("test data"), Pos: 9}
		buf := make([]byte, 4)

		// When
		n, err := f.Read(buf)

		// Then
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
	})
}

func TestInMemoryFileObject_Write(t *testing.T) {
	t.Run("Write to empty buffer", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{}
		data := []byte("test data")

		// When
		n, err := f.Write(data)

		// Then
		assert.Equal(t, len(data), n)
		assert.NoError(t, err)
		assert.Equal(t, data, f.Data)
	})

	t.Run("Write empty slice", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{Data: []byte{}}
		data := []byte("")

		// When
		n, err := f.Write(data)

		// Then
		assert.Equal(t, 0, n)
		assert.NoError(t, err)
		assert.Equal(t, []byte(""), f.Data)
	})

	t.Run("Write more data to existing buffer", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{Data: []byte("existing"), Pos: 8}
		moreData := []byte(" more data")

		// When
		n, err := f.Write(moreData)

		// Then
		assert.Equal(t, len(moreData), n)
		assert.NoError(t, err)
		assert.Equal(t, append([]byte("existing"), moreData...), f.Data)
	})
}

func TestInMemoryFileObject_Close(t *testing.T) {
	t.Run("Close file", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{}

		// When
		err := f.Close()

		// Then
		assert.NoError(t, err)
	})
}

func TestInMemoryFileObject_Seek(t *testing.T) {
	t.Run("Seek to start", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{Data: []byte("test data"), Pos: 0}

		// When
		pos, err := f.Seek(0, io.SeekStart)

		// Then
		assert.Equal(t, int64(0), pos)
		assert.NoError(t, err)
	})

	t.Run("Seek to current position", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{Data: []byte("test data"), Pos: 4}

		// When
		pos, err := f.Seek(0, io.SeekCurrent)

		// Then
		assert.Equal(t, int64(4), pos)
		assert.NoError(t, err)
	})

	t.Run("Seek to end", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{Data: []byte("test data"), Pos: 0}

		// When
		pos, err := f.Seek(0, io.SeekEnd)

		// Then
		assert.Equal(t, int64(9), pos)
		assert.NoError(t, err)
	})

	t.Run("Seek beyond end", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{Data: []byte("test data"), Pos: 0}

		// When
		pos, err := f.Seek(100, io.SeekStart)

		// Then
		assert.Equal(t, int64(100), pos)
		assert.NoError(t, err)
	})

	t.Run("Seek with invalid offset", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{Data: []byte("test data"), Pos: 0}

		// When
		_, err := f.Seek(-1, io.SeekStart)

		// Then
		assert.Error(t, err)
	})

	t.Run("Seek with invalid whence", func(t *testing.T) {
		// Given
		f := &directory.InMemoryFileObject{Data: []byte("test data"), Pos: 0}

		// When
		_, err := f.Seek(0, 42)

		// Then
		assert.Error(t, err)
	})
}
