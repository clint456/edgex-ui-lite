// Package edgex contains direct HTTP clients for the EdgeX microservices.
package edgex

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/edge-hy/edge-hy-console/internal/config"
)

// APIError is a failed EdgeX request that is safe to show at the UI boundary.
type APIError struct {
	Status  int
	Path    string
	Message string
}

func (e *APIError) Error() string {
	if e.Status == 0 {
		return e.Message
	}
	return e.Message + "（HTTP " + strconv.Itoa(e.Status) + "，" + e.Path + "）"
}

// Client talks to the EdgeX services using the configuration store.
type Client struct {
	store *config.Store
	http  *http.Client
}

// NewClient builds a client bound to the shared configuration store.
func NewClient(store *config.Store) *Client {
	return &Client{store: store, http: &http.Client{}}
}

// Store exposes the configuration store (used by the settings handlers).
func (c *Client) Store() *config.Store { return c.store }

var statusHints = map[int]string{
	http.StatusBadRequest:          "请求数据无效",
	http.StatusUnauthorized:        "Token 无效或已过期",
	http.StatusForbidden:           "没有操作权限",
	http.StatusNotFound:            "资源或接口不存在",
	http.StatusConflict:            "资源已存在或冲突",
	http.StatusServiceUnavailable:  "服务不可用",
	http.StatusInternalServerError: "服务内部错误",
}

func errorMessage(data []byte, status int) string {
	var body struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &body) == nil && strings.TrimSpace(body.Message) != "" {
		return strings.TrimSpace(body.Message)
	}
	text := strings.TrimSpace(string(data))
	if text == "" || strings.HasPrefix(strings.ToLower(text), "<") {
		if hint, ok := statusHints[status]; ok {
			return hint
		}
		return "服务请求失败"
	}
	if len(text) > 300 {
		text = text[:300]
	}
	return text
}

func correlationID() string {
	buffer := make([]byte, 16)
	_, _ = rand.Read(buffer)
	return hex.EncodeToString(buffer)
}

func (c *Client) roundTrip(
	ctx context.Context, method, base, path string, query url.Values, contentType string, body []byte,
) ([]byte, error) {
	cfg := c.store.Get()
	if strings.TrimSpace(base) == "" {
		return nil, &APIError{Message: "服务地址未配置", Path: path}
	}
	endpoint := strings.TrimRight(base, "/") + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout())
	defer cancel()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(reqCtx, method, endpoint, reader)
	if err != nil {
		return nil, &APIError{Message: err.Error(), Path: path}
	}
	req.Header.Set("X-Correlation-ID", correlationID())
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &APIError{Message: "无法连接服务：" + err.Error(), Path: path}
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if readErr != nil {
		return nil, &APIError{Message: "读取响应失败", Path: path}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			Status:  resp.StatusCode,
			Path:    path,
			Message: errorMessage(data, resp.StatusCode),
		}
	}
	return data, nil
}

func (c *Client) requestJSON(
	ctx context.Context, base, method, path string, query url.Values, body any,
) ([]byte, error) {
	var payload []byte
	contentType := ""
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, &APIError{Message: "请求体序列化失败", Path: path}
		}
		payload = encoded
		contentType = "application/json"
	}
	return c.roundTrip(ctx, method, base, path, query, contentType, payload)
}

func (c *Client) getJSON(ctx context.Context, base, path string, query url.Values, out any) error {
	data, err := c.roundTrip(ctx, http.MethodGet, base, path, query, "", nil)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func (c *Client) sendJSON(ctx context.Context, base, method, path string, body, out any) error {
	data, err := c.requestJSON(ctx, base, method, path, nil, body)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func asMap(data []byte) map[string]any {
	if len(data) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func asList(data []byte) []any {
	if len(data) == 0 {
		return []any{}
	}
	var out []any
	if err := json.Unmarshal(data, &out); err != nil {
		return []any{}
	}
	return out
}

func decodeAny(data []byte) any {
	if len(data) == 0 {
		return map[string]any{}
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return string(data)
	}
	return out
}

func pageQuery(offset, limit int) url.Values {
	query := url.Values{}
	query.Set("offset", strconv.Itoa(offset))
	query.Set("limit", strconv.Itoa(limit))
	return query
}

func intParam(values map[string]any, key string, fallback int) int {
	raw, ok := values[key]
	if !ok {
		return fallback
	}
	switch value := raw.(type) {
	case float64:
		return int(value)
	case string:
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}
