package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"
)

// Object implements directory.FileContent for S3 objects using a state pattern.
// It manages the lifecycle of an S3 object, transitioning between states based on
// whether the object exists in S3 or not.
type Object struct {
	client s3client.Client
	file   *directory.File

	currentState s3ObjectState
}

var (
	_ directory.FileContent = (*Object)(nil)
)

// NewObject creates a new Object and initializes its state based on
// whether the object exists in S3. If the object exists, it downloads the content
// and initializes the state with it. If not, it starts in a non-existent state.
func NewObject(ctx context.Context, client s3client.Client, file *directory.File) (*Object, error) {
	obj := &Object{
		file:   file,
		client: client,
	}

	// Check if an object exists to determine the initial state
	buff := types.NewWriteAtBuffer([]byte{})
	key := buildS3Key(file)
	if err := client.Download(ctx, key, buff); err != nil {
		if isNotFoundError(err) {
			obj.setState(&s3ObjectNotExists{obj: obj})
		} else {
			return nil, fmt.Errorf("failed to check object existence: %w", err)
		}
	} else {
		b := buff.Bytes()
		obj.setState(&s3ObjectExists{obj: obj, content: b, position: int64(len(b))})
	}

	return obj, nil
}

// Read delegates to the current state's Read implementation
func (o *Object) Read(p []byte) (n int, err error) {
	return o.currentState.Read(p)
}

// Write delegates to the current state's Write implementation
func (o *Object) Write(p []byte) (n int, err error) {
	return o.currentState.Write(p)
}

// Close delegates to the current state's Close implementation
func (o *Object) Close() error {
	return o.currentState.Close()
}

func (o *Object) Seek(offset int64, whence int) (int64, error) {
	return o.currentState.Seek(offset, whence)
}

func (o *Object) Cancel() {
	o.currentState.Cancel()
}

func (o *Object) setState(state s3ObjectState) {
	o.currentState = state
}

// buildS3Key constructs the S3 key from the file's directory path and name
func buildS3Key(file *directory.File) string {
	path := file.DirectoryPath()
	if path == directory.RootPath {
		return string(file.Name())
	}
	return path.String()[1:] + string(file.Name())
}

// s3ObjectState represents the state interface for Object.
// Each state implements different behavior for Read, Write, and Close operations.
type s3ObjectState directory.FileContent

var (
	_ s3ObjectState = (*s3ObjectNotExists)(nil)
	_ s3ObjectState = (*s3ObjectExists)(nil)
	_ s3ObjectState = (*s3ObjectLazyReadOnly)(nil)
)

type withCancelCbs struct {
	cancelCbs []func()
}

func (c *withCancelCbs) Cancel() {
	for _, cb := range c.cancelCbs {
		cb()
	}
}

func (c *withCancelCbs) addCallback(cb func()) {
	c.cancelCbs = append(c.cancelCbs, cb)
}

// s3ObjectNotExists represents the state when the S3 object does not exist.
// In this state, reads will fail and writes will create the object and transition to the existing state.
type s3ObjectNotExists struct {
	withCancelCbs

	obj    *Object
	buffer *bytes.Buffer
}

// Read returns an error since the object doesn't exist
func (s *s3ObjectNotExists) Read(_ []byte) (n int, err error) {
	return 0, fmt.Errorf("object does not exist: %s", s.obj.file.Name())
}

// Write buffers the content and uploads it to S3, then transitions to exist state
func (s *s3ObjectNotExists) Write(p []byte) (n int, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	s.addCallback(cancel)

	if s.buffer == nil {
		s.buffer = new(bytes.Buffer)
	}

	n, err = s.buffer.Write(p)
	if err != nil {
		return n, fmt.Errorf("failed to buffer content: %w", err)
	}

	key := buildS3Key(s.obj.file)

	if err := s.obj.client.Upload(ctx, key, bytes.NewReader(s.buffer.Bytes())); err != nil {
		return n, fmt.Errorf("failed to upload object: %w", err)
	}

	s.obj.setState(&s3ObjectExists{
		obj:      s.obj,
		content:  s.buffer.Bytes(),
		position: int64(n),
	})

	return n, nil
}

// Close is a no-op for non-existent objects
func (s *s3ObjectNotExists) Close() error {
	return nil
}

func (s *s3ObjectNotExists) Seek(_ int64, _ int) (int64, error) {
	return 0, errors.New("cannot seek on non-existent object")
}

// s3ObjectExists represents the state when the S3 object exists.
// In this state, reads stream from the downloaded content and writes append and re-upload.
type s3ObjectExists struct {
	withCancelCbs

	obj      *Object
	content  []byte
	position int64
}

// Read reads from the downloaded content, advancing the position
func (s *s3ObjectExists) Read(p []byte) (n int, err error) {
	if s.position >= int64(len(s.content)) {
		return 0, io.EOF
	}

	n = copy(p, s.content[s.position:])
	s.position += int64(n)

	return n, nil
}

func (s *s3ObjectExists) Write(p []byte) (n int, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	s.addCallback(cancel)

	endPos := s.position + int64(len(p))
	initialContentLen := len(s.content)

	if endPos > int64(initialContentLen) {
		s.content = append(s.content, make([]byte, endPos-int64(initialContentLen))...)
	}

	truncLen := int64(initialContentLen) - s.position
	truncatedParts := make([]byte, truncLen)
	copy(truncatedParts, s.content[s.position:initialContentLen])

	copy(s.content[s.position:], p)
	s.content = s.content[:endPos]

	key := buildS3Key(s.obj.file)
	s.position = 0 // reset the cursor to let the sdk reads the entier content

	if err := s.obj.client.Upload(ctx, key, s); err != nil {
		s.position = endPos - int64(len(p))
		s.content = append(s.content[:s.position], truncatedParts...)
		return 0, fmt.Errorf("failed to upload updated content: %w", err)
	}

	s.position = endPos

	return len(p), nil
}

func (s *s3ObjectExists) Close() error {
	return nil
}

func (s *s3ObjectExists) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		s.position = offset
	case io.SeekCurrent:
		newPos := s.position + offset
		if newPos < 0 {
			return 0, errors.New("cannot seek before beginning of file")
		}
		if newPos > int64(len(s.content)) {
			s.position = int64(len(s.content))
			return s.position, io.EOF
		}
		s.position += offset
	case io.SeekEnd:
		newPos := int64(len(s.content)) + offset
		if newPos < 0 {
			return 0, errors.New("cannot seek before beginning of file")
		}
		if newPos > int64(len(s.content)) {
			s.position = int64(len(s.content))
			return s.position, io.EOF
		}
		s.position = newPos

	default:
		return 0, errors.New("invalid whence")
	}

	return s.position, nil
}

// s3ObjectLazyReadOnly represents a lazy-loaded, read-only S3 object.
// It fetches data in ranges from S3 on-demand and caches all loaded parts.
type s3ObjectLazyReadOnly struct {
	withCancelCbs

	obj          *Object
	position     int64
	totalSize    int64
	nextFetchPos int64
	cache        map[int64][]byte
}

// NewLazyObject creates a new Object in lazy read-only mode.
func NewLazyObject(ctx context.Context, client s3client.Client, file *directory.File) (*Object, error) {
	obj := &Object{
		file:   file,
		client: client,
	}

	key := buildS3Key(file)

	// Use GetObject to get file size from metadata
	// We close the body immediately without reading to avoid downloading content
	resp, err := client.GetObject(ctx, key)
	if err != nil {
		if isNotFoundError(err) {
			obj.setState(&s3ObjectNotExists{obj: obj})
		} else {
			return nil, fmt.Errorf("failed to check object existence: %w", err)
		}
		return obj, nil
	}

	contentLength := int64(0)
	if resp.ContentLength != nil {
		contentLength = *resp.ContentLength
	}
	if contentLength == 0 {
		contentLength = 1 // At least one byte exists
	}

	// Close the body without reading to avoid downloading content
	if resp.Body != nil {
		resp.Body.Close()
	}

	obj.setState(&s3ObjectLazyReadOnly{
		obj:          obj,
		position:     0,
		totalSize:    contentLength,
		nextFetchPos: 0,
		cache:        make(map[int64][]byte),
	})

	return obj, nil
}

func (s *s3ObjectLazyReadOnly) Read(p []byte) (n int, err error) {
	if s.position >= s.totalSize {
		return 0, io.EOF
	}

	remainingInFile := s.totalSize - s.position
	if remainingInFile <= 0 {
		return 0, io.EOF
	}

	bytesToRead := len(p)
	if int64(bytesToRead) > remainingInFile {
		bytesToRead = int(remainingInFile)
	}

	cachedData, ok := s.getCachedData(s.position, int64(bytesToRead))
	if ok {
		n = copy(p, cachedData)
		s.position += int64(n)
		return n, nil
	}

	// Use continuous range counter - fetch from nextFetchPos
	// For simplicity, fetch exactly what we need from position
	fetchStart := s.position
	fetchSize := int64(bytesToRead)

	maxFetchSize := s.totalSize - fetchStart
	if fetchSize > maxFetchSize {
		fetchSize = maxFetchSize
		if fetchSize <= 0 {
			return 0, io.EOF
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.addCallback(cancel)

	data, fetchErr := s.fetchRange(ctx, fetchStart, fetchSize)
	if fetchErr != nil {
		return 0, fmt.Errorf("failed to fetch range %d-%d: %w", fetchStart, fetchStart+fetchSize-1, fetchErr)
	}

	s.cache[fetchStart] = data
	// Update nextFetchPos for continuous counter
	if fetchStart+fetchSize > s.nextFetchPos {
		s.nextFetchPos = fetchStart + fetchSize
	}

	n = copy(p, data)
	s.position += int64(n)

	return n, nil
}

func (s *s3ObjectLazyReadOnly) getCachedData(start, length int64) ([]byte, bool) {
	// Check if the range [start, start+length) is fully cached
	if start+length > s.nextFetchPos {
		return nil, false
	}

	// For sequential caching, find which cache entry contains the start position
	for cacheStart, cacheData := range s.cache {
		cacheEnd := cacheStart + int64(len(cacheData))
		if cacheStart <= start && cacheEnd >= start+length {
			offset := start - cacheStart
			return cacheData[offset : offset+length], true
		}
	}

	return nil, false
}

func (s *s3ObjectLazyReadOnly) fetchRange(ctx context.Context, start, length int64) ([]byte, error) {
	key := buildS3Key(s.obj.file)
	rangeStr := fmt.Sprintf("bytes=%d-%d", start, start+length-1)

	resp, err := s.obj.client.GetObject(ctx, key, func(in any) {
		if getInput, ok := in.(*s3.GetObjectInput); ok {
			getInput.Range = aws.String(rangeStr)
		}
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (s *s3ObjectLazyReadOnly) Write(p []byte) (n int, err error) {
	return 0, errors.New("write not supported in lazy read-only mode")
}

func (s *s3ObjectLazyReadOnly) Close() error {
	return nil
}

func (s *s3ObjectLazyReadOnly) Seek(offset int64, whence int) (int64, error) {
	var newPos int64

	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = s.position + offset
	case io.SeekEnd:
		newPos = s.totalSize + offset
	default:
		return 0, directory.ErrInvalidSeek
	}

	if newPos < 0 {
		return 0, directory.ErrInvalidSeek
	}
	if newPos > s.totalSize {
		newPos = s.totalSize
	}

	s.position = newPos
	return newPos, nil
}
