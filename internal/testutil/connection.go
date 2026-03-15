package testutil

import (
	"math/rand"
	"testing"
	"time"

	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
)

var (
	FakeAwsConnectionId    = connection_deck.NewConnectionID()
	FakeS3LikeConnectionId = connection_deck.NewConnectionID()

	FakeAwsConnectionName    = "fake-aws-conn"
	FakeS3LikeConnectionName = "fake-s3-like-conn"

	FakeAwsAccessKeyId     = "fake-aws-access-key-id"
	FakeAwsSecretAccessKey = "fake-aws-secret-access-key"

	FakeS3LikeAccessKeyId = "fake-s3-like-access-key"
	FakeS3LikeSecretKey   = "fake-s3-like-secret-key"

	FakeAwsBucketName    = "fake-aws-bucket-name"
	FakeS3LikeBucketName = "fake-s3-like-bucket-name"

	FakeAwsRegion = "eu-west-1"
)

func FakeEmptyDeck(t *testing.T) *connection_deck.Deck {
	t.Helper()
	return connection_deck.New()
}

func FakeDeckWithConnections(t *testing.T, connections ...*connection_deck.Connection) *connection_deck.Deck {
	t.Helper()
	return connection_deck.New(connection_deck.WithConnections(connections))
}

func FakeAwsConnection(t *testing.T, bucket string) *connection_deck.Connection {
	t.Helper()
	return FakeEmptyDeck(t).
		New(FakeAwsConnectionName, FakeAwsAccessKeyId, FakeAwsSecretAccessKey, bucket,
			connection_deck.AsAWS(FakeAwsRegion),
			connection_deck.WithID(FakeAwsConnectionId)).
		Connection()
}

func FakeS3LikeConnection(t *testing.T, endpoint, bucket string) *connection_deck.Connection {
	t.Helper()
	return FakeEmptyDeck(t).
		New(FakeS3LikeConnectionName, FakeS3LikeAccessKeyId, FakeS3LikeSecretKey, bucket,
			connection_deck.AsS3Like(endpoint, false),
			connection_deck.WithID(FakeS3LikeConnectionId)).
		Connection()
}

func FakeDeckWithS3LikeConnection(t *testing.T, endpoint, bucket string) *connection_deck.Deck {
	t.Helper()

	return connection_deck.New(connection_deck.WithConnections([]*connection_deck.Connection{
		FakeS3LikeConnection(t, endpoint, bucket),
	}))
}

func FakeRandomBucketName() string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	const length = 16

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[r.Intn(len(charset))]
	}
	return string(result)
}
