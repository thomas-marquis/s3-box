package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/logging"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func (r *S3DirectoryRepository) manageAwsSdkError(err error, objName string, sess *s3Session) error {
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

func (r *S3DirectoryRepository) getSession(ctx context.Context, id connection_deck.ConnectionID) (*s3Session, error) {
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
	s3Client := s3.New(s3.Options{
		Credentials:  credentials.NewStaticCredentialsProvider(conn.AccessKey(), conn.SecretKey(), ""),
		Region:       region,
		BaseEndpoint: aws.String(endpoint),
		Logger:       logging.NewStandardLogger(logger.Writer()),
		UsePathStyle: true,
	})

	s := &s3Session{
		client:     s3Client,
		connection: conn,
	}
	r.Lock()
	defer r.Unlock()
	r.cache[conn.ID()] = s
	return s, nil
}

func (r *S3DirectoryRepository) getFromCache(c *connection_deck.Connection) *s3Session {
	r.Lock()
	defer r.Unlock()

	found, ok := r.cache[c.ID()]
	if ok && found != nil && found.connection.Is(c) {
		return found
	}
	return nil
}
