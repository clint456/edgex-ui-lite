// Command edge-hy-console serves the EdgeX management console.
package main

import (
	"log"
	"os"

	"github.com/edge-hy/edge-hy-console/internal/api"
	"github.com/edge-hy/edge-hy-console/internal/config"
	"github.com/edge-hy/edge-hy-console/internal/edgex"
)

func main() {
	store := config.NewStore(config.Load())
	client := edgex.NewClient(store)

	router, err := api.NewRouter(store, client)
	if err != nil {
		log.Fatalf("初始化路由失败: %v", err)
	}

	addr := os.Getenv("EDGEX_LISTEN")
	if addr == "" {
		addr = ":8080"
	}
	cfg := store.Get()
	log.Printf("edge-hy-console 启动于 %s（metadata=%s data=%s command=%s mappings=%s）",
		addr, cfg.MetadataURL, cfg.DataURL, cfg.CommandURL, cfg.MappingsURL)
	if err := router.Run(addr); err != nil {
		log.Fatalf("服务退出: %v", err)
	}
}
