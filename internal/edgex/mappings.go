package edgex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

func mappingSide(direction string) string {
	if direction == "reverse" {
		return "south"
	}
	return "north"
}

// ListMappings returns a page of mappings from support-mappings.
func (c *Client) ListMappings(
	ctx context.Context, filter, name string, offset, limit int,
) (map[string]any, error) {
	cfg := c.store.Get()
	path := "/api/v3/mapping/all"
	if filter != "" && filter != "all" {
		path = "/api/v3/mapping/" + filter + "/" + url.PathEscape(name)
	}
	raw, err := c.roundTrip(ctx, http.MethodGet, cfg.MappingsURL, path, pageQuery(offset, limit), "", nil)
	if err != nil {
		return nil, err
	}
	payload := asMap(raw)
	items, _ := payload["mappings"].([]any)
	if items == nil {
		items = []any{}
	}
	return map[string]any{
		"items":      items,
		"totalCount": intParam(payload, "totalCount", len(items)),
	}, nil
}

// SaveMapping creates or updates a mapping.
func (c *Client) SaveMapping(
	ctx context.Context, body map[string]any, creating bool,
) (any, error) {
	cfg := c.store.Get()
	method := http.MethodPut
	if creating {
		method = http.MethodPost
	}
	raw, err := c.requestJSON(ctx, cfg.MappingsURL, method, "/api/v3/mapping", nil, body)
	if err != nil {
		return nil, err
	}
	return asMap(raw), nil
}

// DeleteMapping removes one mapping by id.
func (c *Client) DeleteMapping(ctx context.Context, id string) error {
	cfg := c.store.Get()
	_, err := c.requestJSON(ctx, cfg.MappingsURL, http.MethodDelete,
		"/api/v3/mapping/"+url.PathEscape(id), nil, nil)
	return err
}

// LookupMapping resolves a single north/south point.
func (c *Client) LookupMapping(
	ctx context.Context, direction string, point map[string]string,
) (any, error) {
	cfg := c.store.Get()
	side := mappingSide(direction)
	query := url.Values{}
	query.Set(side+"DeviceName", point["device"])
	query.Set(side+"ProfileName", point["profile"])
	query.Set(side+"ResourceName", point["resource"])
	raw, err := c.roundTrip(ctx, http.MethodGet, cfg.MappingsURL,
		"/api/v3/mapping/"+direction, query, "", nil)
	if err != nil {
		return nil, err
	}
	return asMap(raw), nil
}

// BatchMappings resolves many points in one direction.
func (c *Client) BatchMappings(
	ctx context.Context, direction string, points []map[string]string,
) (any, error) {
	cfg := c.store.Get()
	side := mappingSide(direction)
	entries := make([]map[string]string, 0, len(points))
	for _, point := range points {
		entries = append(entries, map[string]string{
			side + "DeviceName":   point["device"],
			side + "ProfileName":  point["profile"],
			side + "ResourceName": point["resource"],
		})
	}
	body := map[string]any{side + "Points": entries}
	raw, err := c.requestJSON(ctx, cfg.MappingsURL, http.MethodPost,
		"/api/v3/mapping/batch/"+direction, nil, body)
	if err != nil {
		return nil, err
	}
	var list []any
	if json.Unmarshal(raw, &list) == nil {
		return list, nil
	}
	return asMap(raw), nil
}

// DeleteMappingScope removes every mapping under one side/kind/name scope.
func (c *Client) DeleteMappingScope(ctx context.Context, side, kind, name string) error {
	cfg := c.store.Get()
	path := "/api/v3/mapping/" + side + "/" + kind + "/" + url.PathEscape(name)
	_, err := c.requestJSON(ctx, cfg.MappingsURL, http.MethodDelete, path, nil, nil)
	return err
}
