package notification_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-marquis/s3-box/internal/domain/notification"
)

func TestLevel_String(t *testing.T) {
	t.Run("should return the string representation of the level", func(t *testing.T) {
		// Given
		level := notification.LevelError

		// When
		result := level.String()

		// Then
		assert.Equal(t, "notification.level.error", result)
	})
}

func TestLevel_LowerOrEqual(t *testing.T) {
	tests := []struct {
		current  notification.Level
		target   notification.Level
		expected bool
	}{
		// When the error level is set, only error logs are emitted
		{notification.LevelError, notification.LevelError, true},
		{notification.LevelInfo, notification.LevelError, false},
		{notification.LevelDebug, notification.LevelError, false},

		// When the info level is set, only error and info logs are emitted
		{notification.LevelError, notification.LevelInfo, true},
		{notification.LevelInfo, notification.LevelInfo, true},
		{notification.LevelDebug, notification.LevelInfo, false},

		// When the debug level is set, all logs are emitted
		{notification.LevelError, notification.LevelDebug, true},
		{notification.LevelInfo, notification.LevelDebug, true},
		{notification.LevelDebug, notification.LevelDebug, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.current)+" is at least "+string(tt.target), func(t *testing.T) {
			// When
			result := tt.current.LowerOrEqual(tt.target)

			// Then
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewError(t *testing.T) {
	t.Run("should create an error notification", func(t *testing.T) {
		// Given
		err := errors.New("test error")
		now := time.Now()

		// When
		n := notification.NewError(err)

		// Then
		assert.Equal(t, notification.LevelError, n.Type())
		assert.Equal(t, err, n.Error())
		assert.WithinDuration(t, now, n.Time(), time.Second)
	})
}

func TestNewInfo(t *testing.T) {
	t.Run("should create an info notification", func(t *testing.T) {
		// Given
		msg := "test message"
		now := time.Now()

		// When
		n := notification.NewInfo(msg)

		// Then
		assert.Equal(t, notification.LevelInfo, n.Type())
		assert.Equal(t, msg, n.Message())
		assert.WithinDuration(t, now, n.Time(), time.Second)
	})
}

func TestNewDebug(t *testing.T) {
	t.Run("should create a debug notification", func(t *testing.T) {
		// Given
		msg := "test debug message"
		now := time.Now()

		// When
		n := notification.NewDebug(msg)

		// Then
		assert.Equal(t, notification.LevelDebug, n.Type())
		assert.Equal(t, msg, n.Message())
		assert.WithinDuration(t, now, n.Time(), time.Second)
	})
}
