package s3

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (r *RepositoryImpl) handleDeleteFile(evt event.Event) {
	ctx := evt.Context()
	e := evt.(directory.FileDeletedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed deleting file: %w", err))
		r.bus.Publish(directory.NewFileDeletedFailureEvent(err, e.Parent()))
	}

	sess, err := r.getSession(ctx, e.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	file := e.File()
	if file == nil {
		err := fmt.Errorf("file is nil for deletion event")
		handleError(err)
		return
	}

	key := mapFileToKey(file)
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(key),
	}

	if _, err := sess.client.DeleteObject(ctx, input); err != nil {
		err := r.manageAwsSdkError(err, file.FullPath(), sess)
		handleError(err)
		return
	}

	r.bus.Publish(
		directory.NewFileDeletedSuccessEvent(e.Parent(), e.File()))
}

func (r *RepositoryImpl) handleDeleteDirectory(_ event.Event) {
	err := fmt.Errorf("deleting directories is not yet implemented")
	r.notifier.NotifyError(err)
	r.bus.Publish(directory.NewDeletedFailureEvent(err))
}
