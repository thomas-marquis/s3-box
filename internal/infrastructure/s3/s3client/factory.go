package s3client

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
)

type Factory interface {
	Get(ctx context.Context, connID connection_deck.ConnectionID) (Client, error)
	Remove(connId connection_deck.ConnectionID)
}

func NewFactory(connectionRepository connection_deck.Repository, notifier notification.Repository, opts ...func(*s3.Options)) Factory {
	return &factoryImpl{
		cache:      make(map[connection_deck.ConnectionID]Client),
		repository: connectionRepository,
		notifier:   notifier,
		opts:       opts,
	}
}

type factoryImpl struct {
	sync.Mutex

	cache      map[connection_deck.ConnectionID]Client
	repository connection_deck.Repository
	notifier   notification.Repository
	opts       []func(*s3.Options)
}

func (f *factoryImpl) Get(ctx context.Context, connID connection_deck.ConnectionID) (Client, error) {
	f.Lock()
	defer f.Unlock()
	if c, ok := f.cache[connID]; ok {
		return c, nil
	}

	deck, err := f.repository.Get(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := deck.GetByID(connID)
	if err != nil {
		return nil, err
	}

	var newClient Client
	// TODO: implement a better discrimination system between connection types
	if conn.Region() == "" {
		// S3Like
		newClient = NewS3LikeClient(conn, f.opts...)
	} else {
		// AWS
		newClient = NewAwsClient(conn, f.opts...)
	}

	f.cache[connID] = newClient
	return newClient, nil
}

func (f *factoryImpl) Remove(connId connection_deck.ConnectionID) {
	delete(f.cache, connId)
}
