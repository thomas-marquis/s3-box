package s3client

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/logging"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
)

type s3LikeClient struct {
	*baseApiImpl

	notifier notification.Repository
	logger   *log.Logger
}

func newS3LikeClient(conn *connection_deck.Connection, notifier notification.Repository, opts ...func(*s3.Options)) Client {
	logger := log.New(os.Stdout, conn.ID().String(), log.LstdFlags)
	client := s3.New(s3.Options{
		Credentials:  credentials.NewStaticCredentialsProvider(conn.AccessKey(), conn.SecretKey(), ""),
		Region:       conn.Region(),
		Logger:       logging.NewStandardLogger(logger.Writer()),
		UsePathStyle: true,
	}, opts...)

	return newClientImpl(client, conn.Bucket(), &s3LikeClient{
		baseApiImpl: newBaseApiImpl(client, conn.Bucket()),
		notifier:    notifier,
		logger:      logger,
	})
}
