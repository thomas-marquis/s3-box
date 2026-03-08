package s3_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	mocks_connection_deck "github.com/thomas-marquis/s3-box/mocks/connection_deck"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	mocks_notification "github.com/thomas-marquis/s3-box/mocks/notification"
	"go.uber.org/mock/gomock"
)

func TestNewS3DirectoryRepository_GetFileContent(t *testing.T) {
	ctx := context.Background()
	endpoint, terminate := setupS3testContainer(ctx, t)
	defer terminate()
	client := setupS3Client(t, endpoint)

	setupS3Bucket(ctx, t, client, testutil.FakeS3LikeBucketName, []fakeS3Object{
		{Key: "root_file.txt", Body: strings.NewReader("coucou")},
		{Key: "mydir/file_in_dir.txt", Body: strings.NewReader("lolo")},
	})

	fakeDeck := testutil.FakeDeckWithS3LikeConnection(t, endpoint)

	t.Run("should return the file content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		fakeEventChan := make(chan event.Event)
		defer close(fakeEventChan)
		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(testutil.CtxType)).
			Return(fakeDeck, nil).
			Times(1)

		repo, err := s3.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		file, err := directory.NewFile("root_file.txt", directory.RootPath)
		require.NoError(t, err)

		// When
		res, err := repo.GetFileContent(context.TODO(), testutil.FakeS3LikeConnectionId, file)

		// Then
		assert.NoError(t, err)

		f, err := res.Open()
		assert.NoError(t, err)
		var resContent []byte
		resContent, err = io.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, "coucou", string(resContent))
	})

	t.Run("should return the file content", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)
		mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

		fakeEventChan := make(chan event.Event)
		defer close(fakeEventChan)
		mockBus.EXPECT().
			Subscribe().
			Return(event.NewSubscriber(fakeEventChan))

		mockConnRepo.EXPECT().
			Get(gomock.AssignableToTypeOf(testutil.CtxType)).
			Return(fakeDeck, nil).
			Times(1)

		repo, err := s3.NewS3DirectoryRepository(mockConnRepo, mockBus, mockNotifRepo)
		require.NoError(t, err)

		file, err := directory.NewFile("file_in_dir.txt", directory.NewPath("/mydir/"))
		require.NoError(t, err)

		// When
		res, err := repo.GetFileContent(context.TODO(), testutil.FakeS3LikeConnectionId, file)

		// Then
		assert.NoError(t, err)

		f, err := res.Open()
		assert.NoError(t, err)
		var resContent []byte
		resContent, err = io.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, "lolo", string(resContent))
	})
}
