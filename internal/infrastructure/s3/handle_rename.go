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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/transport/http"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	maxRenamingWorkers = 10

	markerSrcFileName = ".s3box-rename-src"
	markerDstFileName = ".s3box-rename-dst"
)

func isRenameMarkerFile(key string) bool {
	return strings.HasSuffix(key, markerSrcFileName) || strings.HasSuffix(key, markerDstFileName)
}

func (r *RepositoryImpl) checkSrcRenameMarker(ctx context.Context, sess *s3Session, srcDirKey, dstDirKey string) error {
	markerKey := srcDirKey + markerSrcFileName
	marker, err := readRenameMarker(ctx, sess, markerKey)
	if err != nil {
		if isNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("failed to read rename marker for source directory %s: %w", srcDirKey, err)
	}
	if marker.DstDirPath != directory.NewPath(dstDirKey) {
		return fmt.Errorf("an existing renaming process is still in progress for directory %s to %s", srcDirKey, marker.DstDirPath)
	}
	return nil
}

func (r *RepositoryImpl) checkRenamingState(ctx context.Context, sess *s3Session, srcDirKey, dstDirKey string) (bool, error) {
	srcMarkerKey := srcDirKey + markerSrcFileName
	dstMarkerKey := dstDirKey + markerDstFileName

	srcMarker, err := readRenameMarker(ctx, sess, srcMarkerKey)
	if err == nil {
		if srcMarker.DstDirPath == directory.NewPath(dstDirKey) {
			return true, nil // Resume
		}
		return false, fmt.Errorf("an existing renaming process is still in progress for directory %s to %s", srcDirKey, srcMarker.DstDirPath)
	}
	if !isNotFoundError(err) {
		return false, err
	}

	lsDst, err := r.listObjects(ctx, sess, dstDirKey, true)
	if err != nil {
		return false, err
	}

	if !lsDst.IsEmpty() {
		dstMarker, err := readRenameMarker(ctx, sess, dstMarkerKey)
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
	evt := e.(directory.FileRenamedEvent)

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed renaming file: %w", err))
		r.bus.Publish(directory.NewFileRenamedFailureEvent(err, evt.Parent(), evt.File(), evt.OldName()))
	}

	sess, err := r.getSession(ctx, evt.Parent().ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	file := evt.File()
	if file == nil {
		handleError(fmt.Errorf("file is nil for rename event"))
		return
	}

	// Construct old key using the old file name
	oldFile, err := directory.NewFile(string(evt.OldName()), evt.Parent().Path())
	if err != nil {
		handleError(err)
		return
	}
	oldKey := mapFileToKey(oldFile)

	// Construct new key with new filename
	newKey := mapFileToKey(evt.File())

	// Copy the file to new location
	copySource := sess.connection.Bucket() + "/" + oldKey
	_, err = sess.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(sess.connection.Bucket()),
		Key:        aws.String(newKey),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		handleError(r.manageAwsSdkError(err, oldKey, sess))
		return
	}

	// Delete the old file
	_, err = sess.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(oldKey),
	})
	if err != nil {
		handleError(r.manageAwsSdkError(err, oldKey, sess))
		return
	}

	r.bus.Publish(directory.NewFileRenamedSuccessEvent(evt.Parent(), evt.File(), evt.OldName()))
}

func (r *RepositoryImpl) handleRenameRequest(e event.Event) {
	ctx := e.Context()
	evt := e.(directory.RenameEvent)
	dir := evt.Directory()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed handling rename request: %w", err))
		r.bus.Publish(directory.NewRenameFailureEvent(err, dir, evt.NewName()))
	}

	sess, err := r.getSession(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	srcDirKey := mapDirToObjectKey(dir)
	dstDirKey := getDstDirKey(srcDirKey, evt.NewName())

	lsDst, err := r.listObjects(ctx, sess, dstDirKey, true)
	if err != nil {
		handleError(err)
		return
	}
	if !lsDst.IsEmpty() {
		handleError(fmt.Errorf("destination directory already exists"))
		return
	}

	lsSrc, err := r.listObjects(ctx, sess, mapPathToSearchKey(dir.Path()), true)
	if err != nil {
		handleError(err)
		return
	}

	if lsSrc.IsEmpty() {
		if err := r.renameObjects(ctx, sess, dir.Path(), evt.NewName(), lsSrc.Keys, true); err != nil {
			handleError(err)
			return
		}
		r.bus.Publish(directory.NewRenamedSuccessEvent(dir, evt.NewName()))
	} else {
		for _, key := range lsSrc.Keys {
			if isRenameMarkerFile(key) {
				handleError(r.getPendingRenameErr(ctx, sess, dir, key))
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
	uve := e.(directory.UserValidationSuccessEvent)

	re, ok := uve.Reason().(directory.RenameEvent)
	if !ok {
		return
	}
	if !uve.Validated() {
		return
	}

	dir := re.Directory()
	newName := re.NewName()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed handling rename: %w", err))
		r.bus.Publish(directory.NewRenameFailureEvent(err, dir, re.NewName()))
	}

	sess, err := r.getSession(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	srcDirKey := mapDirToObjectKey(dir)
	dstDirKey := getDstDirKey(srcDirKey, newName)

	if _, err := r.checkRenamingState(ctx, sess, srcDirKey, dstDirKey); err != nil {
		handleError(err)
		return
	}

	lsRes, err := r.listObjects(ctx, sess, mapPathToSearchKey(dir.Path()), true)
	if err != nil {
		handleError(err)
		return
	}

	if err := r.renameObjects(ctx, sess, dir.Path(), newName, lsRes.Keys, true); err != nil {
		handleError(err)
		return
	}

	r.bus.Publish(directory.NewRenamedSuccessEvent(dir, newName))
}

func (r *RepositoryImpl) handleRenameResume(evt event.Event) {
	e := evt.(directory.RenameResumeEvent)

	srcDir := e.Directory()
	dstDir := e.DstDir()

	srcPath := srcDir.Path()
	dstPath := dstDir.Path()

	ctx := e.Context()
	newName := dstPath.DirectoryName()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed handling rename: %w", err))
		r.bus.Publish(directory.NewRenameFailureEvent(err, srcDir, newName))
	}

	sess, err := r.getSession(ctx, srcDir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	srcDirKey := mapPathToSearchKey(srcPath)
	dstDirKey := mapPathToSearchKey(dstPath)

	srcMrk, err := readRenameMarker(ctx, sess, srcDirKey+markerSrcFileName)
	if err != nil {
		handleError(fmt.Errorf("failed reading rename marker at %s: %w", srcDirKey+markerSrcFileName, err))
		return
	}
	dstMrk, err := readRenameMarker(ctx, sess, dstDirKey+markerDstFileName)
	if err != nil {
		handleError(fmt.Errorf("failed reading rename marker at %s: %w", dstDirKey+markerDstFileName, err))
		return
	}

	if dstMrk.SrcDirPath != srcPath || srcMrk.DstDirPath != dstPath {
		handleError(errors.New("invalid rename marker(s) content"))
		return
	}

	lsRes, err := r.listObjects(ctx, sess, mapPathToSearchKey(srcPath), true)
	if err != nil {
		handleError(err)
		return
	}

	if err := r.renameObjects(ctx, sess, srcPath, newName, lsRes.Keys, false); err != nil {
		handleError(err)
		return
	}

	r.bus.Publish(directory.NewRenamedSuccessEvent(srcDir, newName))
}

func (r *RepositoryImpl) renameObjects(
	ctx context.Context,
	sess *s3Session,
	srcPath directory.Path,
	newName string,
	keys []string,
	createMarkers bool,
) error {
	//aclRes, err := sess.client.GetBucketAcl(ctx, &s3.GetBucketAclInput{
	//	Bucket: aws.String(sess.connection.Bucket()),
	//})
	//if err != nil {
	//	return err
	//}

	srcDirKey := mapPathToSearchKey(srcPath)
	dstDirKey := getDstDirKey(srcDirKey, newName)

	if len(keys) == 0 {
		return deleteRenameMarkers(ctx, sess.client, sess.connection.Bucket(), srcDirKey, dstDirKey)
	}

	if createMarkers {
		if err := createRenameMarkers(ctx, sess.client, sess.connection.Bucket(), srcDirKey, dstDirKey); err != nil {
			return err
		}
	}

	if len(keys) == 1 {
		key := keys[0]
		if err := r.renameObject(ctx, sess, key, getObjectDstKey(srcDirKey, dstDirKey, key)); err != nil {
			return err
		}
		return deleteRenameMarkers(ctx, sess.client, sess.connection.Bucket(), srcDirKey, dstDirKey)
	}

	var (
		nbWorkers = min(len(keys), maxRenamingWorkers)
		workload  = make(chan string)

		errCnt int64
		wg     sync.WaitGroup
	)

	for range nbWorkers {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case key := <-workload:
					if err := r.renameObject(ctx, sess, key, getObjectDstKey(srcDirKey, dstDirKey, key)); err != nil {
						atomic.AddInt64(&errCnt, 1)
					}
					wg.Done()
				}
			}
		}()
	}

	for _, key := range keys {
		wg.Add(1)
		workload <- key
	}

	wg.Wait() // TODO: !!WARNING!! this WG will block if the context is canceled before all workers are done

	if errCnt > 0 {
		return directory.UncompletedRename{
			SourceDirPath:      srcPath,
			DestinationDirPath: directory.NewPath(dstDirKey),
			Wrapped:            fmt.Errorf("%d error(s) occurred while renaming objects", errCnt),
		}
	}

	return deleteRenameMarkers(ctx, sess.client, sess.connection.Bucket(), srcDirKey, dstDirKey)
}

func (r *RepositoryImpl) renameObject(ctx context.Context, sess *s3Session, oldKey, newKey string) error {
	if strings.HasSuffix(oldKey, markerSrcFileName) || strings.HasSuffix(oldKey, markerDstFileName) {
		return nil
	}

	bucket := sess.connection.Bucket()
	headRes, err := sess.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(oldKey),
	})
	if err != nil {
		return err
	}

	var useDefaultAcl bool
	aclRes, err := sess.client.GetObjectAcl(ctx, &s3.GetObjectAclInput{ // TODO: AWS only?
		Bucket: aws.String(bucket),
		Key:    aws.String(oldKey),
	})
	if err != nil {
		var (
			opErr   *smithy.OperationError
			respErr *http.ResponseError
		)
		if errors.As(err, &opErr) && errors.As(opErr.Err, &respErr) && respErr.Response.Response.StatusCode == 403 {
			useDefaultAcl = true
		} else {
			return err
		}
	}

	var (
		grantRead        []string
		grantReadAcp     []string
		grantWriteAcp    []string
		grantFullControl []string
	)

	if !useDefaultAcl {
		for _, grant := range aclRes.Grants {
			switch grant.Permission {
			case types.PermissionRead:
				grantRead = append(grantRead, generatePermissionGrant(grant.Grantee))
			case types.PermissionReadAcp:
				grantReadAcp = append(grantReadAcp, generatePermissionGrant(grant.Grantee))
			case types.PermissionWriteAcp:
				grantWriteAcp = append(grantWriteAcp, generatePermissionGrant(grant.Grantee))
			case types.PermissionFullControl:
				grantFullControl = append(grantFullControl, generatePermissionGrant(grant.Grantee))
			case types.PermissionWrite:
				r.notifier.NotifyDebug(fmt.Sprintf("ignoring write permission for copy on key: %s", oldKey))
			default:
				r.notifier.NotifyDebug(fmt.Sprintf("unknown permission for grant: %s", grant.Permission))
			}
		}
	}

	cpyInput := &s3.CopyObjectInput{
		Bucket:                         aws.String(bucket),
		CopySource:                     aws.String(bucket + "/" + oldKey),
		Key:                            aws.String(newKey),
		CacheControl:                   headRes.CacheControl,
		ContentDisposition:             headRes.ContentDisposition,
		ContentEncoding:                headRes.ContentEncoding,
		ContentLanguage:                headRes.ContentLanguage,
		ContentType:                    headRes.ContentType,
		CopySourceSSECustomerAlgorithm: headRes.SSECustomerAlgorithm,
		CopySourceSSECustomerKeyMD5:    headRes.SSECustomerKeyMD5,
		GrantFullControl:               joinGrants(grantFullControl),
		GrantRead:                      joinGrants(grantRead),
		GrantReadACP:                   joinGrants(grantReadAcp),
		GrantWriteACP:                  joinGrants(grantWriteAcp),
		Metadata:                       headRes.Metadata,
		MetadataDirective:              "REPLACE",
		ObjectLockLegalHoldStatus:      headRes.ObjectLockLegalHoldStatus,
		ObjectLockMode:                 headRes.ObjectLockMode,
		ObjectLockRetainUntilDate:      headRes.ObjectLockRetainUntilDate,
		SSECustomerAlgorithm:           headRes.SSECustomerAlgorithm,
		SSECustomerKeyMD5:              headRes.SSECustomerKeyMD5,
		SSEKMSKeyId:                    headRes.SSEKMSKeyId,
		ServerSideEncryption:           headRes.ServerSideEncryption,
		StorageClass:                   headRes.StorageClass,
		TaggingDirective:               "COPY",
		WebsiteRedirectLocation:        headRes.WebsiteRedirectLocation,
	}
	if _, err := sess.client.CopyObject(ctx, cpyInput); err != nil {
		return err
	}

	_, err = sess.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(oldKey),
	})
	return err
}

func (r *RepositoryImpl) getPendingRenameErr(ctx context.Context, s *s3Session, dir *directory.Directory, markerKey string) error {
	m, err := readRenameMarker(ctx, s, markerKey)
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

func createRenameMarkers(ctx context.Context, client *s3.Client, bucket, srcDirPrefix, dstDirPrefix string) error {
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
		if _, err := client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(content),
		}); err != nil {
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
		if err := deleteRenameMarkers(ctx, client, bucket, srcKey, dstKey); err != nil {
			return err
		}
		return err
	default:
		return nil
	}
}

func readRenameMarker(ctx context.Context, sess *s3Session, key string) (*renameMarker, error) {
	res, err := sess.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(sess.connection.Bucket()),
		Key:    aws.String(key),
	})
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

func deleteRenameMarkers(ctx context.Context, client *s3.Client, bucket, srcDirPrefix, dstDirPrefix string) error {
	var (
		srcKey = srcDirPrefix + markerSrcFileName
		dstKey = dstDirPrefix + markerDstFileName

		wg      sync.WaitGroup
		errChan = make(chan error)
	)

	wg.Add(2)

	deleteObject := func(key string) {
		defer wg.Done()
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
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

func generatePermissionGrant(grantee *types.Grantee) string {
	if grantee.URI != nil {
		return "uri=" + *grantee.URI
	}
	if grantee.ID != nil {
		return "id=" + *grantee.ID
	}
	if grantee.EmailAddress != nil {
		return "emailAddress=" + *grantee.EmailAddress
	}
	return ""
}

func joinGrants(grants []string) *string {
	if len(grants) == 0 {
		return nil
	}
	return aws.String(strings.Join(grants, ", "))
}
