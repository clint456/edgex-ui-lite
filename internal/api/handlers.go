package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/edge-hy/edge-hy-console/internal/config"
	"github.com/edge-hy/edge-hy-console/internal/edgex"
)

func atoiDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func (s *server) fail(c *gin.Context, err error) {
	status := http.StatusBadGateway
	var apiErr *edgex.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Status {
		case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict,
			http.StatusUnauthorized, http.StatusForbidden, http.StatusServiceUnavailable:
			status = apiErr.Status
		}
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

// --- configuration -------------------------------------------------------

func (s *server) getConfig(c *gin.Context) {
	cfg := s.store.Get()
	c.JSON(http.StatusOK, gin.H{
		"metadataUrl": cfg.MetadataURL,
		"dataUrl":     cfg.DataURL,
		"commandUrl":  cfg.CommandURL,
		"mappingsUrl": cfg.MappingsURL,
		"keeperUrl":   cfg.KeeperURL,
		"systemUrl":   cfg.SystemURL,
		"timeoutSec":  cfg.TimeoutSec,
		"tokenSet":    cfg.Token != "",
	})
}

type configRequest struct {
	MetadataURL string   `json:"metadataUrl"`
	DataURL     string   `json:"dataUrl"`
	CommandURL  string   `json:"commandUrl"`
	MappingsURL string   `json:"mappingsUrl"`
	KeeperURL   string   `json:"keeperUrl"`
	SystemURL   string   `json:"systemUrl"`
	Token       *string  `json:"token"`
	TimeoutSec  *float64 `json:"timeoutSec"`
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func (s *server) putConfig(c *gin.Context) {
	var req configRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体必须是 JSON 对象"})
		return
	}
	current := s.store.Get()
	next := config.Config{
		MetadataURL: orDefault(req.MetadataURL, current.MetadataURL),
		DataURL:     orDefault(req.DataURL, current.DataURL),
		CommandURL:  orDefault(req.CommandURL, current.CommandURL),
		MappingsURL: orDefault(req.MappingsURL, current.MappingsURL),
		KeeperURL:   orDefault(req.KeeperURL, current.KeeperURL),
		SystemURL:   orDefault(req.SystemURL, current.SystemURL),
		Token:       current.Token,
		TimeoutSec:  current.TimeoutSec,
	}
	if req.Token != nil {
		next.Token = strings.TrimSpace(*req.Token)
	}
	if req.TimeoutSec != nil && *req.TimeoutSec > 0 {
		next.TimeoutSec = *req.TimeoutSec
	}
	s.store.Set(next)
	s.getConfig(c)
}

// --- system --------------------------------------------------------------

func (s *server) getHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"services": s.client.Health(c.Request.Context())})
}

func (s *server) getServices(c *gin.Context) {
	registrations, err := s.client.Registrations(c.Request.Context())
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": registrations})
}

var overviewLabels = map[string]string{
	"device":  "设备",
	"service": "设备服务",
	"profile": "设备模型",
	"watcher": "自动发现",
	"event":   "事件",
	"reading": "读数",
}

func (s *server) getOverview(c *gin.Context) {
	ctx := c.Request.Context()

	type job struct {
		key   string
		run   func() (int, error)
	}
	jobs := make([]job, 0, 6)
	for _, key := range []string{"device", "service", "profile", "watcher"} {
		kind, _ := edgex.LookupKind(key)
		captured := kind
		jobs = append(jobs, job{key, func() (int, error) { return s.client.CountMetadata(ctx, captured) }})
	}
	jobs = append(jobs,
		job{"event", func() (int, error) { return s.client.CountData(ctx, "event") }},
		job{"reading", func() (int, error) { return s.client.CountData(ctx, "reading") }},
	)

	rows := make([]gin.H, len(jobs))
	var wg sync.WaitGroup
	for index, item := range jobs {
		wg.Add(1)
		go func(index int, item job) {
			defer wg.Done()
			total, err := item.run()
			row := gin.H{"key": item.key, "label": overviewLabels[item.key], "total": total}
			if err != nil {
				row["error"] = err.Error()
			}
			rows[index] = row
		}(index, item)
	}
	wg.Wait()

	c.JSON(http.StatusOK, gin.H{
		"counts": rows,
		"health": s.client.Health(ctx),
	})
}

func (s *server) systemMetrics(c *gin.Context) {
	value, err := s.client.SystemMetrics(c.Request.Context(), c.Query("services"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": value})
}

func (s *server) systemConfig(c *gin.Context) {
	value, err := s.client.SystemConfig(c.Request.Context(), c.Query("services"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": value})
}

func (s *server) systemOperation(c *gin.Context) {
	var req struct {
		ServiceName string `json:"serviceName"`
		Action      string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体必须是 JSON 对象"})
		return
	}
	if req.Action != "start" && req.Action != "stop" && req.Action != "restart" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action 必须是 start/stop/restart"})
		return
	}
	if strings.TrimSpace(req.ServiceName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "serviceName 不能为空"})
		return
	}
	value, err := s.client.SystemOperation(c.Request.Context(), req.ServiceName, req.Action)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": value})
}

// --- metadata ------------------------------------------------------------

func (s *server) kindFrom(c *gin.Context) (edgex.MetaKind, bool) {
	kind, ok := edgex.LookupKind(c.Param("kind"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "未知资源类型"})
	}
	return kind, ok
}

func (s *server) listMetadata(c *gin.Context) {
	kind, ok := s.kindFrom(c)
	if !ok {
		return
	}
	result, err := s.client.ListMetadata(
		c.Request.Context(), kind, c.Query("filter"), c.Query("value"),
		atoiDefault(c.Query("offset"), 0), atoiDefault(c.Query("limit"), 20),
	)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *server) getMetadata(c *gin.Context) {
	kind, ok := s.kindFrom(c)
	if !ok {
		return
	}
	record, err := s.client.GetMetadata(c.Request.Context(), kind, c.Param("name"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func (s *server) createMetadata(c *gin.Context) {
	kind, ok := s.kindFrom(c)
	if !ok {
		return
	}
	record, ok := bindRecord(c)
	if !ok {
		return
	}
	if err := validateRecord(kind.Key, record, true); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.client.SaveMetadata(c.Request.Context(), kind, record, true); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *server) updateMetadata(c *gin.Context) {
	kind, ok := s.kindFrom(c)
	if !ok {
		return
	}
	record, ok := bindRecord(c)
	if !ok {
		return
	}
	if err := validateRecord(kind.Key, record, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.client.SaveMetadata(c.Request.Context(), kind, record, false); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *server) deleteMetadata(c *gin.Context) {
	kind, ok := s.kindFrom(c)
	if !ok {
		return
	}
	if err := s.client.DeleteMetadata(c.Request.Context(), kind, c.Param("name")); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func bindRecord(c *gin.Context) (map[string]any, bool) {
	var record map[string]any
	if err := c.ShouldBindJSON(&record); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体必须是 JSON 对象"})
		return nil, false
	}
	return record, true
}

// --- device commands -----------------------------------------------------

func (s *server) deviceCommands(c *gin.Context) {
	commands, err := s.client.DeviceCommands(c.Request.Context(), c.Param("name"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, commands)
}

func (s *server) executeCommandGet(c *gin.Context) {
	value, err := s.client.ExecuteCommand(
		c.Request.Context(), c.Param("name"), c.Param("command"), http.MethodGet, nil,
	)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (s *server) executeCommandPut(c *gin.Context) {
	body, ok := bindRecord(c)
	if !ok {
		return
	}
	value, err := s.client.ExecuteCommand(
		c.Request.Context(), c.Param("name"), c.Param("command"), http.MethodPut, body,
	)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

// --- data center ---------------------------------------------------------

func (s *server) listData(c *gin.Context) {
	kind := c.Param("kind")
	if !edgex.DataKindValid(kind) {
		c.JSON(http.StatusNotFound, gin.H{"error": "未知数据类型"})
		return
	}
	result, err := s.client.ListData(
		c.Request.Context(), kind, c.Query("device"),
		atoiDefault(c.Query("offset"), 0), atoiDefault(c.Query("limit"), 20),
	)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *server) countData(c *gin.Context) {
	kind := c.Param("kind")
	if !edgex.DataKindValid(kind) {
		c.JSON(http.StatusNotFound, gin.H{"error": "未知数据类型"})
		return
	}
	total, err := s.client.CountData(c.Request.Context(), kind)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": total})
}

// --- mappings ------------------------------------------------------------

func (s *server) listMappings(c *gin.Context) {
	filter := c.Query("filter")
	name := c.Query("name")
	if filter != "" && filter != "all" && strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写需要精确筛选的设备或模型名称"})
		return
	}
	result, err := s.client.ListMappings(
		c.Request.Context(), filter, name,
		atoiDefault(c.Query("offset"), 0), atoiDefault(c.Query("limit"), 20),
	)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *server) saveMapping(c *gin.Context, creating bool) {
	body, ok := bindRecord(c)
	if !ok {
		return
	}
	result, err := s.client.SaveMapping(c.Request.Context(), body, creating)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "mapping": result})
}

func (s *server) createMapping(c *gin.Context) { s.saveMapping(c, true) }
func (s *server) updateMapping(c *gin.Context) { s.saveMapping(c, false) }

func (s *server) deleteMapping(c *gin.Context) {
	if err := s.client.DeleteMapping(c.Request.Context(), c.Param("id")); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *server) lookupMapping(c *gin.Context) {
	point := map[string]string{
		"device":   c.Query("device"),
		"profile":  c.Query("profile"),
		"resource": c.Query("resource"),
	}
	result, err := s.client.LookupMapping(c.Request.Context(), c.DefaultQuery("direction", "forward"), point)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"mapping": result})
}

func (s *server) batchMappings(c *gin.Context) {
	var req struct {
		Direction string              `json:"direction"`
		Points    []map[string]string `json:"points"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体必须是 JSON 对象"})
		return
	}
	if req.Direction != "forward" && req.Direction != "reverse" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "direction 必须是 forward/reverse"})
		return
	}
	if len(req.Points) == 0 || len(req.Points) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "points 数量必须在 1–1000 之间"})
		return
	}
	result, err := s.client.BatchMappings(c.Request.Context(), req.Direction, req.Points)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": result})
}

func (s *server) deleteMappingScope(c *gin.Context) {
	err := s.client.DeleteMappingScope(
		c.Request.Context(), c.Param("side"), c.Param("kind"), c.Param("name"),
	)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- device profile YAML -------------------------------------------------

func (s *server) profileYAML(c *gin.Context) {
	kind, _ := edgex.LookupKind("profile")
	record, err := s.client.GetMetadata(c.Request.Context(), kind, c.Param("name"))
	if err != nil {
		s.fail(c, err)
		return
	}
	data, err := yaml.Marshal(record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "YAML 生成失败"})
		return
	}
	c.Data(http.StatusOK, "application/x-yaml; charset=utf-8", data)
}

func (s *server) uploadProfile(c *gin.Context) {
	var req struct {
		Name   string `json:"name"`
		Yaml   string `json:"yaml"`
		Update bool   `json:"update"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体必须是 JSON 对象"})
		return
	}
	if strings.TrimSpace(req.Yaml) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "YAML 内容不能为空"})
		return
	}
	filename := "device-profile.yaml"
	if strings.TrimSpace(req.Name) != "" {
		filename = strings.TrimSpace(req.Name) + ".yaml"
	}
	if err := s.client.UploadProfile(c.Request.Context(), filename, req.Yaml, req.Update); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
