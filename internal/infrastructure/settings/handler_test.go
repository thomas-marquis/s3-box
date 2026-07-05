package settings_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/eventest"
	"github.com/thomas-marquis/s3-box/internal/domain/settings"
	infra "github.com/thomas-marquis/s3-box/internal/infrastructure/settings"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	mocks_event "github.com/thomas-marquis/s3-box/mocks/event"
	mocks_fyne "github.com/thomas-marquis/s3-box/mocks/fyne"
	"go.uber.org/mock/gomock"
)

type jsonEqMatcher struct {
	expectedJson string
	t            *testing.T
}

var _ gomock.Matcher = (*jsonEqMatcher)(nil)

func (m jsonEqMatcher) Matches(val any) bool {
	return assert.JSONEq(m.t, m.expectedJson, val.(string))
}

func (m jsonEqMatcher) String() string {
	return fmt.Sprintf("JSON is equal to %s", m.expectedJson)
}

func TestFyneSettingsHandler_write(t *testing.T) {
	t.Run("should write the new settings when it exists", func(t *testing.T) {
		// Given
		ctrl := gomock.NewController(t)
		mockPrefs := mocks_fyne.NewMockPreferences(ctrl)
		mockBus := mocks_event.NewMockBus(ctrl)

		existingPrefs := `{
			"timeout": {
				"name": "timeout",
				"nsValue": 10000000
			},
			"maxConcurrency": {
				"name": "maxConcurrency",
				"u64Value": 3
			},
			"lang": {
				"name": "lang",
				"strValue": "en"
			}
		}`
		mockPrefs.EXPECT().String(gomock.Eq("settingsV2")).Return(existingPrefs).Times(1)

		newPrefs := `{
			"timeout": {
				"name": "timeout",
				"nsValue": 10000000
			},
			"maxConcurrency": {
				"name": "maxConcurrency",
				"u64Value": 3
			},
			"lang": {
				"name": "lang",
				"strValue": "fr"
			}
		}`
		mockPrefs.EXPECT().
			SetString(gomock.Eq("settingsV2"), jsonEqMatcher{t: t, expectedJson: newPrefs}).
			Times(1)

		events := make(chan event.Event)
		defer close(events)
		mockBus.EXPECT().Subscribe().Return(event.NewSubscriber(events)).Times(1)

		done := make(chan struct{})
		mockBus.EXPECT().
			Publish(eventest.PayloadEq(settings.WriteSucceeded{
				Name:  "lang",
				Value: "fr",
			})).
			Do(func(evt event.Event) {
				defer close(done)
			}).
			Times(1)

		// When
		infra.FyneSettingsHandler(mockBus, mockPrefs)
		events <- event.New(settings.WriteTriggered{
			Name:  "lang",
			Value: "fr",
		})

		// Then
		testutil.AssertEventually(t, done)
	})
}
