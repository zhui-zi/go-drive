package onedrive

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/req"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"net/http"
	"strconv"
	"time"
)

func init() {
	drive_util.RegisterDrive(drive_util.DriveFactoryConfig{
		Type:        "onedrive",
		DisplayName: t("name"),
		README:      t("readme"),
		ConfigForm: []types.FormItem{
			{
				Field: "site", Label: t("form.site.label"), Type: "select", Description: t("form.site.description"),
				Options: []types.FormItemOption{
					{Name: t("form.site.global"), Value: "global", Title: t("form.site.global")},
					{Name: t("form.site.china"), Value: "china", Title: t("form.site.china")},
				},
				DefaultValue: "global", Required: true,
			},
			{
				Field: "tenant", Label: t("form.tenant.label"), Type: "select", Description: t("form.tenant.description"),
				Options: []types.FormItemOption{
					{Name: "common", Value: "common", Title: t("form.tenant.common")},
					{Name: "organizations", Value: "organizations", Title: t("form.tenant.organizations")},
					{Name: "consumers", Value: "consumers", Title: t("form.tenant.consumers")},
				},
				DefaultValue: "consumers", Required: true,
			},
			{Field: "client_id", Label: t("form.client_id.label"), Type: "text", Description: t("form.client_id.description"), Required: true},
			{Field: "client_secret", Label: t("form.client_secret.label"), Type: "password", Description: t("form.client_secret.description"), Required: true},
			{Field: "share_point", Label: t("form.share_point.label"), Type: "text", Description: t("form.share_point.description"), Required: false},
			{Field: "proxy_upload", Label: t("form.proxy_in.label"), Type: "checkbox", Description: t("form.proxy_in.description")},
			{Field: "proxy_download", Label: t("form.proxy_out.label"), Type: "checkbox", Description: t("form.proxy_out.description")},
			{Field: "cache_ttl", Label: t("form.cache_ttl.label"), Type: "text", Description: t("form.cache_ttl.description")},
		},
		Factory: drive_util.DriveFactory{Create: NewOneDrive, InitConfig: InitConfig, Init: Init},
	})
}

type OneDrive struct {
	c         *req.Client
	reqPrefix string

	cacheTTL time.Duration
	cache    drive_util.DriveCache

	uploadProxy   bool
	downloadProxy bool
}

func NewOneDrive(_ context.Context, config types.SM,
	driveUtils drive_util.DriveUtils) (types.IDrive, error) {
	resp, e := drive_util.OAuthGet(*oauthReq(driveUtils.Config, config), config, driveUtils.Data)
	if e != nil {
		return nil, e
	}

	cacheTtl := config.GetDuration("cache_ttl", -1)
	params, e := driveUtils.Data.Load("drive_id", "share_point_id")
	od := &OneDrive{
		cacheTTL:      cacheTtl,
		uploadProxy:   config.GetBool("proxy_upload"),
		downloadProxy: config.GetBool("proxy_download"),
	}
	if cacheTtl <= 0 {
		od.cache = drive_util.DummyCache()
	} else {
		od.cache = driveUtils.CreateCache(od.deserializeEntry, nil)
	}

	driveId := params["drive_id"]
	sharePointId := params["share_point_id"]

	if driveId == "" && sharePointId == "" {
		return nil, err.NewNotAllowedMessageError(t("drive_not_selected"))
	}

	site := getSiteConfig(config["site"])
	var reqPrefix string
	if driveId != "" {
		reqPrefix = utils.BuildURL(site.ApiBase+"/drives/{}", driveId)
	} else {
		reqPrefix = utils.BuildURL(site.ApiBase+"/sites/{}/drive", sharePointId)
	}

	od.c, e = req.NewClient(reqPrefix, nil, ifApiCallError, resp.Client())
	od.reqPrefix = reqPrefix

	return od, e
}

func (o *OneDrive) Meta(context.Context) types.DriveMeta {
	return types.DriveMeta{Writable: true}
}

func (o *OneDrive) Get(ctx context.Context, path string) (types.IEntry, error) {
	if utils.IsRootPath(path) {
		return &oneDriveEntry{id: "root", path: path, isDir: true}, nil
	}
	if cached, _ := o.cache.GetEntry(path); cached != nil {
		return cached, nil
	}
	resp, e := o.c.Get(ctx, utils.BuildURL("/root:/{}?expand=thumbnails", path), nil)
	if e != nil {
		return nil, e
	}
	entry, e := o.toEntry(resp)
	if e == nil {
		_ = o.cache.PutEntry(entry, o.cacheTTL)
	}
	return entry, nil
}

func (o *OneDrive) Save(ctx types.TaskCtx, path string, size int64,
	override bool, reader io.Reader) (types.IEntry, error) {
	var entry *oneDriveEntry = nil
	get, e := o.Get(ctx, path)
	if e != nil && !err.IsNotFoundError(e) {
		return nil, e
	}
	if e == nil {
		entry = get.(*oneDriveEntry)
	}
	if !override && entry != nil {
		return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
	}
	filename := utils.PathBase(path)
	if filename == "" {
		return nil, err.NewBadRequestError(i18n.T("drive.invalid_path"))
	}
	parent, e := o.Get(ctx, utils.PathParent(path))
	if e != nil {
		return nil, e
	}
	if size <= uploadChunkSize {
		if entry != nil {
			entry, e = o.uploadSmallFileOverride(ctx, entry.id, size, reader)
		} else {
			entry, e = o.uploadSmallFile(ctx, parent.(*oneDriveEntry).id, filename, size, reader)
		}
	} else {
		entry, e = o.uploadLargeFile(ctx, parent.(*oneDriveEntry).id, filename, size, override, reader)
	}
	if e != nil {
		return nil, e
	}
	_ = o.cache.Evict(parent.Path(), false)
	_ = o.cache.Evict(path, false)
	return entry, nil
}

func (o *OneDrive) MakeDir(ctx context.Context, path string) (types.IEntry, error) {
	if dir, e := o.Get(ctx, path); e == nil {
		if !dir.Type().IsDir() {
			return nil, err.NewNotAllowedMessageError(i18n.T("drive.file_exists"))
		}
		return dir, nil
	}
	parent := utils.PathParent(path)
	name := utils.PathBase(path)
	resp, e := o.c.Post(ctx, pathURL(parent)+"/children", nil, req.NewJsonBody(types.M{
		"name":                              name,
		"folder":                            types.M{},
		"@microsoft.graph.conflictBehavior": "fail",
	}))
	if e != nil {
		return nil, e
	}
	_ = o.cache.Evict(parent, false)
	return o.toEntry(resp)
}

func (o *OneDrive) Copy(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	from = drive_util.GetSelfEntry(o, from)
	if from == nil {
		return nil, err.NewUnsupportedError()
	}
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, o, to); e != nil {
			return nil, e
		}
	}
	ctx.Total(from.Size(), false)
	toParentPath := utils.PathParent(to)
	toName := utils.PathBase(to)
	resp, e := o.c.Post(ctx, idURL(from.(*oneDriveEntry).id)+"/copy", nil, req.NewJsonBody(types.M{
		"parentReference": types.M{"path": itemPath(toParentPath)},
		"name":            toName,
	}))
	if e != nil {
		return nil, e
	}
	_ = resp.Dispose()
	if resp.Status() == 202 {
		// we should wait for it to finish
		waitUrl := resp.Response().Header.Get("Location")
		if e := waitLongRunningAction(ctx, waitUrl); e != nil {
			return nil, e
		}
	}
	ctx.Progress(from.Size(), false)
	_ = o.cache.Evict(to, true)
	_ = o.cache.Evict(utils.PathParent(to), false)
	return o.Get(ctx, to)
}

func (o *OneDrive) Move(ctx types.TaskCtx, from types.IEntry, to string, override bool) (types.IEntry, error) {
	from = drive_util.GetSelfEntry(o, from)
	if from == nil {
		return nil, err.NewUnsupportedError()
	}
	if !override {
		if _, e := drive_util.RequireFileNotExists(ctx, o, to); e != nil {
			return nil, e
		}
	}
	ctx.Total(from.Size(), false)
	toParentPath := utils.PathParent(to)
	toName := utils.PathBase(to)
	resp, e := o.c.Request(ctx, "PATCH", idURL(from.(*oneDriveEntry).id), nil,
		req.NewJsonBody(types.M{
			"parentReference": types.M{"path": itemPath(toParentPath)},
			"name":            toName,
		}),
	)
	if e != nil {
		return nil, e
	}
	_ = o.cache.Evict(utils.PathParent(to), false)
	_ = o.cache.Evict(from.Path(), true)
	_ = o.cache.Evict(utils.PathParent(from.Path()), false)
	ctx.Progress(from.Size(), false)
	return o.toEntry(resp)
}

func (o *OneDrive) List(ctx context.Context, path string) ([]types.IEntry, error) {
	if cached, _ := o.cache.GetChildren(path); cached != nil {
		return cached, nil
	}
	entries := make([]types.IEntry, 0)
	reqPath := pathURL(path) + "/children?$expand=thumbnails"
	for {
		res := driveItems{}
		resp, e := o.c.Get(ctx, reqPath, nil)
		if e != nil {
			return nil, e
		}
		if e := resp.Json(&res); e != nil {
			return nil, e
		}
		for _, v := range res.Items {
			if v.Deleted != nil {
				continue
			}
			entries = append(entries, o.newEntry(v))
		}
		reqPath = res.NextPage
		if reqPath == "" {
			break
		}
	}
	_ = o.cache.PutChildren(path, entries, o.cacheTTL)
	return entries, nil
}

func (o *OneDrive) Delete(ctx types.TaskCtx, path string) error {
	entry, e := o.Get(ctx, path)
	if e != nil {
		return e
	}
	resp, e := o.c.Request(ctx, "DELETE", idURL(entry.(*oneDriveEntry).id), nil, nil)
	if e != nil {
		return e
	}
	_ = resp.Dispose()
	_ = o.cache.Evict(path, true)
	_ = o.cache.Evict(utils.PathParent(path), false)
	return nil
}

func (o *OneDrive) Upload(ctx context.Context, path string, size int64,
	override bool, config types.SM) (*types.DriveUploadConfig, error) {
	action := config["action"]
	switch action {
	case "CompleteUpload":
		_ = o.cache.Evict(path, false)
		_ = o.cache.Evict(utils.PathParent(path), false)
		return nil, nil
	default:
		if !override {
			if _, e := drive_util.RequireFileNotExists(ctx, o, path); e != nil {
				return nil, e
			}
		}
		if o.uploadProxy {
			return types.UseLocalProvider(size), nil
		}
		parent, e := o.Get(ctx, utils.PathParent(path))
		if e != nil {
			return nil, e
		}
		filename := utils.PathBase(path)
		sessionUrl, e := o.createUploadSession(ctx, parent.(*oneDriveEntry).id, filename, override)
		if e != nil {
			return nil, e
		}
		return &types.DriveUploadConfig{
			Provider: types.OneDriveProvider,
			Config:   types.SM{"url": sessionUrl},
		}, nil
	}
}

func (o *OneDrive) newEntry(item driveItem) *oneDriveEntry {
	modTime, _ := time.Parse(time.RFC3339, item.ModTime)
	thumbnailUrl := ""
	if supportThumbnail(item) &&
		item.Thumbnails != nil && len(item.Thumbnails) > 0 &&
		item.Thumbnails[0].Large != nil {
		thumbnailUrl = item.Thumbnails[0].Large.URL
	}
	return &oneDriveEntry{
		id:                   item.Id,
		path:                 item.Path(),
		isDir:                item.Folder != nil,
		size:                 item.Size,
		modTime:              utils.Millisecond(modTime),
		d:                    o,
		thumbnail:            thumbnailUrl,
		downloadUrl:          item.DownloadURL,
		downloadUrlExpiresAt: time.Now().Add(downloadUrlTTL).Unix(),
	}
}

type oneDriveEntry struct {
	id      string
	path    string
	isDir   bool
	size    int64
	modTime int64
	d       *OneDrive

	thumbnail string

	downloadUrl          string
	downloadUrlExpiresAt int64
}

func (o *oneDriveEntry) Path() string {
	return o.path
}

func (o *oneDriveEntry) Type() types.EntryType {
	if o.isDir {
		return types.TypeDir
	}
	return types.TypeFile
}

func (o *oneDriveEntry) Size() int64 {
	return o.size
}

func (o *oneDriveEntry) Meta() types.EntryMeta {
	return types.EntryMeta{
		Readable: true, Writable: true,
		Thumbnail: o.thumbnail,
	}
}

func (o *oneDriveEntry) ModTime() int64 {
	return o.modTime
}

func (o *oneDriveEntry) Drive() types.IDrive {
	return o.d
}

func (o *oneDriveEntry) Name() string {
	return utils.PathBase(o.path)
}

func (o *oneDriveEntry) GetReader(ctx context.Context) (io.ReadCloser, error) {
	u, resp, e := o.get(ctx)
	if e != nil {
		return nil, e
	}
	if resp != nil {
		return resp.Response().Body, nil
	}
	return drive_util.GetURL(ctx, u, nil)
}

func (o *oneDriveEntry) GetURL(ctx context.Context) (*types.ContentURL, error) {
	if o.isDir {
		return nil, err.NewNotAllowedError()
	}
	if o.downloadUrlExpiresAt <= time.Now().Unix() {
		u, resp, e := o.get(ctx)
		if e != nil {
			return nil, e
		}
		if resp != nil {
			_ = resp.Dispose()
			// fallback to GetReader
			return nil, err.NewUnsupportedError()
		}
		o.downloadUrl = u
		o.downloadUrlExpiresAt = time.Now().Add(downloadUrlTTL).Unix()
		_ = o.d.cache.PutEntry(o, o.d.cacheTTL)
	}
	return &types.ContentURL{URL: o.downloadUrl, Proxy: o.d.downloadProxy}, nil
}

func (o *oneDriveEntry) get(ctx context.Context) (string, req.Response, error) {
	resp, e := o.d.c.Get(ctx, pathURL(o.path)+"/content", nil)
	if e != nil {
		return "", nil, e
	}
	if resp.Status() == http.StatusOK {
		// OneDrive API returns 200 sometimes
		return "", resp, nil
	}
	if resp.Status() != http.StatusFound {
		_ = resp.Dispose()
		return "", nil, err.NewUnsupportedMessageError(fmt.Sprintf("%d", resp.Status()))
	}
	_ = resp.Dispose()
	u := resp.Response().Header.Get("Location")
	if u == "" {
		return "", nil, errors.New("invalid Location header")
	}
	return u, nil, nil
}

func (o *oneDriveEntry) EntryData() types.SM {
	return types.SM{
		"id": o.id,
		"du": o.downloadUrl,
		"de": strconv.FormatInt(o.downloadUrlExpiresAt, 10),
		"th": o.thumbnail,
	}
}
