package handlers

import (
	"github.com/evgfitil/runtime-registries-config/internal/client"
	"github.com/evgfitil/runtime-registries-config/internal/logger"
	"github.com/evgfitil/runtime-registries-config/internal/nodeconfig"
	"github.com/evgfitil/runtime-registries-config/internal/runtimeconfig"
)

type ConfigMapHandler struct {
	NodeConfig  *nodeconfig.NodeConfig
	runtimeType string
}

func NewConfigMapHandler(nc *nodeconfig.NodeConfig, runtimeType string) *ConfigMapHandler {
	return &ConfigMapHandler{
		NodeConfig:  nc,
		runtimeType: runtimeType,
	}
}

func (ch *ConfigMapHandler) AddConfigMap(newData *[]client.ConfigMapData) {
	tempConfig, err := runtimeconfig.NewRuntimeConfig(ch.runtimeType)

	if err != nil {
		logger.Sugar.Errorf("failed to initialize temp config: %v", err)
	}

	err = tempConfig.BuildConfig(*newData)
	if err != nil {
		logger.Sugar.Errorf("error building config: %v", err)
	}

	isEqual, err := ch.NodeConfig.Config.IsEqual(tempConfig)
	if err != nil {
		logger.Sugar.Errorf("error comparing configs: %v", err)
	}

	if !isEqual {
		err = ch.NodeConfig.UpdateNodeRuntimeConfig(tempConfig)
		if err != nil {
			logger.Sugar.Errorf("failed to update node config with new data: %v", err)
		}
		logger.Sugar.Infoln("new config has been applied")
	} else {
		logger.Sugar.Infoln("no updates required")
	}
}

func (ch *ConfigMapHandler) UpdateConfigMap(newData *[]client.ConfigMapData) {
	tempConfig, err := runtimeconfig.NewRuntimeConfig(ch.runtimeType)
	if err != nil {
		logger.Sugar.Errorf("failed to initialize temp config: %v", err)
	}

	err = tempConfig.BuildConfig(*newData)
	if err != nil {
		logger.Sugar.Errorf("error deserialize config: %v", err)
	}

	isEqual, err := ch.NodeConfig.Config.IsEqual(tempConfig)
	if err != nil {
		logger.Sugar.Errorf("error comparing configs: %v", err)
	}
	if !isEqual {
		err = ch.NodeConfig.UpdateNodeRuntimeConfig(tempConfig)
		if err != nil {
			logger.Sugar.Errorf("failed to update node config with new data: %v", err)
		}
		logger.Sugar.Infoln("new config has been applied")
	} else {
		logger.Sugar.Infoln("no updates required")
	}
}

func (ch *ConfigMapHandler) DeleteConfigMap(obj interface{}) {
	logger.Sugar.Infoln("trigger delete handler")
	if err := ch.NodeConfig.TruncateRuntimeConfigFile(); err != nil {
		logger.Sugar.Errorf("error deleting config: %v", err)
	} else {
		logger.Sugar.Infoln("config reset successfully")
	}
}
