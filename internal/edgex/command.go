package edgex

import (
	"context"
	"net/http"
	"net/url"
)

// DeviceCommands returns the command metadata for a device (core-command).
func (c *Client) DeviceCommands(ctx context.Context, device string) (any, error) {
	cfg := c.store.Get()
	raw, err := c.roundTrip(ctx, http.MethodGet, cfg.CommandURL,
		"/api/v3/device/name/"+url.PathEscape(device), nil, "", nil)
	if err != nil {
		return nil, err
	}
	payload := asMap(raw)
	if core, ok := payload["deviceCoreCommand"]; ok {
		return core, nil
	}
	return payload, nil
}

// ExecuteCommand issues a GET or PUT against a device command.
func (c *Client) ExecuteCommand(
	ctx context.Context, device, command, method string, body any,
) (any, error) {
	cfg := c.store.Get()
	path := "/api/v3/device/name/" + url.PathEscape(device) + "/" + url.PathEscape(command)
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
