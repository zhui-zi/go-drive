package server

import (
	"context"
	"fmt"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/task"
	"go-drive/common/types"
	"go-drive/common/utils"
	"go-drive/drive"
	"go-drive/server/search"
	"go-drive/server/thumbnail"
	"go-drive/storage"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxProxySizeKey = "proxy.maxSize"
)

func InitDriveRoutes(
	router gin.IRouter,
	access *drive.Access,
	searcher *search.Service,
	config common.Config,
	thumbnail *thumbnail.Maker,
	signer *utils.Signer,
	chunkUploader *ChunkUploader,
	runner task.Runner,
	tokenStore types.TokenStore,
	optionsDAO *storage.OptionsDAO) error {

	dr := driveRoute{
		config:        config,
		access:        access,
		searcher:      searcher,
		chunkUploader: chunkUploader,
		thumbnail:     thumbnail,
		runner:        runner,
		signer:        signer,
		options:       optionsDAO,
	}

	// get file content
	router.HEAD("/content/*path", dr.getContent)
	router.GET("/content/*path", dr.getContent)
	router.GET("/thumbnail/*path", dr.getThumbnail)

	r := router.Group("/", TokenAuth(tokenStore))

	// list entries/drives
	r.GET("/entries/*path", dr.list)
	// get entry info
	r.GET("/entry/*path", dr.get)
	// mkdir
	r.POST("/mkdir/*path", dr.makeDir)
	// copy file
	r.POST("/copy", dr.copyEntry)
	// move file
	r.POST("/move", dr.move)
	// deleteEntry entry
	r.DELETE("/entry/*path", dr.deleteEntry)
	// get upload config
	r.POST("/upload/*path", dr.upload)
	// write file
	r.PUT("/content/*path", dr.writeContent)
	// chunk upload request
	r.POST("/chunk", dr.chunkUploadRequest)
	// chunk upload
	r.PUT("/chunk/:id/:seq", dr.chunkUpload)
	// chunk upload complete
	r.POST("/chunk-content/*path", dr.chunkUploadComplete)
	// delete chunk upload
	r.DELETE("/chunk/:id", dr.deleteChunkUpload)
	// search
	r.GET("/search/*path", dr.search)

	return nil
}

type driveRoute struct {
	config common.Config

	access   *drive.Access
	searcher *search.Service

	chunkUploader *ChunkUploader
	thumbnail     *thumbnail.Maker
	runner        task.Runner
	signer        *utils.Signer

	options *storage.OptionsDAO
}

func (dr *driveRoute) getDrive(c *gin.Context) types.IDrive {
	return dr.access.GetDrive(c.Request, GetSession(c))
}

func (dr *driveRoute) list(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	d := dr.getDrive(c)

	entry, e := d.Get(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	entries, e := d.List(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	res := make([]entryJson, 0, len(entries)+1)
	res = append(res, *newEntryJson(entry))
	for _, v := range entries {
		res = append(res, *newEntryJson(v))
	}
	SetResult(c, res)
}

func (dr *driveRoute) get(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	entry, e := dr.getDrive(c).Get(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, newEntryJson(entry))
}

func (dr *driveRoute) makeDir(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	entry, e := dr.getDrive(c).MakeDir(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, newEntryJson(entry))
}

func (dr *driveRoute) copyEntry(c *gin.Context) {
	drive_ := dr.getDrive(c)
	from := utils.CleanPath(c.Query("from"))
	fromEntry, e := drive_.Get(c.Request.Context(), from)
	if e != nil {
		_ = c.Error(e)
		return
	}
	to := utils.CleanPath(c.Query("to"))
	if e := checkCopyOrMove(from, to); e != nil {
		_ = c.Error(e)
		return
	}
	override := c.Query("override")
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		r, e := drive_.Copy(ctx, fromEntry, to, override != "")
		if e != nil {
			return nil, e
		}
		return newEntryJson(r), nil
	}, 2*time.Second, task.WithNameGroup(from+" -> "+to, "drive/copy"))

	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) move(c *gin.Context) {
	drive_ := dr.getDrive(c)
	from := utils.CleanPath(c.Query("from"))
	fromEntry, e := drive_.Get(c.Request.Context(), from)
	if e != nil {
		_ = c.Error(e)
		return
	}
	to := utils.CleanPath(c.Query("to"))
	if e := checkCopyOrMove(from, to); e != nil {
		_ = c.Error(e)
		return
	}
	override := c.Query("override")
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		r, e := drive_.Move(ctx, fromEntry, to, override != "")
		if e != nil {
			return nil, e
		}
		return newEntryJson(r), nil
	}, 2*time.Second, task.WithNameGroup(from+" -> "+to, "drive/move"))

	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func checkCopyOrMove(from, to string) error {
	if from == to {
		return err.NewNotAllowedMessageError(i18n.T("api.drive.copy_to_same_path_not_allowed"))
	}
	if strings.HasPrefix(to, from) && utils.PathDepth(from) != utils.PathDepth(to) {
		return err.NewNotAllowedMessageError(i18n.T("api.drive.copy_to_child_path_not_allowed"))
	}
	return nil
}

func (dr *driveRoute) deleteEntry(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		return nil, dr.getDrive(c).Delete(ctx, path)
	}, 2*time.Second, task.WithNameGroup(path, "drive/delete"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) upload(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	override := c.Query("override")
	size := utils.ToInt64(c.Query("size"), -1)
	request := make(types.SM, 0)
	if e := c.Bind(&request); e != nil {
		_ = c.Error(e)
		return
	}
	config, e := dr.getDrive(c).Upload(c.Request.Context(), path, size, override != "", request)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if config != nil {
		SetResult(c, uploadConfig{config.Provider, config.Config})
	}
}

func (dr *driveRoute) getContent(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	file, e := dr.getDrive(c).Get(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if content, ok := file.(types.IContent); ok {
		useProxy := c.Query("proxy")
		proxyMaxSize := dr.options.GetValue(maxProxySizeKey).DataSize(-1)

		if proxyMaxSize > 0 && file.Size() > proxyMaxSize {
			useProxy = ""
		}
		if e := drive_util.DownloadIContent(c.Request.Context(), content, c.Writer, c.Request, useProxy != ""); e != nil {
			_ = c.Error(e)
			return
		}
		return
	}
	_ = c.Error(err.NewNotAllowedError())
}

func (dr *driveRoute) getThumbnail(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	if !utils.CheckSignature(dr.signer, c.Request, path) {
		_ = c.Error(err.NewNotFoundError())
		return
	}
	entry, e := dr.getDrive(c).Get(c.Request.Context(), path)
	if e != nil {
		_ = c.Error(e)
		return
	}
	if entry.Meta().Props != nil && entry.Meta().Thumbnail != "" {
		c.Redirect(http.StatusFound, entry.Meta().Thumbnail)
		return
	}
	makeCtx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	file, e := dr.thumbnail.Make(makeCtx, entry)
	if e != nil {
		_ = c.Error(e)
		return
	}
	defer func() { _ = file.Close() }()
	c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(dr.config.Thumbnail.TTL.Seconds())))
	c.Header("Content-Type", file.MimeType())
	http.ServeContent(c.Writer, c.Request, "", file.ModTime(), file)
}

func (dr *driveRoute) writeContent(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	override := c.Query("override")
	size := utils.ToInt64(c.GetHeader("Content-Length"), -1)
	defer func() { _ = c.Request.Body.Close() }()
	file, e := drive_util.CopyReaderToTempFile(task.DummyContext(), c.Request.Body, dr.config.TempDir)
	if e != nil {
		_ = c.Error(e)
		return
	}
	stat, e := file.Stat()
	if e != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		_ = c.Error(e)
		return
	}
	if size != stat.Size() {
		_ = file.Close()
		_ = os.Remove(file.Name())
		_ = c.Error(err.NewBadRequestError(i18n.T("api.drive.invalid_file_size")))
		return
	}
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		tempFile := utils.NewTempFile(file)
		defer func() {
			_ = tempFile.Close()
			_ = os.Remove(tempFile.Name())
		}()
		return dr.getDrive(c).Save(ctx, path, size, override != "", tempFile)
	}, 2*time.Second, task.WithNameGroup(path, "drive/write"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) chunkUploadRequest(c *gin.Context) {
	size := utils.ToInt64(c.Query("size"), -1)
	chunkSize := utils.ToInt64(c.Query("chunkSize"), -1)
	if size <= 0 || chunkSize <= 0 {
		_ = c.Error(err.NewBadRequestError(i18n.T("api.drive.invalid_size_or_chunk_size")))
		return
	}
	upload, e := dr.chunkUploader.CreateUpload(size, chunkSize)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, upload)
}

func (dr *driveRoute) chunkUpload(c *gin.Context) {
	id := c.Param("id")
	seq, e := strconv.Atoi(c.Param("seq"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	if e := dr.chunkUploader.ChunkUpload(id, seq, c.Request.Body); e != nil {
		_ = c.Error(e)
	}
}

func (dr *driveRoute) chunkUploadComplete(c *gin.Context) {
	path := utils.CleanPath(c.Param("path"))
	id := c.Query("id")
	t, e := dr.runner.ExecuteAndWait(func(ctx types.TaskCtx) (interface{}, error) {
		file, e := dr.chunkUploader.CompleteUpload(id, ctx)
		if e != nil {
			return nil, e
		}
		stat, e := file.Stat()
		if e != nil {
			_ = file.Close()
			return nil, e
		}
		ctx.Progress(0, true)
		tempFile := utils.NewTempFile(file)
		entry, e := dr.getDrive(c).Save(ctx, path, stat.Size(), true, tempFile)
		if e != nil {
			_ = tempFile.Close()
			return nil, e
		}
		_ = tempFile.Close()
		e = dr.chunkUploader.DeleteUpload(id)
		return newEntryJson(entry), nil
	}, 2*time.Second, task.WithNameGroup(path, "drive/chunk-merge"))
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, t)
}

func (dr *driveRoute) deleteChunkUpload(c *gin.Context) {
	id := c.Param("id")
	if e := dr.chunkUploader.DeleteUpload(id); e != nil {
		_ = c.Error(e)
	}
}

func (dr *driveRoute) search(c *gin.Context) {
	root := utils.CleanPath(c.Param("path"))
	query := c.Query("q")
	next := utils.ToInt(c.Query("next"), 0)

	r, e := dr.searcher.Search(
		c.Request.Context(), root, query, next,
		dr.access.GetPerms().Filter(GetSession(c)),
	)
	if e != nil {
		_ = c.Error(e)
		return
	}
	SetResult(c, r)
}

type entryJson struct {
	Path    string          `json:"path"`
	Name    string          `json:"name"`
	Type    types.EntryType `json:"type"`
	Size    int64           `json:"size"`
	Meta    types.M         `json:"meta"`
	ModTime int64           `json:"modTime"`
}

func newEntryJson(e types.IEntry) *entryJson {
	entryMeta := e.Meta()
	meta := utils.CopyMap(entryMeta.Props)
	meta["writable"] = entryMeta.Writable
	if entryMeta.Thumbnail != "" {
		meta["thumbnail"] = entryMeta.Thumbnail
	}
	if entryMeta.Thumbnail == "" {
		// thumbnail is true
		// so the thumbnail is generated by the entry self
		if te := thumbnail.GetWrappedThumbnailEntry(e); te != nil {
			meta["thumbnail"] = true
		}
	}
	return &entryJson{
		Path:    e.Path(),
		Name:    utils.PathBase(e.Path()),
		Type:    e.Type(),
		Size:    e.Size(),
		Meta:    meta,
		ModTime: e.ModTime(),
	}
}

type uploadConfig struct {
	Provider string      `json:"provider"`
	Config   interface{} `json:"config"`
}
