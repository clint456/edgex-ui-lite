// Package api exposes the JSON API and the embedded frontend over Gin.
package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/edge-hy/edge-hy-console/internal/config"
	"github.com/edge-hy/edge-hy-console/internal/edgex"
	"github.com/edge-hy/edge-hy-console/internal/web"
)

type server struct {
	store  *config.Store
	client *edgex.Client
}

// NewRouter builds the Gin engine: JSON API under /api plus the static UI.
func NewRouter(store *config.Store, client *edgex.Client) (*gin.Engine, error) {
	static, err := web.Static()
	if err != nil {
		return nil, err
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	s := &server{store: store, client: client}

	api := router.Group("/api")
	{
		api.GET("/config", s.getConfig)
		api.PUT("/config", s.putConfig)

		api.GET("/health", s.getHealth)
		api.GET("/services", s.getServices)
		api.GET("/overview", s.getOverview)

		api.GET("/metadata/:kind", s.listMetadata)
		api.POST("/metadata/:kind", s.createMetadata)
		api.PUT("/metadata/:kind", s.updateMetadata)
		api.GET("/metadata/:kind/:name", s.getMetadata)
		api.DELETE("/metadata/:kind/:name", s.deleteMetadata)

		api.GET("/devices/:name/commands", s.deviceCommands)
		api.GET("/devices/:name/commands/:command", s.executeCommandGet)
		api.PUT("/devices/:name/commands/:command", s.executeCommandPut)

		api.GET("/data/:kind", s.listData)
		api.GET("/data/:kind/count", s.countData)

		api.GET("/mappings", s.listMappings)
		api.POST("/mappings", s.createMapping)
		api.PUT("/mappings", s.updateMapping)
		api.GET("/mappings/lookup", s.lookupMapping)
		api.POST("/mappings/batch", s.batchMappings)
		api.DELETE("/mappings/:id", s.deleteMapping)
		api.DELETE("/mappings/scope/:side/:kind/:name", s.deleteMappingScope)

		api.GET("/system/metrics", s.systemMetrics)
		api.GET("/system/config", s.systemConfig)
		api.POST("/system/operation", s.systemOperation)

		api.GET("/profile/:name/yaml", s.profileYAML)
		api.POST("/profile/upload", s.uploadProfile)
	}

	serveStatic(router, static)
	return router, nil
}

// serveStatic serves embedded assets and falls back to index.html for the SPA.
func serveStatic(router *gin.Engine, static fs.FS) {
	fileServer := http.FileServer(http.FS(static))
	router.NoRoute(func(c *gin.Context) {
		requested := strings.TrimPrefix(c.Request.URL.Path, "/")
		if requested != "" {
			if info, err := fs.Stat(static, requested); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		index, err := fs.ReadFile(static, "index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})
}
