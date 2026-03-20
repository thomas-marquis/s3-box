package s3

import (
	"errors"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"

	"github.com/aws/smithy-go"
)

//func (r *RepositoryImpl) getSession(ctx context.Context, id connection_deck.ConnectionID) (*s3Session, error) {
//	deck, err := r.connectionRepository.Get(ctx)
//	if err != nil {
//		return nil, err
//	}
//
//	conn, err := deck.GetByID(id)
//	if err != nil {
//		return nil, err
//	}
//	if found := r.getFromCache(conn); found != nil {
//		return found, nil
//	}
//
//	region := conn.Region()
//	if region == "" {
//		region = "us-east-1" // for custom endpoints, value is not important but still required
//	}
//
//	var endpoint string
//	if conn.Server() != "" {
//		if conn.IsTLSActivated() {
//			endpoint = "https://" + conn.Server()
//		} else {
//			endpoint = "http://" + conn.Server()
//		}
//	}
//
//	var baseEp *string
//	if endpoint != "" {
//		baseEp = aws.String(endpoint)
//	}
//	s3Client := s3.New(s3.Options{
//		Credentials:  credentials.NewStaticCredentialsProvider(conn.AccessKey(), conn.SecretKey(), ""),
//		Region:       region,
//		BaseEndpoint: baseEp,
//		Logger:       logging.NewStandardLogger(r.logger.Writer()),
//		UsePathStyle: true,
//	}, r.s3ClientOptions...)
//
//	s := &s3Session{
//		client:     s3Client,
//		connection: conn,
//		downloader: manager.NewDownloader(s3Client),
//		uploader:   manager.NewUploader(s3Client),
//	}
//	r.Lock()
//	defer r.Unlock()
//	r.cache[conn.ID()] = s
//	return s, nil
//}

//func (r *RepositoryImpl) getFromCache(c *connection_deck.Connection) *s3Session {
//	r.Lock()
//	defer r.Unlock()
//
//	found, ok := r.cache[c.ID()]
//	if ok && found != nil && found.connection.Is(c) {
//		return found
//	}
//	return nil
//}

// isNotFoundError checks if the error is a "not found" error from AWS
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, directory.ErrNotFound) {
		return true
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NotFound"
	}

	return false
}
