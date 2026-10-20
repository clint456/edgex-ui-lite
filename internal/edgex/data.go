package edgex

import (
	"context"
	"net/http"
	"net/url"
)

var dataCollections = map[string]string{
	"event":   "events",
	"reading": "readings",
}

// DataKindValid reports whether kind is a core-data resource.
func DataKindValid(kind string) bool {
	_, ok := dataCollections[kind]
	return ok
}

// ListData returns a page of events or readings, optionally filtered by node.
func (c *Client) ListData(
	ctx context.Context, kind, node string, offset, limit int,
) (map[string]any, error) {
	cfg := c.store.Get()
	path := "/api/v3/" + kind + "/all"
	if node != "" {
		path = "/api/v3/" + kind + "/node/name/" + url.PathEscape(node)
	}
	raw, err := c.roundTrip(ctx, http.MethodGet, cfg.DataURL, path, pageQuery(offset, limit), "", nil)
	if err != nil {
		return nil, err
	}
	payload := asMap(raw)
	items, _ := payload[dataCollections[kind]].([]any)
	if items == nil {
		items = []any{}
	}
	return map[string]any{
		"items":      items,
		"totalCount": intParam(payload, "totalCount", len(items)),
	}, nil
}

// CountData returns the total number of events or readings.
func (c *Client) CountData(ctx context.Context, kind string) (int, error) {
	cfg := c.store.Get()
	var body struct {
		Count int `json:"count"`
	}
	if err := c.getJSON(ctx, cfg.DataURL, "/api/v3/"+kind+"/count", nil, &body); err != nil {
		return 0, err
	}
	return body.Count, nil
}
