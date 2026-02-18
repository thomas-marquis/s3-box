package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type s3Session struct {
	connection *connection_deck.Connection
	client     *s3.Client
	downloader *manager.Downloader
	uploader   *manager.Uploader
}

type S3DirectoryRepository struct {
	sync.Mutex
	connectionRepository connection_deck.Repository
	logger               *log.Logger
	cache                map[connection_deck.ConnectionID]*s3Session
	bus                  event.Bus
	notifier             notification.Repository
}

var _ directory.Repository = (*S3DirectoryRepository)(nil)

func NewS3DirectoryRepository(
	connectionsRepository connection_deck.Repository,
	bus event.Bus,
	notifier notification.Repository,
) (*S3DirectoryRepository, error) {
	r := &S3DirectoryRepository{
		connectionRepository: connectionsRepository,
		logger:               log.New(os.Stdout, "S3Repository: ", log.LstdFlags),
		cache:                make(map[connection_deck.ConnectionID]*s3Session),
		bus:                  bus,
		notifier:             notifier,
	}

	go r.listen(bus, notifier)

	return r, nil
}

func (r *S3DirectoryRepository) GetFileContent(
	ctx context.Context,
	connId connection_deck.ConnectionID,
	file *directory.File,
) (*directory.Content, error) {
	s, err := r.getSession(ctx, connId)
	if err != nil {
		return nil, fmt.Errorf("GetFileContent: %w", err)
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.connection.Bucket()),
		Key:    aws.String(mapFileToKey(file)),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, r.manageAwsSdkError(err, file.FullPath(), s)
	}

	defer result.Body.Close() //nolint:errcheck

	buff := new(bytes.Buffer)
	if _, err = buff.ReadFrom(result.Body); err != nil {
		return nil, fmt.Errorf("fail reading the body content: %w", err)
	}

	rd, w, _ := os.Pipe()
	defer w.Close() //nolint:errcheck
	if _, err := w.Write(buff.Bytes()); err != nil {
		return nil, fmt.Errorf("fail writing the body content: %w", err)
	}

	content := directory.NewFileContent(file, directory.ContentFromFile(rd))

	return content, nil
}

func (r *S3DirectoryRepository) listen(bus event.Bus, notifier notification.Repository) {
	bus.SubscribeV2().
		On(event.Is(directory.CreatedEventType), r.handleCreateDirectory).
		On(event.Is(directory.DeletedEventType), r.handleDeleteDirectory).
		On(event.Is(directory.FileCreatedEventType), r.handleCreateFile).
		On(event.Is(directory.FileDeletedEventType), r.handleDeleteFile).
		On(event.Is(directory.ContentUploadedEventType), r.handleUploadFile).
		On(event.Is(directory.ContentDownloadEventType), r.handleDownloadFile).
		On(event.Is(directory.LoadEventType), r.handleLoading).
		On(event.Is(directory.FileLoadEventType), r.handleLoadFile).
		ListenNonBlocking()
}
