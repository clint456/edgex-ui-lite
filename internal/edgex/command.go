package edgex

import (
	"context"
	"net/http"
	"net/url"
)

// Commands returns the command metadata for a node (core-command).
func (c *Client) Commands(ctx context.Context, node string) (any, error) {
	cfg := c.store.Get()
	raw, err := c.roundTrip(ctx, http.MethodGet, cfg.CommandURL,
		"/api/v3/node/name/"+url.PathEscape(node), nil, "", nil)
	if err != nil {
		return nil, err
	}
	payload := asMap(raw)
	if core, ok := payload["nodeCoreCommand"]; ok {
		return core, nil
	}
	return payload, nil
}

// ExecuteCommand issues a GET or PUT against a node command.
func (c *Client) ExecuteCommand(
	ctx context.Context, node, command, method string, body any,
) (any, error) {
	cfg := c.store.Get()
	path := "/api/v3/node/name/" + url.PathEscape(node) + "/" + url.PathEscape(command)
	if method == http.MethodGet {
		query := url.Values{}
		query.Set("ds-pushevent", "false")
		query.Set("ds-returnevent", "true")
		raw, err := c.roundTrip(ctx, http.MethodGet, cfg.CommandURL, path, query, "", nil)
		if err != nil {
			return nil, err
		}
		return asMap(raw), nil
	}
	raw, err := c.requestJSON(ctx, cfg.CommandURL, http.MethodPut, path, nil, body)
	if err != nil {
		return nil, err
	}
	return asMap(raw), nil
}
