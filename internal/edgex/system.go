package edgex

import (
	"context"
	"net/http"
	"net/url"
)

// ServiceHealth is one probed service row.
type ServiceHealth struct {
	Service string `json:"service"`
	URL     string `json:"url"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Registration is one core-keeper service registration.
type Registration struct {
	ServiceID string `json:"serviceId"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Status    string `json:"status"`
}

type probeTarget struct {
	name string
	url  string
}

func (c *Client) probeTargets() []probeTarget {
	cfg := c.store.Get()
	return []probeTarget{
		{"core-metadata", cfg.MetadataURL},
		{"core-data", cfg.DataURL},
		{"core-command", cfg.CommandURL},
		{"support-mappings", cfg.MappingsURL},
		{"core-keeper", cfg.KeeperURL},
	}
}

// Health probes every configured EdgeX service ping endpoint.
func (c *Client) Health(ctx context.Context) []ServiceHealth {
	targets := c.probeTargets()
	results := make([]ServiceHealth, 0, len(targets))
	for _, target := range targets {
		row := ServiceHealth{Service: target.name, URL: target.url, Status: "在线"}
		if _, err := c.roundTrip(ctx, http.MethodGet, target.url, "/api/v3/ping", nil, "", nil); err != nil {
			row.Status = "不可用"
			row.Message = err.Error()
		}
		results = append(results, row)
	}
	return results
}

// Registrations lists services registered in core-keeper.
func (c *Client) Registrations(ctx context.Context) ([]Registration, error) {
	cfg := c.store.Get()
	var body struct {
		Registrations []Registration `json:"registrations"`
	}
	if err := c.getJSON(ctx, cfg.KeeperURL, "/api/v3/registry/all", nil, &body); err != nil {
		return nil, err
	}
	if body.Registrations == nil {
		body.Registrations = []Registration{}
	}
	return body.Registrations, nil
}

func (c *Client) systemGet(ctx context.Context, suffix, services string) (any, error) {
	cfg := c.store.Get()
	query := url.Values{}
	if services != "" {
		query.Set("services", services)
	}
	raw, err := c.roundTrip(ctx, http.MethodGet, cfg.SystemURL, "/api/v3/system/"+suffix, query, "", nil)
	if err != nil {
		return nil, err
	}
	return decodeAny(raw), nil
}

// SystemMetrics fetches sys-mgmt-agent metrics for the given services.
func (c *Client) SystemMetrics(ctx context.Context, services string) (any, error) {
	return c.systemGet(ctx, "metrics", services)
}

// SystemConfig fetches sys-mgmt-agent configuration for the given services.
func (c *Client) SystemConfig(ctx context.Context, services string) (any, error) {
	return c.systemGet(ctx, "config", services)
}

// SystemOperation starts, stops or restarts services through sys-mgmt-agent.
func (c *Client) SystemOperation(ctx context.Context, serviceName, action string) (any, error) {
	cfg := c.store.Get()
	body := []map[string]string{{"apiVersion": "v3", "serviceName": serviceName, "action": action}}
	raw, err := c.requestJSON(ctx, cfg.SystemURL, http.MethodPost, "/api/v3/system/operation", nil, body)
	if err != nil {
		return nil, err
	}
	return decodeAny(raw), nil
}
