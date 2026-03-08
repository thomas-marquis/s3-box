package s3

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (r *RepositoryImpl) handleDownloadFile(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.ContentDownloadedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed downloading file: %w", err))
		r.bus.Publish(directory.NewContentDownloadedFailureEvent(err))
	}

	sess, err := r.getSession(ctx, evt.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	downloader := s3manager.NewDownloader(sess.client)

	file, err := evt.Content().Open()
	if err != nil {
		handleError(fmt.Errorf("failed opening the file to download: %w", err))
		return
	}
	defer file.Close() //nolint:errcheck

	if _, err = downloader.Download(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(mapFileToKey(evt.Content().File())),
	}); err != nil {
		handleError(r.manageAwsSdkError(err, evt.Content().File().FullPath(), sess))
		return
	}

	r.bus.Publish(directory.NewContentDownloadedSuccessEvent(evt.Content()))
}
