package s3_test

import (
	"context"
	"strings"
	"testing"

	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/domain/shared/event"
	"github.com/thomas-marquis/s3-box/internal/infrastructure/s3"
	"github.com/thomas-marquis/s3-box/internal/testutil"
	"go.uber.org/mock/gomock"
)

func TestS3EventHandler_upload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testcontainers tests in short mode")
	}

	ctx := context.Background()
	endpoint, terminate := testutil.SetupS3testContainer(ctx, t)
	defer terminate()
	client := testutil.SetupS3Client(t, endpoint)

	t.Run("should compute and emit a preview when some content already exists", func(t *testing.T) {
		// Given
		bucket := testutil.FakeRandomBucketName()
		testutil.SetupS3Bucket(ctx, t, client, bucket, []testutil.FakeS3Object{
			{Key: "mydir/"},
			{Key: "mydir/file1.txt"},
			{Key: "mydir/file2.txt"},
			{Key: "mydir/subdir/"},
			{Key: "mydir/subdir/otherfile1.txt", Body: strings.NewReader("toto")},
			{Key: "mydir/subdir/otherfile2.txt", Body: strings.NewReader("lolo")},
			{Key: "mydir/dir2/something1.md", Body: strings.NewReader("hello")},
			{Key: "mydir/dir2/something2.md", Body: strings.NewReader("hei")},
		})
		fakeDeck := testutil.FakeDeckWithAwsConnection(t, endpoint, bucket)

		mydir := testutil.MakeDirectory(t, "mydir",
			testutil.WithConnectionId(testutil.FakeAwsConnectionId),
			testutil.IsLoaded(),
			testutil.WithRootParent(),
			testutil.WithFiles("file1.txt", "file2.txt"),
			testutil.WithSubDirectory("subdir",
				testutil.WithFiles("otherfile1.txt", "otherfile2.txt")),
			testutil.WithSubDirectory("dir2",
				testutil.WithFiles("something1.md", "something2.md")))

		itemsToUpload := []*directory.FsItem{
			{Name: "file2.txt"},
			{Name: "file3.txt"},
			{Name: "newdir", IsDir: true, Children: []*directory.FsItem{
				{Name: "subfile.txt"},
			}},
			{Name: "emptydir", IsDir: true},
			{Name: "dir2", IsDir: true, Children: []*directory.FsItem{ // TODO: move this in another test case
				{Name: "new.txt"},
			}},
		}

		// skip:

		expectedPreviews := map[directory.UploadMode]directory.UploadedItemPreview{
			directory.UploadModeSkip: {
				Name:       "",
				IsDir:      false,
				IsNew:      false,
				IsReplaced: false,
				Children:   nil,
			},
			directory.UploadModeDuplicate: {},
			directory.UploadModeReplace:   {},
		}

		fakeEventChan := make(chan event.Event, 1)
		defer close(fakeEventChan)
		mockBus, mockConnRepo, mockNotifRepo := setupMocks(t, fakeDeck, fakeEventChan)

		done := make(chan struct{})

		mockBus.EXPECT().
			Publish(gomock.Cond(func(evt event.Event) bool {
				close(done) // TODO:
				return true
			})).
			Times(1)

		eh := s3.NewS3EventHandler(mockConnRepo, mockBus, mockNotifRepo)
		defer eh.Destroy()
		eh.Listen()

		// When
		fakeEventChan <- event.New(directory.UploadTriggered{Directory: mydir, Items: itemsToUpload})

		// Then
		testutil.AssertEventually(t, done)
	})
}
