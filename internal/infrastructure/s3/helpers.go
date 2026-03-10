package s3

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/logging"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

type listDirResult struct {
	Keys         []string
	SizeBytesTot int64
}

func (lsr *listDirResult) IsEmpty() bool {
	return len(lsr.Keys) == 0 || (len(lsr.Keys) == 1 && strings.HasSuffix(lsr.Keys[0], "/"))
}

func (r *RepositoryImpl) manageAwsSdkError(err error, objName string, sess *s3Session) error {
	if err == nil {
		return nil
	}

	var nsk *s3types.NoSuchKey
	if errors.As(err, &nsk) {
		return errors.Join(
			directory.ErrNotFound,
			fmt.Errorf("object %s not found in bucket %s: %w",
				objName, sess.connection.Bucket(), err),
		)
	}

	var ios *s3types.InvalidObjectState
	if errors.As(err, &ios) {
		return errors.Join(
			directory.ErrTechnical,
			fmt.Errorf("impossible to read object %s from bucket %s: %w",
				objName, sess.connection.Bucket(), err),
		)
	}

	var nsb *s3types.NoSuchBucket
	if errors.As(err, &nsb) {
		return errors.Join(
			directory.ErrNotFound,
			fmt.Errorf("bucket %s not found: %w", sess.connection.Bucket(), err),
		)
	}

	return fmt.Errorf("another kind of s3 error occurred: %w", err)
}

func (r *RepositoryImpl) getSession(ctx context.Context, id connection_deck.ConnectionID) (*s3Session, error) {
	deck, err := r.connectionRepository.Get(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := deck.GetByID(id)
	if err != nil {
		return nil, err
	}
	if found := r.getFromCache(conn); found != nil {
		return found, nil
	}

	region := conn.Region()
	if region == "" {
		region = "us-east-1" // for custom endpoints, value is not important but still required
	}

	var endpoint string
	if conn.Server() != "" {
		if conn.IsTLSActivated() {
			endpoint = "https://" + conn.Server()
		} else {
			endpoint = "http://" + conn.Server()
		}
	}

	var baseEp *string
	if endpoint != "" {
		baseEp = aws.String(endpoint)
	}
	s3Client := s3.New(s3.Options{
		Credentials:  credentials.NewStaticCredentialsProvider(conn.AccessKey(), conn.SecretKey(), ""),
		Region:       region,
		BaseEndpoint: baseEp,
		Logger:       logging.NewStandardLogger(r.logger.Writer()),
		UsePathStyle: true,
	}, r.s3ClientOptions...)

	s := &s3Session{
		client:     s3Client,
		connection: conn,
		downloader: manager.NewDownloader(s3Client),
		uploader:   manager.NewUploader(s3Client),
	}
	r.Lock()
	defer r.Unlock()
	r.cache[conn.ID()] = s
	return s, nil
}

func (r *RepositoryImpl) getFromCache(c *connection_deck.Connection) *s3Session {
	r.Lock()
	defer r.Unlock()

	found, ok := r.cache[c.ID()]
	if ok && found != nil && found.connection.Is(c) {
		return found
	}
	return nil
}

func (r *RepositoryImpl) listObjects(ctx context.Context, sess *s3Session, prefix string, recursive bool) (listDirResult, error) {
	var keys []string
	var sizeBytesTot int64

	var delimiter *string
	if !recursive {
		delimiter = aws.String("/")
	}

	inputs := &s3.ListObjectsV2Input{
		Bucket:    aws.String(sess.connection.Bucket()),
		Prefix:    aws.String(prefix),
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
