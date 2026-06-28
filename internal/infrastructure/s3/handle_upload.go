package s3

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thomas-marquis/it-happened/event"
	"github.com/thomas-marquis/s3-box/internal/domain/directory"
)

func (h *EventHandler) handleUploadFile(e event.Event) {
	ctx := e.Context()
	pl := e.Payload().(directory.UploadFileTriggered)

	handleError := func(err error) {
		h.notifier.NotifyError(fmt.Errorf("failed uploading file: %w", err))
		h.bus.Publish(e.NewFollowup(directory.UploadFileFailed{Err: err, Directory: pl.Directory}))
	}

	client, err := h.clientFactory.Get(ctx, pl.Directory.ConnectionID())
	if err != nil {
		handleError(err)
		return
	}

	localFile, err := os.Open(pl.SrcPath)
	if err != nil {
		handleError(err)
		return
	}
	defer localFile.Close() //nolint:errcheck

	info, err := localFile.Stat()
	if err != nil {
		handleError(fmt.Errorf("failed reading the file info: %w", err))
		return
	}
	if info.IsDir() {
		handleError(fmt.Errorf("failed opening the file to upload: path is a directory"))
		return
	}

	fileName := filepath.Base(pl.SrcPath)
	newFile, err := directory.NewFile(fileName, pl.Directory,
		directory.WithFileSize(int(info.Size())),
		directory.WithFileLastModified(info.ModTime()))
	if err != nil {
		handleError(err)
		return
	}

	if err := client.Upload(ctx, mapFileToKey(newFile), localFile); err != nil {
		handleError(fmt.Errorf("failed uploading file: %w", err))
		return
	}

	h.bus.Publish(e.NewFollowup(directory.UploadFileSucceeded{File: newFile, Directory: pl.Directory}))
}

//func (h *EventHandler) handleUploadTriggered(e event.Event) {
//	pl := e.Payload().(directory.UploadTriggered)
//
//	handleError := func(err error) {
//		h.notifier.NotifyError(fmt.Errorf("upload failed: %w", err))
//		h.bus.Publish(e.NewFollowup(directory.UploadFailed{Directory: pl.Directory, Err: err}))
//	}
//
//	client, err := h.clientFactory.Get(e.Context(), pl.Directory.ConnectionID())
//	if err != nil {
//		handleError(err)
//		return
//	}
//
//	if len(pl.Items) == 1 && !pl.Items[0].IsDir {
//		if err := h.doUpload(e.Context(), pl.Directory, pl.Items); err != nil {
//			handleError(err)
//		}
//		return
//	}
//
//	// 1. check the directories are empty. Emit a failure event if it's not the case, as well as a load event for the directory (to reload it)
//
//	// 2. Build search keys for all items
//	searchKeys := mapPathToSearchKey(pl.Directory.Path())
//	res, err := client.ListObjects(e.Context(), searchKeys, true)
//	if err != nil {
//		handleError(err)
//		return
//	}
//
//	if !res.IsEmpty() {
//		errs := make([]error, 0)
//		for _, item := range pl.Items {
//			if !item.IsDir {
//				continue
//			}
//
//			for _, key := range res.Keys {
//				if key == strings.ReplaceAll(item.RelPath(), string(filepath.Separator), "/") {
//					errs = append(errs, fmt.Errorf("directory %s already exists on the server and is not empty", item.Name))
//				}
//			}
//		}
//
//		if len(errs) > 0 {
//			handleError(fmt.Errorf("failed to upload: %w", errors.Join(errs...)))
//			return
//		}
//	}
//
//	// 4. Merge the search result with the items following each update mode rule
//
//	//evt := e.NewFollowup(directory.UploadPreviewed{
//	//	Directory:       pl.Directory,
//	//	Previews:        nil,
//	//	UploadableItems: pl.Items,
//	//})
//	//h.bus.Publish(evt)
//}
