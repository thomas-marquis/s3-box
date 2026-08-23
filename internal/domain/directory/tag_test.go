package directory_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/eventest"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func TestTagSet_Load(t *testing.T) {
	t.Run("should load successfully", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		require.Empty(t, ts.Get())

		// When
		evt := ts.Load()

		// Then
		eventest.AssertIsType(t, evt, directory.TagsLoadTriggeredType)
		assert.Equal(t, directory.TagsLoadTriggered{TagSet: ts}, evt.Payload())

		// When loaded
		sEvt := evt.NewFollowup(directory.TagsLoadSucceeded{
			TagSet: ts,
			Tags: []directory.Tag{
				{Key: "project", Value: "toto"},
				{Key: "team", Value: "dream"},
			},
		})
		ts.Notify(sEvt)

		// Then
		assert.Len(t, ts.Get(), 2)
		assert.Contains(t, ts.Get(), directory.Tag{
			Key:   "project",
			Value: "toto",
		})
		assert.Contains(t, ts.Get(), directory.Tag{
			Key:   "team",
			Value: "dream",
		})
	})
}

func TestTagSet_Add(t *testing.T) {
	t.Run("should add a new tag", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())

		t1 := directory.Tag{
			Key:   "mykey",
			Value: "toto",
		}
		t2 := directory.Tag{
			Key:   "other",
			Value: "lolo",
		}

		// When
		err1 := ts.Add("mykey", "toto")
		err2 := ts.Add("other", "lolo")
		evt := ts.Save()

		// Then
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Empty(t, ts.Get())

		eventest.AssertIsType(t, evt, directory.TagsSaveTriggeredType)
		assert.Equal(t, directory.TagsSaveTriggered{
			TagSet: ts,
			Tags:   []directory.Tag{t1, t2},
		}, evt.Payload())

		// When saved
		sEvt := evt.NewFollowup(directory.TagsSaveSucceeded{
			TagSet: ts,
			Tags:   []directory.Tag{t1, t2},
		})
		ts.Notify(sEvt)

		// Then
		assert.Len(t, ts.Get(), 2)
		assert.Contains(t, ts.Get(), t1)
		assert.Contains(t, ts.Get(), t2)
	})

	t.Run("should return an error when the tag already exists", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())

		t1 := directory.Tag{
			Key:   "mykey",
			Value: "toto",
		}

		ts.Notify(event.New(directory.TagsLoadSucceeded{
			TagSet: ts,
			Tags:   []directory.Tag{t1},
		}))
		require.Len(t, ts.Get(), 1)

		// When
		err := ts.Add("mykey", "toto")

		// Then
		assert.ErrorIs(t, err, directory.ErrTagKeyAlreadyExists)
	})

	t.Run("should return an error when the tag has already been added but still in pending", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())

		require.NoError(t, ts.Add("mykey", "toto"))
		require.Len(t, ts.Get(), 0)

		// When
		err := ts.Add("mykey", "soso")

		// Then
		assert.ErrorIs(t, err, directory.ErrTagOperationPending)
	})

	t.Run("should return an error when 10 tags are already attached to the file", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		existingTags := make([]directory.Tag, 10)
		for i := range 9 {
			existingTags = append(existingTags, directory.Tag{Key: fmt.Sprintf("key%d", i), Value: fmt.Sprintf("value%d", i)})
		}
		ts.Notify(event.New(directory.TagsLoadSucceeded{TagSet: ts, Tags: existingTags}))
		require.Len(t, ts.Get(), 10)

		// When
		err := ts.Add("key10", "value10")

		// Then
		assert.ErrorIs(t, err, directory.ErrMaxTagCountReached)
	})

	t.Run("should return an error when the tag key length is greater than 128 Unicode characters", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		longKey := string(make([]rune, 129))

		// When
		err := ts.Add(longKey, "value")

		// Then
		assert.ErrorIs(t, err, directory.ErrTagKeyTooLong)
	})

	t.Run("should return an error when the tag value length is greater than 256 Unicode characters", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		longValue := string(make([]rune, 257))

		// When
		err := ts.Add("key", longValue)

		// Then
		assert.ErrorIs(t, err, directory.ErrTagValueTooLong)
	})
}

func TestTagSet_Remove(t *testing.T) {
	t.Run("should remove an existing tag", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		t1 := directory.Tag{Key: "mykey", Value: "toto"}
		t2 := directory.Tag{Key: "other", Value: "lolo"}

		ts.Notify(event.New(directory.TagsLoadSucceeded{
			TagSet: ts,
			Tags:   []directory.Tag{t1, t2},
		}))
		require.Len(t, ts.Get(), 2)

		// When
		err := ts.Remove("mykey")
		require.NoError(t, err)
		evt := ts.Save()

		// Then
		eventest.AssertIsType(t, evt, directory.TagsSaveTriggeredType)
		assert.Equal(t, directory.TagsSaveTriggered{
			TagSet: ts,
			Tags:   []directory.Tag{t2},
		}, evt.Payload())

		// When saved
		sEvt := evt.NewFollowup(directory.TagsSaveSucceeded{
			TagSet: ts,
			Tags:   []directory.Tag{t2},
		})
		ts.Notify(sEvt)

		// Then
		assert.Len(t, ts.Get(), 1)
		assert.Contains(t, ts.Get(), t2)
	})

	t.Run("should return an error when the tag does not exist", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())

		// When
		err := ts.Remove("nonexistent")

		// Then
		assert.ErrorIs(t, err, directory.ErrTagKeyNotExists)
	})

	t.Run("should return an error when the tag key length is greater than 128 Unicode characters", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		longKey := string(make([]rune, 129))

		// When
		err := ts.Remove(longKey)

		// Then
		assert.ErrorIs(t, err, directory.ErrTagKeyTooLong)
	})

	t.Run("should return an error when the tag has already been removed but still in pending", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		t1 := directory.Tag{Key: "mykey", Value: "toto"}

		ts.Notify(event.New(directory.TagsLoadSucceeded{
			TagSet: ts,
			Tags:   []directory.Tag{t1},
		}))

		require.NoError(t, ts.Remove("mykey"))

		// When
		err := ts.Remove("mykey")

		// Then
		assert.ErrorIs(t, err, directory.ErrTagOperationPending)
	})
}

func TestTagSet_Update(t *testing.T) {
	t.Run("should update an existing tag's value", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		t1 := directory.Tag{Key: "mykey", Value: "toto"}
		t2 := directory.Tag{Key: "other", Value: "lolo"}

		ts.Notify(event.New(directory.TagsLoadSucceeded{
			TagSet: ts,
			Tags:   []directory.Tag{t1, t2},
		}))
		require.Len(t, ts.Get(), 2)

		// When
		err := ts.Update("mykey", "mykey", "newvalue")
		require.NoError(t, err)
		evt := ts.Save()

		// Then
		eventest.AssertIsType(t, evt, directory.TagsSaveTriggeredType)
		assert.Equal(t, directory.TagsSaveTriggered{
			TagSet: ts,
			Tags: []directory.Tag{
				{Key: "mykey", Value: "newvalue"},
				{Key: "other", Value: "lolo"},
			},
		}, evt.Payload())

		// When saved
		sEvt := evt.NewFollowup(directory.TagsSaveSucceeded{
			TagSet: ts,
			Tags: []directory.Tag{
				{Key: "mykey", Value: "newvalue"},
				{Key: "other", Value: "lolo"},
			},
		})
		ts.Notify(sEvt)

		// Then
		assert.Len(t, ts.Get(), 2)
		assert.Contains(t, ts.Get(), directory.Tag{Key: "mykey", Value: "newvalue"})
		assert.Contains(t, ts.Get(), directory.Tag{Key: "other", Value: "lolo"})
	})

	t.Run("should return an error when the tag does not exist", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())

		// When
		err := ts.Update("nonexistent", "newKey", "newvalue")

		// Then
		assert.ErrorIs(t, err, directory.ErrTagKeyNotExists)
	})

	t.Run("should return an error when the tag key length is greater than 128 Unicode characters", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		t1 := directory.Tag{Key: "mykey", Value: "toto"}

		ts.Notify(event.New(directory.TagsLoadSucceeded{
			TagSet: ts,
			Tags:   []directory.Tag{t1},
		}))

		longKey := string(make([]rune, 129))

		// When
		err := ts.Update("mykey", longKey, "value")

		// Then
		assert.ErrorIs(t, err, directory.ErrTagKeyTooLong)
	})

	t.Run("should return an error when the tag value length is greater than 256 Unicode characters", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		t1 := directory.Tag{Key: "mykey", Value: "toto"}

		ts.Notify(event.New(directory.TagsLoadSucceeded{
			TagSet: ts,
			Tags:   []directory.Tag{t1},
		}))

		longValue := string(make([]rune, 257))

		// When
		err := ts.Update("mykey", "mykey", longValue)

		// Then
		assert.ErrorIs(t, err, directory.ErrTagValueTooLong)
	})

	t.Run("should update the tag with the last set new value and key", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		t1 := directory.Tag{Key: "mykey", Value: "toto"}

		ts.Notify(event.New(directory.TagsLoadSucceeded{
			TagSet: ts,
			Tags:   []directory.Tag{t1},
		}))

		// When
		assert.NoError(t, ts.Update("mykey", "newKey", "newvalue"))
		assert.NoError(t, ts.Update("mykey", "another", "newvalue2"))

		// Then
		evt := ts.Save()
		pl := evt.Payload().(directory.TagsSaveTriggered)

		assert.Len(t, pl.Tags, 1)
		assert.Equal(t, "another", pl.Tags[0].Key)
		assert.Equal(t, "newvalue2", pl.Tags[0].Value)
	})

	t.Run("should return an error when the new key match an existing key", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		t1 := directory.Tag{Key: "mykey", Value: "toto"}
		t2 := directory.Tag{Key: "existingKey", Value: "lolo"}

		ts.Notify(event.New(directory.TagsLoadSucceeded{
			TagSet: ts,
			Tags:   []directory.Tag{t1, t2},
		}))

		// When
		err := ts.Update("mykey", "existingKey", "newvalue")

		// Then
		assert.ErrorIs(t, err, directory.ErrTagKeyAlreadyExists)

		evt := ts.Save()
		eventest.AssertIsType(t, evt, event.NothingHappenedType)
	})
}

func TestTagSet(t *testing.T) {
	t.Run("should handle add, update, and remove operations together", func(t *testing.T) {
		// Given
		ts := directory.NewTagSet("/data/", connection_deck.NewConnectionID())
		t1 := directory.Tag{Key: "mykey", Value: "toto"}
		t2 := directory.Tag{Key: "other", Value: "lolo"}

		ts.Notify(event.New(directory.TagsLoadSucceeded{
			TagSet: ts,
			Tags:   []directory.Tag{t1, t2},
		}))
		require.Len(t, ts.Get(), 2)

		// When
		require.NoError(t, ts.Add("newkey", "newvalue"))
		require.NoError(t, ts.Update("mykey", "mykey", "updated"))
		require.NoError(t, ts.Remove("other"))

		evt := ts.Save()

		// Then
		eventest.AssertIsType(t, evt, directory.TagsSaveTriggeredType)
		payload := evt.Payload().(directory.TagsSaveTriggered)
		assert.Len(t, payload.Tags, 2)
		assert.Contains(t, payload.Tags, directory.Tag{Key: "mykey", Value: "updated"})
		assert.Contains(t, payload.Tags, directory.Tag{Key: "newkey", Value: "newvalue"})
	})
}

func TestNewTag(t *testing.T) {
	t.Run("should create a new tag with valid key and value", func(t *testing.T) {
		// When
		tag, err := directory.NewTag("key", "value")

		// Then
		assert.NoError(t, err)
		assert.Equal(t, directory.Tag{Key: "key", Value: "value"}, tag)
	})

	t.Run("should return an error when the tag key length is greater than 128 Unicode characters", func(t *testing.T) {
		// Given
		longKey := string(make([]rune, 129))

		// When
		_, err := directory.NewTag(longKey, "value")

		// Then
		assert.ErrorIs(t, err, directory.ErrTagKeyTooLong)
	})

	t.Run("should return an error when the tag value length is greater than 256 Unicode characters", func(t *testing.T) {
		// Given
		longValue := string(make([]rune, 257))

		// When
		_, err := directory.NewTag("key", longValue)

		// Then
		assert.ErrorIs(t, err, directory.ErrTagValueTooLong)
	})
}
