package infrastructure_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	mocks_fyne "github.com/thomas-marquis/s3-box/mocks/fyne"
	"go.uber.org/mock/gomock"
)

func TestFyneConnectionsRepository_Get(t *testing.T) {
	t.Run("should return all connections", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		id1 := connection_deck.NewConnectionID()
		id2 := connection_deck.NewConnectionID()

		connJson := fmt.Sprintf(`[
			{
				"id": "%s",
				"name": "conn 1",
				"server": "",
				"accessKey": "ak1",
				"secretKey": "sk1",
				"bucket": "b1",
				"type": "aws",
				"region": "us-east-1",
				"useTls": true
			},
			{
				"id": "%s",
				"name": "conn 2",
				"server": "http://localhost:9000",
				"accessKey": "ak2",
				"secretKey": "sk2",
				"bucket": "b2",
				"selected": true,
				"type": "s3-like",
				"useTls": true
			}
		]`, id1, id2)

		mockPrefs.EXPECT().
			String(gomock.Eq("allConnections")).
			Return(connJson).
			Times(1)

		mockBus.EXPECT().
			Subscribe().
			Return(make(chan event.Event)).
			Times(1)

		repo := infrastructure.NewFyneConnectionsRepository(mockPrefs, mockBus)

		// When
		res, err := repo.Get(context.TODO())

		// Then
		assert.NoError(t, err)
		assert.Len(t, res.Get(), 2)
	})

	t.Run("should return ErrTechnical when json loading failed", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		mockPrefs.EXPECT().
			String(gomock.Eq("allConnections")).
			Return("invalid json").
			Times(1)

		mockBus.EXPECT().
			Subscribe().
			Return(make(chan event.Event)).
			Times(1)

		repo := infrastructure.NewFyneConnectionsRepository(mockPrefs, mockBus)

		// When
		res, err := repo.Get(context.TODO())

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, connection_deck.ErrTechnical)
		assert.Nil(t, res)
	})
}

func TestFyneConnectionsRepository(t *testing.T) {
	t.Run("SelectEventType", func(t *testing.T) {
		t.Run("should save connections to json and publish the selected connection on success", func(t *testing.T) {
			// Given & Then
			ctrl := gomock.NewController(t)
			mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
			mockBus := mocks_event.NewMockBus(ctrl)
			events := make(chan event.Event)

			done := make(chan struct{})

			mockBus.EXPECT().
				Subscribe().
				Return(events).
				Times(1)

			_ = infrastructure.NewFyneConnectionsRepository(mockPrefs, mockBus)
			defer close(events)

			deck := connection_deck.New()
			c1 := deck.New("conn 1", "ak", "sk", "bucket").Connection()
			evt, err := deck.Select(c1.ID())
			require.NoError(t, err)

			mockPrefs.EXPECT().
				SetString(gomock.Eq("allConnections"), gomock.Any()).
				Times(1)

			mockBus.EXPECT().
				Publish(gomock.Eq(connection_deck.NewSelectSuccessEvent(deck, c1))).
				Do(func(e event.Event) { close(done) }).
				Times(1)

			// When
			events <- evt
			<-done
		})
	})

	t.Run("CreateEventType", func(t *testing.T) {
		t.Run("should save connections to json and publish the new connection on success", func(t *testing.T) {
			// Given & Then
			ctrl := gomock.NewController(t)
			mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
			mockBus := mocks_event.NewMockBus(ctrl)
			events := make(chan event.Event)

			done := make(chan struct{})

			mockBus.EXPECT().
				Subscribe().
				Return(events).
				Times(1)

			_ = infrastructure.NewFyneConnectionsRepository(mockPrefs, mockBus)
			defer close(events)

			deck := connection_deck.New()
			evt := deck.New("conn 1", "ak", "sk", "bucket")
			c1 := evt.Connection()

			mockPrefs.EXPECT().
				SetString(gomock.Eq("allConnections"), gomock.Any()).
				Times(1)

			mockBus.EXPECT().
				Publish(gomock.Eq(connection_deck.NewCreateSuccessEvent(deck, c1))).
				Do(func(e event.Event) { close(done) }).
				Times(1)

			// When
			events <- evt
			<-done
		})
	})

	t.Run("RemoveEventType", func(t *testing.T) {
		t.Run("should save connections to json and publish the removed connection on success", func(t *testing.T) {
			// Given & Then
			ctrl := gomock.NewController(t)
			mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
			mockBus := mocks_event.NewMockBus(ctrl)
			events := make(chan event.Event)

			done := make(chan struct{})

			mockBus.EXPECT().
				Subscribe().
				Return(events).
				Times(1)

			_ = infrastructure.NewFyneConnectionsRepository(mockPrefs, mockBus)
			defer close(events)

			deck := connection_deck.New()
			c1 := deck.New("conn 1", "ak", "sk", "bucket").Connection()
			evt, err := deck.RemoveAConnection(c1.ID())
			require.NoError(t, err)

			mockPrefs.EXPECT().
				SetString(gomock.Eq("allConnections"), gomock.Any()).
				Times(1)

			mockBus.EXPECT().
				Publish(gomock.Eq(connection_deck.NewRemoveSuccessEvent(deck, c1))).
				Do(func(e event.Event) { close(done) }).
				Times(1)

			// When
			events <- evt
			<-done
		})
	})

	t.Run("UpdateEventType", func(t *testing.T) {
		t.Run("should save connections to json and publish the updated connection on success", func(t *testing.T) {
			// Given & Then
			ctrl := gomock.NewController(t)
			mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
			mockBus := mocks_event.NewMockBus(ctrl)
			events := make(chan event.Event)

			done := make(chan struct{})

			mockBus.EXPECT().
				Subscribe().
				Return(events).
				Times(1)

			_ = infrastructure.NewFyneConnectionsRepository(mockPrefs, mockBus)
			defer close(events)

			deck := connection_deck.New()
			c1 := deck.New("conn 1", "ak", "sk", "bucket").Connection()
			evt, err := deck.Update(c1.ID(), connection_deck.WithName("new name"))
			require.NoError(t, err)
			c2 := evt.Connection()

			mockPrefs.EXPECT().
				SetString(gomock.Eq("allConnections"), gomock.Any()).
				Times(1)

			mockBus.EXPECT().
				Publish(gomock.Eq(connection_deck.NewUpdateSuccessEvent(deck, c2))).
				Do(func(e event.Event) { close(done) }).
				Times(1)

			// When
			events <- evt
			<-done
		})
	})
}

func TestFyneConnectionsRepository_Export(t *testing.T) {
	t.Run("should export connections to writer", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		id1 := connection_deck.NewConnectionID()
		connJson := fmt.Sprintf(
			`[{"id":"%s","name":"conn 1","accessKey":"ak","secretKey":"sk","bucket":"b","type":"aws","region":"us-east-1","useTls":true}]`,
			id1)

		mockPrefs.EXPECT().
			String("allConnections").
			Return(connJson).
			Times(1)
		mockBus.EXPECT().
			Subscribe().
			Return(make(chan event.Event)).
			Times(1)

		repo := infrastructure.NewFyneConnectionsRepository(mockPrefs, mockBus)
		buf := &bytes.Buffer{}

		// When
		err := repo.Export(context.TODO(), buf)

		// Then
		assert.NoError(t, err)
		assert.JSONEq(t, connJson, buf.String())
	})

	t.Run("should return error when Get fails", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		mockPrefs.EXPECT().
			String("allConnections").
			Return("invalid json").
			Times(1)
		mockBus.EXPECT().
			Subscribe().
			Return(make(chan event.Event)).
			Times(1)

		repo := infrastructure.NewFyneConnectionsRepository(mockPrefs, mockBus)

		// When
		err := repo.Export(context.TODO(), io.Discard)

		// Then
		assert.Error(t, err)
		assert.ErrorIs(t, err, connection_deck.ErrTechnical)
	})
}
