package edgex

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/url"
)

// MetaKind describes a core-metadata resource family.
type MetaKind struct {
	Key        string
	Resource   string
	Entity     string
	Collection string
	UpdateVerb string
}

// Kinds maps a UI key to the matching core-metadata resource.
var Kinds = map[string]MetaKind{
	"node":    {"node", "node", "node", "nodes", http.MethodPatch},
	"service": {"service", "nodeservice", "service", "services", http.MethodPatch},
	"profile": {"profile", "profile", "profile", "profiles", http.MethodPut},
	"watcher": {"watcher", "provisionwatcher", "provisionWatcher", "provisionWatchers", http.MethodPatch},
}

// LookupKind resolves a UI key to a metadata kind.
func LookupKind(key string) (MetaKind, bool) {
	kind, ok := Kinds[key]
	return kind, ok
}

// ListMetadata returns a page of records with a normalized shape.
func (c *Client) ListMetadata(
	ctx context.Context, kind MetaKind, filter, value string, offset, limit int,
) (map[string]any, error) {
	cfg := c.store.Get()
	path := "/api/v3/" + kind.Resource + "/all"
	if kind.Key == "node" && (filter == "service" || filter == "profile") && value != "" {
		path = "/api/v3/node/" + filter + "/name/" + url.PathEscape(value)
	}
	raw, err := c.roundTrip(ctx, http.MethodGet, cfg.MetadataURL, path, pageQuery(offset, limit), "", nil)
	if err != nil {
		return nil, err
	}
	payload := asMap(raw)
	items, _ := payload[kind.Collection].([]any)
	if items == nil {
		items = []any{}
	}
	return map[string]any{
		"items":      items,
		"totalCount": intParam(payload, "totalCount", len(items)),
	}, nil
}

// GetMetadata loads one record by name.
func (c *Client) GetMetadata(ctx context.Context, kind MetaKind, name string) (any, error) {
	cfg := c.store.Get()
	raw, err := c.roundTrip(ctx, http.MethodGet, cfg.MetadataURL,
		"/api/v3/"+kind.Resource+"/name/"+url.PathEscape(name), nil, "", nil)
	if err != nil {
		return nil, err
	}
	payload := asMap(raw)
	if entity, ok := payload[kind.Entity]; ok {
		return entity, nil
	}
	return payload, nil
}

// CountMetadata counts records through the list endpoint's totalCount.
func (c *Client) CountMetadata(ctx context.Context, kind MetaKind) (int, error) {
	cfg := c.store.Get()
	raw, err := c.roundTrip(ctx, http.MethodGet, cfg.MetadataURL,
		"/api/v3/"+kind.Resource+"/all", pageQuery(0, 1), "", nil)
	if err != nil {
		return 0, err
	}
	payload := asMap(raw)
	items, _ := payload[kind.Collection].([]any)
	return intParam(payload, "totalCount", len(items)), nil
}

// SaveMetadata creates or updates a record using the EdgeX batch envelope.
func (c *Client) SaveMetadata(
	ctx context.Context, kind MetaKind, record map[string]any, creating bool,
) error {
	cfg := c.store.Get()
	method := kind.UpdateVerb
	if creating {
		method = http.MethodPost
	}
	envelope := []map[string]any{{"apiVersion": "v3", kind.Entity: record}}
	_, err := c.requestJSON(ctx, cfg.MetadataURL, method, "/api/v3/"+kind.Resource, nil, envelope)
	return err
}

// DeleteMetadata removes a record by name.
func (c *Client) DeleteMetadata(ctx context.Context, kind MetaKind, name string) error {
	cfg := c.store.Get()
	_, err := c.requestJSON(ctx, cfg.MetadataURL, http.MethodDelete,
		"/api/v3/"+kind.Resource+"/name/"+url.PathEscape(name), nil, nil)
	return err
}

// UploadProfile sends a node profile as a YAML file (multipart uploadfile).
func (c *Client) UploadProfile(
	ctx context.Context, filename, yamlText string, update bool,
) error {
	cfg := c.store.Get()
	if filename == "" {
		filename = "node-profile.yaml"
	}
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return &APIError{Message: "构建上传请求失败"}
	}
	if _, err := part.Write([]byte(yamlText)); err != nil {
		return &APIError{Message: "写入上传内容失败"}
	}
	if err := writer.Close(); err != nil {
		return &APIError{Message: "构建上传请求失败"}
	}
	method := http.MethodPost
	if update {
		method = http.MethodPut
	}
	_, err = c.roundTrip(ctx, method, cfg.MetadataURL,
		"/api/v3/profile/uploadfile", nil, writer.FormDataContentType(), buffer.Bytes())
	return err
}
