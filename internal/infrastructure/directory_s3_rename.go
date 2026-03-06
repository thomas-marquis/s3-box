package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
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
	return len(lsr.Keys) == 0
}

func (r *S3DirectoryRepository) handleRenameRequest(e event.Event) {
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

	lsRes, err := r.listDir(ctx, sess, dir)
	if err != nil {
		handleError(err)
		return
	}

	if lsRes.IsEmpty() {
		if err := r.renameObjects(ctx, sess, dir, evt.NewName(), lsRes.Keys); err != nil {
			handleError(err)
		}
	} else {
		msg := strings.Builder{}
		msg.WriteString("This directory is not empty.\n")
		msg.WriteString(fmt.Sprintf("It contains %d objects (%d kB).\n", len(lsRes.Keys), lsRes.SizeBytesTot/1024))
		msg.WriteString("This operation will modify all of them. Are you sure you want to proceed?")
		r.bus.Publish(directory.NewUserValidationEvent(dir, evt, msg.String()))
	}
}

func (r *S3DirectoryRepository) handleRenameDirectory(e event.Event) {
	ctx := e.Context()
	var evt directory.RenamedEvent

	if tmp, ok := e.(directory.RenamedEvent); ok {
		evt = tmp
	} else if tmp, ok := e.(directory.UserValidationSuccessEvent); ok {
		if tmp2, ok2 := tmp.Reason().(directory.RenamedEvent); ok2 {
			evt = tmp2
		}
	} else {
		r.notifier.NotifyError(fmt.Errorf("invalid event type: %T", e))
		return
	}

	dir := evt.Directory()
	newName := evt.NewName()

	handleError := func(err error) {
		r.notifier.NotifyError(fmt.Errorf("failed handling rename: %w", err))
		r.bus.Publish(directory.NewRenamedFailureEvent(err, dir)) // TODO: rename and not renameD
	}

	sess, err := r.getSession(ctx, dir.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	lsRes, err := r.listDir(ctx, sess, dir)
	if err != nil {
		handleError(err)
		return
	}

	if err := r.renameObjects(ctx, sess, dir, newName, lsRes.Keys); err != nil {
		handleError(err)
	}
}

func (r *S3DirectoryRepository) renameObjects(ctx context.Context, sess *s3Session, dir *directory.Directory, newName string, keys []string) error {
	sess, err := r.getSession(ctx, dir.ConnectionID())
	if err != nil {
		return err
	}

	aclRes, err := sess.client.GetBucketAcl(ctx, &s3.GetBucketAclInput{
		Bucket: aws.String(sess.connection.Bucket()),
	})
	if err != nil {
		return err
	}

	bucketCannedACL := inferCannedACL(aclRes.Grants)

	var errs []error
	for _, key := range keys {
		if err := r.renameObject(
			ctx, sess, key,
			updateObjectKey(mapDirToObjectKey(dir), key, newName),
			bucketCannedACL,
		); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to rename objects: %v", errs)
	}

	return nil
}

func (r *S3DirectoryRepository) renameObject(ctx context.Context, sess *s3Session, oldKey, newKey, bucketCannedACL string) error {
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
		ACL:                            types.ObjectCannedACL(bucketCannedACL),
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

func updateObjectKey(dirPrefix, oldKey, newDirName string) string {
	return strings.Replace(oldKey, dirPrefix, newDirName, 1) // TODO: fix that with TDD
}

func inferCannedACL(grants []types.Grant) string {
	hasAllUsersRead := false
	hasAllUsersWrite := false
	hasAuthenticatedRead := false

	for _, grant := range grants {
		if grant.Grantee.URI != nil {
			switch *grant.Grantee.URI {
			case "http://acs.amazonaws.com/groups/global/AllUsers":
				if grant.Permission == types.PermissionRead {
					hasAllUsersRead = true
				}
				if grant.Permission == types.PermissionWrite {
					hasAllUsersWrite = true
				}
			case "http://acs.amazonaws.com/groups/global/AuthenticatedUsers":
				if grant.Permission == types.PermissionRead {
					hasAuthenticatedRead = true
				}
			}
		}
	}

	if hasAllUsersRead && hasAllUsersWrite {
		return "public-read-write"
	} else if hasAllUsersRead {
		return "public-read"
	} else if hasAuthenticatedRead {
		return "authenticated-read"
	}
	return "private"
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

func (r *S3DirectoryRepository) listDir(ctx context.Context, sess *s3Session, dir *directory.Directory) (listDirResult, error) {
	var keys []string
	var sizeBytesTot int64

	searchKey := mapPathToSearchKey(dir.Path())
	inputs := &s3.ListObjectsV2Input{
		Bucket:    aws.String(sess.connection.Bucket()),
		Prefix:    aws.String(searchKey),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(1000),
	}
	paginator := s3.NewListObjectsV2Paginator(sess.client, inputs)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return listDirResult{}, err
		}

		for _, obj := range page.Contents {
			if strings.HasSuffix(*obj.Key, "/") {
				continue
			}
			keys = append(keys, *obj.Key)
			sizeBytesTot += *obj.Size
		}
	}

	return listDirResult{Keys: keys, SizeBytesTot: sizeBytesTot}, nil
}
