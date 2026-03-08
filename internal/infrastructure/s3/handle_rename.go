package s3

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
)

const (
	maxRenamingWorkers = 10
)

// RenameMarker represents the marker file used during directory rename operations
type RenameMarker struct {
	SourcePath    string    `json:"source_path"`
	OperationTime time.Time `json:"operation_time"`
}

// renameTask represents a single file/directory rename task
type renameTask struct {
	oldKey string
	newKey string
	sess   *s3Session
}

// renameResult represents the result of a rename task
type renameResult struct {
	success bool
	error   error
}

type listDirResult struct {
	Keys         []string
	SizeBytesTot int64
}

func (lsr *listDirResult) IsEmpty() bool {
	return len(lsr.Keys) == 0 || (len(lsr.Keys) == 1 && strings.HasSuffix(lsr.Keys[0], "/"))
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
	evt := e.(directory.RenamedEvent)
	dir := evt.Directory()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed handling rename request: %w", err))
		r.bus.Publish(directory.NewRenamedFailureEvent(err, dir))
	}

	sess, err := r.getSession(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	lsRes, err := r.listDir(ctx, sess, dir, true)
	if err != nil {
		handleError(err)
		return
	}

	if lsRes.IsEmpty() {
		if err := r.renameObjects(ctx, sess, dir, evt.NewName(), lsRes.Keys); err != nil {
			handleError(err)
		}
		r.bus.Publish(directory.NewRenamedSuccessEvent(dir, evt.NewName()))
	} else {
		msg := strings.Builder{}
		msg.WriteString("This directory is not empty.\n")
		msg.WriteString(fmt.Sprintf("It contains %d objects (%d kB).\n", len(lsRes.Keys), lsRes.SizeBytesTot/1024))
		msg.WriteString("This operation will modify all of them. Are you sure you want to proceed?")
		r.bus.Publish(directory.NewUserValidationEvent(dir, evt, msg.String()))
	}
}

func (r *RepositoryImpl) handleRenameDirectory(e event.Event) {
	ctx := e.Context()
	uve := e.(directory.UserValidationSuccessEvent)

	re, ok := uve.Reason().(directory.RenamedEvent)
	if !ok {
		return
	}

	dir := re.Directory()
	newName := re.NewName()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed handling rename: %w", err))
		r.bus.Publish(directory.NewRenamedFailureEvent(err, dir)) // TODO: rename and not renameD
	}

	sess, err := r.getSession(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	lsRes, err := r.listDir(ctx, sess, dir, true)
	if err != nil {
		handleError(err)
		return
	}

	if err := r.renameObjects(ctx, sess, dir, newName, lsRes.Keys); err != nil {
		handleError(err)
		return
	}

	r.bus.Publish(directory.NewRenamedSuccessEvent(dir, newName))
}

func (r *RepositoryImpl) renameObjects(ctx context.Context, sess *s3Session, dir *directory.Directory, newName string, keys []string) error {
	//aclRes, err := sess.client.GetBucketAcl(ctx, &s3.GetBucketAclInput{
	//	Bucket: aws.String(sess.connection.Bucket()),
	//})
	//if err != nil {
	//	return err
	//}

	if len(keys) == 0 {
		return nil
	}

	if len(keys) == 1 {
		key := keys[0]
		return r.renameObject(ctx, sess, key, updateObjectKey(dir, key, newName))
	}

	nbWorkers := min(len(keys), maxRenamingWorkers)
	workload := make(chan string)
	terminate := make(chan struct{})
	var errCnt int64
	var wg sync.WaitGroup

	for range nbWorkers {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-terminate:
					return
				case key := <-workload:
					wg.Add(1)
					if err := r.renameObject(ctx, sess, key, updateObjectKey(dir, key, newName)); err != nil {
						atomic.AddInt64(&errCnt, 1)
					}
					wg.Done()
				}
			}
		}()
	}

	for _, key := range keys {
		workload <- key
	}

	wg.Wait()

	if errCnt > 0 {
		return fmt.Errorf("%d error(s) occurred while renaming objects", errCnt)
	}

	return nil
}

func (r *RepositoryImpl) renameObject(ctx context.Context, sess *s3Session, oldKey, newKey string) error {
	bucket := sess.connection.Bucket()
	headRes, err := sess.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(oldKey),
	})
	if err != nil {
		return err
	}

	aclRes, err := sess.client.GetObjectAcl(ctx, &s3.GetObjectAclInput{ // TODO: AWS only?
		Bucket: aws.String(bucket),
		Key:    aws.String(oldKey),
	})
	if err != nil {
		return err
	}

	var (
		grantRead        []string
		grantReadAcp     []string
		grantWriteAcp    []string
		grantFullControl []string
	)

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

	var exp *time.Time
	if headRes.ExpiresString != nil {
		if ex, err := time.Parse(time.RFC3339, *headRes.ExpiresString); err != nil {
			exp = &ex
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
		Expires:                        exp, // TODO: is this useful?
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
		StorageClass:                   headRes.StorageClass, // TODO: is this correct?
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

func updateObjectKey(dir *directory.Directory, oldKey, newDirName string) string {
	oldDirPrefix := mapDirToObjectKey(dir)
	oldDirName := dir.Name()

	re := fmt.Sprintf(`^(.*)\/?(%s)\/$`, oldDirName)
	replaceRe := regexp.MustCompile(re)
	newPrefix := replaceRe.ReplaceAllString(oldDirPrefix, fmt.Sprintf("${1}%s/", newDirName))

	res := strings.Replace(oldKey, oldDirPrefix, newPrefix, 1)
	return res
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

func (r *RepositoryImpl) listDir(ctx context.Context, sess *s3Session, dir *directory.Directory, recursive bool) (listDirResult, error) {
	var keys []string
	var sizeBytesTot int64

	searchKey := mapPathToSearchKey(dir.Path())
	var delimiter *string
	if !recursive {
		delimiter = aws.String("/")
	}

	inputs := &s3.ListObjectsV2Input{
		Bucket:    aws.String(sess.connection.Bucket()),
		Prefix:    aws.String(searchKey),
		Delimiter: delimiter,
		MaxKeys:   aws.Int32(1000),
	}
	paginator := s3.NewListObjectsV2Paginator(sess.client, inputs)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return listDirResult{}, err
		}

		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
			sizeBytesTot += *obj.Size
		}
	}

	return listDirResult{Keys: keys, SizeBytesTot: sizeBytesTot}, nil
}
