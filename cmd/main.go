package main

import (
	"fmt"
	"github.com/caarlos0/env/v10"
	"github.com/evgfitil/runtime-registries-config/internal/client"
	"github.com/evgfitil/runtime-registries-config/internal/handlers"
	"github.com/evgfitil/runtime-registries-config/internal/logger"
	"github.com/evgfitil/runtime-registries-config/internal/nodeconfig"
	"github.com/evgfitil/runtime-registries-config/internal/runtimeconfig"
)

func main() {
	cfg := NewConfig()
	if err := env.Parse(cfg); err != nil {
		fmt.Errorf("error parse environment variables: %v", err)
	}

	logger.InitLogger(cfg.LogLevel)
	defer logger.Sugar.Sync()

	runtimeCfg, err := runtimeconfig.NewRuntimeConfig(cfg.CRI)
	if err != nil {
		logger.Sugar.Fatalf("unsupported runtime type: %v", err)
	}

	nodeCfg := nodeconfig.NewNodeConfig(runtimeCfg)
	if err := nodeCfg.LoadNodeRuntimeConfig(); err != nil {
		logger.Sugar.Errorf("Failed to load node runtime config: %v", err)
		return
	}

	watcher, err := client.NewConfigWatcher()
	if err != nil {
		logger.Sugar.Fatalf("cannot create new watcher: %v", err)
		panic(err.Error())
	}

	handlers := handlers.NewConfigMapHandler(nodeCfg, cfg.CRI)
	logger.Sugar.Infoln("starting the Informer")
	watcher.TrackConfigChanges(handlers)
}
