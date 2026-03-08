package s3

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

func (r *RepositoryImpl) handleCreateFile(evt event.Event) {
	ctx := evt.Context()

	e := evt.(directory.FileCreatedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed creating file: %w", err))
		r.bus.Publish(directory.NewFileCreatedFailureEvent(err, e.Directory()))
	}

	obj, err := r.loadFile(ctx, e.File(), e.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}
	if _, err := obj.Write([]byte{}); err != nil {
		handleError(err)
		return
	}

	r.bus.Publish(directory.NewFileCreatedSuccessEvent(e.Directory(), e.File()))
}

func (r *RepositoryImpl) handleCreateDirectory(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.CreatedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed creating directory: %w", err))
		r.bus.Publish(directory.NewCreatedFailureEvent(err, evt.Parent()))
	}

	sess, err := r.getSession(ctx, evt.Parent().ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	newDir := evt.Directory()
	if newDir == nil {
		handleError(fmt.Errorf("directory path is empty for created event"))
		return
	}

	key := mapDirToObjectKey(newDir)
	if _, err := sess.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(key),
		Body:   strings.NewReader(""),
	}); err != nil {
		handleError(r.manageAwsSdkError(err, newDir.Path().String(), sess))
		return
	}

	r.bus.Publish(
		directory.NewCreatedSuccessEvent(evt.Parent(), evt.Directory()))
}
