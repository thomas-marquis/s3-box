package infrastructure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

// S3Object implements directory.FileObject for S3 objects using a state pattern.
// It manages the lifecycle of an S3 object, transitioning between states based on
// whether the object exists in S3 or not.
type S3Object struct {
	ctx        context.Context
	conn       *connection_deck.Connection
	downloader *manager.Downloader
	uploader   *manager.Uploader
	file       *directory.File

	currentState s3ObjectState
}

var (
	_ directory.FileObject = (*S3Object)(nil)
)

// NewS3Object creates a new S3Object and initializes its state based on
// whether the object exists in S3. If the object exists, it downloads the content
// and initializes the state with it. If not, it starts in a non-existent state.
func NewS3Object(ctx context.Context, downloader *manager.Downloader, uploader *manager.Uploader, conn *connection_deck.Connection, file *directory.File) (*S3Object, error) {
	obj := &S3Object{
		ctx:        ctx,
		conn:       conn,
		file:       file,
		downloader: downloader,
		uploader:   uploader,
	}

	// Check if object exists to determine initial state
	buff := manager.NewWriteAtBuffer([]byte{})
	key := buildS3Key(file)
	_, err := downloader.Download(ctx, buff, &s3.GetObjectInput{
		Bucket: aws.String(conn.Bucket()),
		Key:    aws.String(key),
	})
	if err != nil {
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
func (o *S3Object) Read(p []byte) (n int, err error) {
	return o.currentState.Read(p)
}

// Write delegates to the current state's Write implementation
func (o *S3Object) Write(p []byte) (n int, err error) {
	return o.currentState.Write(p)
}

// Close delegates to the current state's Close implementation
func (o *S3Object) Close() error {
	return o.currentState.Close()
}

func (o *S3Object) Seek(offset int64, whence int) (int64, error) {
	return o.currentState.Seek(offset, whence)
}

func (o *S3Object) setState(state s3ObjectState) {
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

// s3ObjectState represents the state interface for S3Object.
// Each state implements different behavior for Read, Write, and Close operations.
type s3ObjectState directory.FileObject

var (
	_ s3ObjectState = (*s3ObjectNotExists)(nil)
	_ s3ObjectState = (*s3ObjectExists)(nil)
)

// s3ObjectNotExists represents the state when the S3 object does not exist.
// In this state, reads will fail and writes will create the object and transition to exists state.
type s3ObjectNotExists struct {
	obj    *S3Object
	buffer *bytes.Buffer
}

// Read returns an error since the object doesn't exist
func (s *s3ObjectNotExists) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("object does not exist: %s", s.obj.file.Name())
}

// Write buffers the content and uploads it to S3, then transitions to exists state
func (s *s3ObjectNotExists) Write(p []byte) (n int, err error) {
	if s.buffer == nil {
		s.buffer = new(bytes.Buffer)
	}

	n, err = s.buffer.Write(p)
	if err != nil {
		return n, fmt.Errorf("failed to buffer content: %w", err)
	}

	// Upload the content to S3
	key := buildS3Key(s.obj.file)
	_, err = s.obj.uploader.Upload(s.obj.ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.obj.conn.Bucket()),
		Key:    aws.String(key),
		Body:   bytes.NewReader(s.buffer.Bytes()),
	})
	if err != nil {
		return n, fmt.Errorf("failed to upload object: %w", err)
	}

	// Transition to exists state
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

func (s *s3ObjectNotExists) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("cannot seek on non-existent object")
}

// s3ObjectExists represents the state when the S3 object exists.
// In this state, reads stream from the downloaded content and writes append and re-upload.
type s3ObjectExists struct {
	obj      *S3Object
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
	endPos := s.position + int64(len(p))
	initialContentLen := len(s.content)

	if endPos > int64(initialContentLen) {
		s.content = append(s.content, make([]byte, endPos-int64(initialContentLen))...)
	}

	//replacedContent := s.content[s.position:initialContentLen]
	truncLen := int64(initialContentLen) - s.position
	truncatedParts := make([]byte, truncLen)
	copy(truncatedParts, s.content[s.position:initialContentLen])

	copy(s.content[s.position:], p)
	s.content = s.content[:endPos]

	key := buildS3Key(s.obj.file)
	s.position = 0 // reset the cursor to let the sdk reads the entier content
	_, err = s.obj.uploader.Upload(s.obj.ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.obj.conn.Bucket()),
		Key:    aws.String(key),
		Body:   s,
	})
	if err != nil {
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

// isNotFoundError checks if the error is a "not found" error from AWS
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NotFound"
	}

	return false
}
