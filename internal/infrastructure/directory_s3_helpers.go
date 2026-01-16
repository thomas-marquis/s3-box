package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func (r *S3DirectoryRepository) manageAwsSdkError(err error, objName string, sess *s3Session) error {
	if err == nil {
		return nil
	}

	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case s3.ErrCodeNoSuchKey:
			return errors.Join(
				directory.ErrNotFound,
				fmt.Errorf("object %s not found in bucket %s: %w",
					objName, sess.connection.Bucket(), aerr),
			)
		case s3.ErrCodeInvalidObjectState:
			return errors.Join(
				directory.ErrTechnical,
				fmt.Errorf("impossible to read object %s from bucket %s: %w",
					objName, sess.connection.Bucket(), aerr),
			)
		case s3.ErrCodeNoSuchBucket:
			return errors.Join(
				directory.ErrNotFound,
				fmt.Errorf("bucket %s not found: %w", sess.connection.Bucket(), aerr),
			)
		default:
			return fmt.Errorf("an error occured while attempting to read object %s from s3: %w",
				objName, err)
		}
	} else {
		return fmt.Errorf("another king of s3 error occured: %w", err)
	}
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

	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(conn.AccessKey(), conn.SecretKey(), ""),
		Endpoint:    aws.String(conn.Server()),
		Logger: aws.LoggerFunc(func(args ...interface{}) {
			logger.Printf(args[0].(string), args[1:]...)
		}),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(!conn.IsTLSActivated()),
	})
	if err != nil {
		r.logger.Printf("Error creating session: %v\n", err)
		return nil, fmt.Errorf("NewS3Repository(conn=%s): %w", conn.Name(), err)
	}
	s := &s3Session{
		session:    sess,
		client:     s3.New(sess),
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
