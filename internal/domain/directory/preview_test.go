package directory_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-marquis/it-happened/carrier"
	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/it-happened/eventest"
	"github.com/thomas-marquis/s3-box/internal/domain/connection_deck"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
	"github.com/thomas-marquis/s3-box/internal/testutil"
)

func assertContainsEvents(t *testing.T, events []event.Event, expectedPayloads ...event.Payload) {
	t.Helper()

	payloads := make([]event.Payload, len(events))
	for i, e := range events {
		payloads[i] = e.Payload()
	}

	for _, e := range expectedPayloads {
		assert.Contains(t, payloads, e)
	}
}

func TestPreview_Materialize(t *testing.T) {
	t.Run("should return a sequence of All carriers with an empty directory", func(t *testing.T) {
		// Given
		connID := connection_deck.NewConnectionID()

		dir := testutil.MakeDirectory(t, "data",
			testutil.WithRootParent(),
			testutil.WithConnectionId(connID),
			testutil.IsLoaded(),
		)

		dirMd, _ := directory.New(connID, "md", dir)
		dirHtml, _ := directory.New(connID, "html", dirMd)
		dirGo, _ := directory.New(connID, "go", dirMd)

		prev, err := dir.Preview()
		require.NoError(t, err)
		require.NoError(t, prev.AddFile("file1.txt", 0, time.Now()))
		require.NoError(t, prev.AddFile("file2.txt", 0, time.Now()))
		prevMd, err := prev.AddSubDirectory("md")
		require.NoError(t, err)
		require.NoError(t, prevMd.AddFile("file3.md", 0, time.Now()))
		prevHtml, err := prevMd.AddSubDirectory("html")
		require.NoError(t, err)
		prevGo, err := prevMd.AddSubDirectory("go")
		require.NoError(t, err)

		require.NoError(t, prevHtml.AddFile("file4.html", 0, time.Now()))
		require.NoError(t, prevHtml.AddFile("file5.html", 0, time.Now()))

		require.NoError(t, prevGo.AddFile("file6.go", 0, time.Now()))

		mat := directory.NewUploadMaterializer(prev, "/home/user/data/")

		// When
		res := mat.Materialize(directory.MaterializeSkip)

		// Then
		eventest.IsType(t, res, carrier.SequenceType)
		cs := res.Payload().(*carrier.Sequence)
		assert.Len(t, cs.Carried, 3)

		ca1 := cs.Carried[0]
		ca2 := cs.Carried[1]
		ca3 := cs.Carried[2]

		eventest.IsType(t, ca1, carrier.AllType)
		eventest.IsType(t, ca2, carrier.AllType)
		eventest.IsType(t, ca3, carrier.AllType)

		// layer 1
		eventest.ContainsExactlyAllPayloads(t, ca1.Payload().(*carrier.All).Carried,
			directory.UploadFileTriggered{Directory: dir, SrcPath: "/home/user/data/file1.txt"},
			directory.UploadFileTriggered{Directory: dir, SrcPath: "/home/user/data/file2.txt"},
			directory.CreateTriggered{ParentDirectory: dir, Directory: dirMd},
		)

		// layer 2
		eventest.ContainsExactlyAllPayloads(t, ca2.Payload().(*carrier.All).Carried,
			directory.UploadFileTriggered{Directory: dirMd, SrcPath: "/home/user/data/md/file3.md"},
			directory.CreateTriggered{ParentDirectory: dirMd, Directory: dirHtml},
			directory.CreateTriggered{ParentDirectory: dirMd, Directory: dirGo},
		)

		// layer 3
		eventest.ContainsExactlyAllPayloads(t, ca3.Payload().(*carrier.All).Carried,
			directory.UploadFileTriggered{Directory: dirHtml, SrcPath: "/home/user/data/md/html/file4.html"},
			directory.UploadFileTriggered{Directory: dirHtml, SrcPath: "/home/user/data/md/html/file5.html"},
			directory.UploadFileTriggered{Directory: dirGo, SrcPath: "/home/user/data/md/go/file6.go"},
		)
	})

	t.Run("should return a sequence of All carriers with a not empty directory", func(t *testing.T) {
		// Given
		connID := connection_deck.NewConnectionID()

		dir := testutil.MakeDirectory(t, "data",
			testutil.WithRootParent(),
			testutil.WithConnectionId(connID),
			testutil.IsLoaded(),
			testutil.WithFiles("file1.txt"),
			testutil.WithSubDirectory("md",
				testutil.IsLoaded(),
				testutil.WithSubDirectory("html",
					testutil.IsLoaded(),
					testutil.WithFiles("file5.html"),
				),
			),
		)

		dirMd, _ := dir.GetSubDirectoryByName("md")
		dirHtml, _ := dirMd.GetSubDirectoryByName("html")

		prev, err := dir.Preview()
		require.NoError(t, err)
		require.NoError(t, prev.AddFile("file1.txt", 0, time.Now()))
		require.NoError(t, prev.AddFile("file2.txt", 0, time.Now()))
		prevMd, err := prev.AddSubDirectory("md")
		require.NoError(t, err)
		require.NoError(t, prevMd.AddFile("file3.md", 0, time.Now()))
		prevHtml, err := prevMd.AddSubDirectory("html")
		require.NoError(t, err)
		prevGo, err := prevMd.AddSubDirectory("go")
		require.NoError(t, err)

		require.NoError(t, prevHtml.AddFile("file4.html", 0, time.Now()))
		require.NoError(t, prevHtml.AddFile("file5.html", 0, time.Now()))

		require.NoError(t, prevGo.AddFile("file6.go", 0, time.Now()))

		mat := directory.NewUploadMaterializer(prev, "/home/user/data/")

		// When
		res := mat.Materialize(directory.MaterializeSkip)

		// Then
		eventest.IsType(t, res, carrier.SequenceType)
		cs := res.Payload().(*carrier.Sequence)
		assert.Len(t, cs.Carried, 3)

		ca1 := cs.Carried[0]
		ca2 := cs.Carried[1]
		ca3 := cs.Carried[2]

		eventest.IsType(t, ca1, carrier.AllType)
		eventest.IsType(t, ca2, carrier.AllType)
		eventest.IsType(t, ca3, carrier.AllType)

		// layer 1
		eventest.ContainsExactlyAllPayloads(t, ca1.Payload().(*carrier.All).Carried,
			directory.UploadFileTriggered{Directory: dir, SrcPath: "/home/user/data/file2.txt"},
		)

		// layer 2
		eventest.ContainsExactlyAllPayloads(t, ca2.Payload().(*carrier.All).Carried,
			directory.UploadFileTriggered{Directory: dirMd, SrcPath: "/home/user/data/md/file3.md"},
			directory.CreateTriggered{ParentDirectory: dirMd, Directory: prevGo.Directory()},
		)

		// layer 3
		eventest.ContainsExactlyAllPayloads(t, ca3.Payload().(*carrier.All).Carried,
			directory.UploadFileTriggered{Directory: dirHtml, SrcPath: "/home/user/data/md/html/file4.html"},
			directory.UploadFileTriggered{Directory: prevGo.Directory(), SrcPath: "/home/user/data/md/go/file6.go"},
		)
	})
}

func TestPreview_GetByPath(t *testing.T) {
	// Given
	connID := connection_deck.NewConnectionID()

	dir := testutil.MakeDirectory(t, "project",
		testutil.WithRootParent(),
		testutil.WithConnectionId(connID),
		testutil.IsLoaded(),
		testutil.WithSubDirectory("src",
			testutil.IsLoaded(),
			testutil.WithSubDirectory("html",
				testutil.IsLoaded(),
			),
			testutil.WithSubDirectory("go"),
		),
	)

	prev, err := dir.Preview()
	require.NoError(t, err)
	prevSrc, err := prev.AddSubDirectory("src")
	require.NoError(t, err)
	prevHtml, err := prevSrc.AddSubDirectory("html")
	require.NoError(t, err)
	_, err = prevSrc.AddSubDirectory("go")
	require.NoError(t, err)

	t.Run("should return the directory for the given path when found", func(t *testing.T) {
		// When
		res, err := prev.GetByPath("/project/src/html/")

		// Then
		require.NoError(t, err)
		assert.Equal(t, prevHtml, res)
	})

	t.Run("should return an error when directory not found", func(t *testing.T) {
		// When
		_, err := prev.GetByPath("/project/src/js/")

		// Then
		assert.ErrorIs(t, err, directory.ErrNotFound)
	})

}
