package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3/s3client"
)

const (
	maxRenamingWorkers = 10

	markerSrcFileName = ".s3box-rename-src"
	markerDstFileName = ".s3box-rename-dst"
)

func (r *RepositoryImpl) checkRenamingState(ctx context.Context, client s3client.Client, srcDirKey, dstDirKey string) (bool, error) {
	srcMarkerKey := srcDirKey + markerSrcFileName
	dstMarkerKey := dstDirKey + markerDstFileName

	srcMarker, err := readRenameMarker(ctx, client, srcMarkerKey)
	if err == nil {
		if srcMarker.DstDirPath == directory.NewPath(dstDirKey) {
			return true, nil // Resume
		}
		return false, fmt.Errorf("an existing renaming process is still in progress for directory %s to %s", srcDirKey, srcMarker.DstDirPath)
	}
	if !isNotFoundError(err) {
		return false, err
	}

	lsDst, err := client.ListObjects(ctx, dstDirKey, true)
	if err != nil {
		return false, err
	}

	if !lsDst.IsEmpty() {
		dstMarker, err := readRenameMarker(ctx, client, dstMarkerKey)
		if err == nil {
			if dstMarker.SrcDirPath == directory.NewPath(srcDirKey) {
				return true, nil // Resume
			}
		}
		return false, fmt.Errorf("destination directory already exists")
	}

	return false, nil
}

func (r *RepositoryImpl) handleRenameFile(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.FileRenameEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed renaming file: %w", err))
		r.bus.Publish(directory.NewFileRenameFailureEvent(err, evt.Directory(), evt.File(), evt.NewName()))
	}

	client, err := r.clientFactory.Get(ctx, evt.Directory().ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	oldKey := mapFileToKey(evt.File())
	newFile, err := directory.NewFile(evt.NewName(), evt.Directory())
	if err != nil {
		handleError(err)
		return
	}
	newKey := mapFileToKey(newFile)

	if err := client.RenameObject(ctx, oldKey, newKey); err != nil {
		handleError(err)
		return
	}

	r.bus.Publish(directory.NewFileRenameSuccessEvent(evt.Directory(), evt.File(), evt.NewName()))
}

func (r *RepositoryImpl) handleRenameRequest(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.RenameEvent)
	dir := evt.Directory()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed handling rename request: %w", err))
		r.bus.Publish(directory.NewRenameFailureEvent(err, dir, evt.NewName()))
	}

	client, err := r.clientFactory.Get(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	srcDirKey := mapDirToObjectKey(dir)
	dstDirKey := getDstDirKey(srcDirKey, evt.NewName())

	lsDst, err := client.ListObjects(ctx, dstDirKey, true)
	if err != nil {
		handleError(err)
		return
	}
	if !lsDst.IsEmpty() {
		handleError(fmt.Errorf("destination directory already exists"))
		return
	}

	lsSrc, err := client.ListObjects(ctx, mapPathToSearchKey(dir.Path()), true)
	if err != nil {
		handleError(err)
		return
	}

	if lsSrc.IsEmpty() {
		if err := r.renameObjects(ctx, client, dir.Path(), evt.NewName(), lsSrc.Keys, true, false); err != nil {
			handleError(err)
			return
		}
		r.bus.Publish(directory.NewRenameSuccessEvent(dir, evt.NewName()))
	} else {
		for _, key := range lsSrc.Keys {
			if isRenameMarkerFile(key) {
				handleError(r.getPendingRenameErr(ctx, client, dir, key))
				return
			}
		}

		msg := strings.Builder{}
		msg.WriteString("This directory is not empty.\n")
		msg.WriteString(fmt.Sprintf("It contains %d objects (%d kB).\n", len(lsSrc.Keys), lsSrc.SizeBytesTot/1024))
		msg.WriteString("This operation will modify all of them. Are you sure you want to proceed?")
		r.bus.Publish(directory.NewUserValidationEvent(dir, evt, msg.String()))
	}
}

func (r *RepositoryImpl) handleRenameDirectory(e event.Event) {
	ctx := e.Context()
	uve := e.(directory.UserValidationAcceptedEvent)

	re, ok := uve.Reason().(directory.RenameEvent)
	if !ok {
		return
	}

	dir := re.Directory()
	newName := re.NewName()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed handling rename: %w", err))
		r.bus.Publish(directory.NewRenameFailureEvent(err, dir, re.NewName()))
	}

	client, err := r.clientFactory.Get(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	srcDirKey := mapDirToObjectKey(dir)
	dstDirKey := getDstDirKey(srcDirKey, newName)

	if _, err := r.checkRenamingState(ctx, client, srcDirKey, dstDirKey); err != nil {
		handleError(err)
		return
	}

	lsRes, err := client.ListObjects(ctx, mapPathToSearchKey(dir.Path()), true)
	if err != nil {
		handleError(err)
		return
	}

	if err := r.renameObjects(ctx, client, dir.Path(), newName, lsRes.Keys, true, false); err != nil {
		handleError(err)
		return
	}

	r.bus.Publish(directory.NewRenameSuccessEvent(dir, newName))
}

func (r *RepositoryImpl) handleRenameRecovery(evt event.Event) {
	e := evt.(directory.RenameRecoverEvent)
	ctx := e.Context()

	switch e.Choice() {
	case directory.RecoveryChoiceRenameResume:
		r.handleRenameResuming(ctx, e.Directory(), e.DstDir(), false)
	case directory.RecoveryChoiceRenameRollback:
		r.handleRenameResuming(ctx, e.DstDir(), e.Directory(), true)
	case directory.RecoveryChoiceRenameAbort:
		r.handleRenameAbort(ctx, e.Directory(), e.DstDir())
	default:
		return
	}
}

func (r *RepositoryImpl) handleRenameResuming(ctx context.Context, srcDir, dstDir *directory.Directory, isRollback bool) {
	srcPath := srcDir.Path()
	dstPath := dstDir.Path()

	newName := dstPath.DirectoryName()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed handling rename: %w", err))
		r.bus.Publish(directory.NewRenameFailureEvent(err, srcDir, newName))
	}

	client, err := r.clientFactory.Get(ctx, srcDir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	srcDirKey := mapPathToSearchKey(srcPath)
	dstDirKey := mapPathToSearchKey(dstPath)

	var srcMarkerKey, dstMarkerKey string
	if isRollback {
		srcMarkerKey = dstDirKey + markerSrcFileName
		dstMarkerKey = srcDirKey + markerDstFileName
	} else {
		srcMarkerKey = srcDirKey + markerSrcFileName
		dstMarkerKey = dstDirKey + markerDstFileName
	}

	srcMrk, err := readRenameMarker(ctx, client, srcMarkerKey)
	if err != nil {
		handleError(fmt.Errorf("failed reading rename marker at %s: %w", srcMarkerKey, err))
		return
	}
	dstMrk, err := readRenameMarker(ctx, client, dstMarkerKey)
	if err != nil {
		handleError(fmt.Errorf("failed reading rename marker at %s: %w", dstMarkerKey, err))
		return
	}

	if (!isRollback && (dstMrk.SrcDirPath != srcPath || srcMrk.DstDirPath != dstPath)) ||
		(isRollback && (srcMrk.DstDirPath != srcPath || dstMrk.SrcDirPath != dstPath)) {
		handleError(errors.New("invalid rename marker(s) content"))
		return
	}

	lsRes, err := client.ListObjects(ctx, mapPathToSearchKey(srcPath), true)
	if err != nil {
		handleError(err)
		return
	}

	if err := r.renameObjects(ctx, client, srcPath, newName, lsRes.Keys, false, isRollback); err != nil {
		handleError(err)
		return
	}

	r.bus.Publish(directory.NewRenameSuccessEvent(srcDir, newName))
}

func (r *RepositoryImpl) handleRenameAbort(ctx context.Context, srcDir, dstDir *directory.Directory) {
	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed aborting rename: %w", err))
		r.bus.Publish(directory.NewRenameFailureEvent(err, srcDir, dstDir.Name()))
	}

	// meh...
	var connID connection_deck.ConnectionID
	if srcDir != nil {
		connID = srcDir.ConnectionID()
	} else {
		connID = dstDir.ConnectionID()
	}

	client, err := r.clientFactory.Get(ctx, connID)
	if err != nil {
		handleError(err)
		return
	}

	var srcDirKey, dstDirKey string
	if srcDir != nil {
		srcDirKey = mapPathToSearchKey(srcDir.Path())
	}
	if dstDir != nil {
		dstDirKey = mapPathToSearchKey(dstDir.Path())
	}

	if srcDirKey == "" {
		srcDirKey = dstDirKey
	} else if dstDirKey == "" {
		dstDirKey = srcDirKey
	}

	if err := deleteRenameMarkers(ctx, client, srcDirKey, dstDirKey, false); err != nil {
		handleError(err)
	}

	if srcDir != nil {
		go func() {
			if err := r.loadDirectory(ctx, client, srcDir); err != nil {
				handleError(err)
			}
		}()
	}
	if dstDir != nil {
		go func() {
			if err := r.loadDirectory(ctx, client, dstDir); err != nil {
				handleError(err)
			}
		}()
	}
}

func (r *RepositoryImpl) renameObjects(
	ctx context.Context,
	client s3client.Client,
	srcPath directory.Path,
	newName string,
	keys []string,
	createMarkers bool,
	isRollback bool,
) error {
	srcDirKey := mapPathToSearchKey(srcPath)
	dstDirKey := getDstDirKey(srcDirKey, newName)

	if len(keys) == 0 {
		return deleteRenameMarkers(ctx, client, srcDirKey, dstDirKey, isRollback)
	}

	if createMarkers {
		if err := createRenameMarkers(ctx, client, srcDirKey, dstDirKey); err != nil {
			return err
		}
	}

	if len(keys) == 1 {
		key := keys[0]
		if isRenameMarkerFile(key) {
			return nil
		}
		if err := client.RenameObject(ctx, key, getObjectDstKey(srcDirKey, dstDirKey, key)); err != nil {
			return err
		}
		return deleteRenameMarkers(ctx, client, srcDirKey, dstDirKey, isRollback)
	}

	var (
		nbWorkers = min(len(keys), maxRenamingWorkers)
		workload  = make(chan string)
		done      = make(chan struct{})

		errCnt int64
		wg     sync.WaitGroup
		once   sync.Once
	)
	defer close(workload)

	for range nbWorkers {
		go func() {
			for {
				select {
				case <-done:
					return
				case key := <-workload:
					if isRenameMarkerFile(key) {
						wg.Done()
						continue
					}
					if err := client.RenameObject(ctx, key, getObjectDstKey(srcDirKey, dstDirKey, key)); err != nil {
						atomic.AddInt64(&errCnt, 1)
					}
					wg.Done()
				}
			}
		}()
	}

	for _, key := range keys {
		select {
		case <-ctx.Done():
			once.Do(func() { close(done) })
		default:
			wg.Add(1)
			workload <- key
		}
	}

	wg.Wait()
	once.Do(func() { close(done) })

	if errCnt > 0 {
		return directory.UncompletedRename{
			SourceDirPath:      srcPath,
			DestinationDirPath: directory.NewPath(dstDirKey),
			Wrapped:            fmt.Errorf("%d error(s) occurred while renaming objects", errCnt),
		}
	}

	return deleteRenameMarkers(ctx, client, srcDirKey, dstDirKey, isRollback)
}

func (r *RepositoryImpl) getPendingRenameErr(ctx context.Context, client s3client.Client, dir *directory.Directory, markerKey string) error {
	m, err := readRenameMarker(ctx, client, markerKey)
	if err != nil {
		wErr := fmt.Errorf("error while reading rename marker: %w", err)
		return wErr
	}

	var srcDirPath, dstDirPath directory.Path
	if strings.HasSuffix(markerKey, markerSrcFileName) {
		srcDirPath = dir.Path()
		dstDirPath = m.DstDirPath
	} else {
		srcDirPath = m.SrcDirPath
		dstDirPath = dir.Path()
	}

	return directory.UncompletedRename{
		SourceDirPath:      srcDirPath,
		DestinationDirPath: dstDirPath,
		Wrapped:            fmt.Errorf("rename operation has not been completed: %s -> %s", srcDirPath, dstDirPath),
	}
}

type renameMarker struct {
	SrcDirPath directory.Path `json:"srcPath,omitempty"`
	DstDirPath directory.Path `json:"dstPath,omitempty"`
}

func createRenameMarkers(ctx context.Context, client s3client.Client, srcDirPrefix, dstDirPrefix string) error {
	mSrcContent, err := json.Marshal(renameMarker{
		DstDirPath: directory.NewPath(dstDirPrefix),
	})
	if err != nil {
		return err
	}
	mDskContent, err := json.Marshal(renameMarker{
		SrcDirPath: directory.NewPath(srcDirPrefix),
	})
	if err != nil {
		return err
	}

	errChan := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	putObject := func(key string, content []byte) {
		defer wg.Done()
		if err := client.PutObject(ctx, key, bytes.NewReader(content)); err != nil {
			select {
			case errChan <- err:
			default:
			}
		}
	}

	var (
		srcKey = srcDirPrefix + markerSrcFileName
		dstKey = dstDirPrefix + markerDstFileName
	)

	go putObject(srcKey, mSrcContent)
	go putObject(dstKey, mDskContent)

	wg.Wait()

	select {
	case err := <-errChan:
		close(errChan)
		if err := deleteRenameMarkers(ctx, client, srcKey, dstKey, false); err != nil {
			return err
		}
		return err
	default:
		return nil
	}
}

func readRenameMarker(ctx context.Context, client s3client.Client, key string) (*renameMarker, error) {
	res, err := client.GetObject(ctx, key)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close() //nolint:errcheck

	var m renameMarker
	if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func deleteRenameMarkers(ctx context.Context, client s3client.Client, srcDirPrefix, dstDirPrefix string, markerInversed bool) error {
	var (
		srcKey = srcDirPrefix + markerSrcFileName
		dstKey = dstDirPrefix + markerDstFileName

		wg      sync.WaitGroup
		errChan = make(chan error)
	)
	if markerInversed {
		srcKey = dstDirPrefix + markerSrcFileName
		dstKey = srcDirPrefix + markerDstFileName
	}

	wg.Add(2)

	deleteObject := func(key string) {
		defer wg.Done()
		if err := client.DeleteObject(ctx, key); err != nil {
			var nskErr *types.NoSuchKey
			if errors.As(err, &nskErr) {
				return
			}
			select {
			case errChan <- err:
			default:
			}
		}
	}

	go deleteObject(srcKey)
	go deleteObject(dstKey)

	wg.Wait()

	select {
	case err := <-errChan:
		close(errChan)
		return err
	default:
		return nil
	}
}

func getObjectDstKey(srcDirPrefix, dstDirPrefix, oldKey string) string {
	return strings.Replace(oldKey, srcDirPrefix, dstDirPrefix, 1)
}

func getDstDirKey(srcDirKey, newName string) string {
	parts := strings.Split(strings.TrimSuffix(srcDirKey, "/"), "/")
	parts[len(parts)-1] = newName
	return strings.Join(parts, "/") + "/"
}

func isRenameMarkerFile(key string) bool {
	return strings.HasSuffix(key, markerSrcFileName) || strings.HasSuffix(key, markerDstFileName)
}
