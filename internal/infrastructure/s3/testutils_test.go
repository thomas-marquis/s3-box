package s3_test

import (
	"testing"

	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	mocks_connection_deck "github.com/thomas-marquis/s3-box/mocks/connection_deck"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	mocks_notification "github.com/thomas-marquis/s3-box/mocks/notification"
	"go.uber.org/mock/gomock"
)

func setupMocks(t *testing.T, deck *connection_deck.Deck, events chan event.Event) (*mocks_event.MockBus, *mocks_connection_deck.MockRepository, *mocks_notification.MockRepository) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockBus := mocks_event.NewMockBus(ctrl)
	mockConnRepo := mocks_connection_deck.NewMockRepository(ctrl)
	mockNotifRepo := mocks_notification.NewMockRepository(ctrl)

	mockNotifRepo.EXPECT().NotifyDebug(gomock.Any()).AnyTimes()

	mockBus.EXPECT().
		Subscribe().
		Return(event.NewSubscriber(events))

	mockConnRepo.EXPECT().
		Get(gomock.AssignableToTypeOf(testutil.CtxType)).
		Return(deck, nil).
		Times(1)

	return mockBus, mockConnRepo, mockNotifRepo
}
