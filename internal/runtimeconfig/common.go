package runtimeconfig

import (
	"fmt"
	"github.com/evgfitil/runtime-registries-config/internal/client"
)

type RuntimeConfig interface {
	BuildConfig(data []client.ConfigMapData) error
	GetConfigFileName() string
	GetConfigFilePath() string
	GetServiceName() string
	DeserializeConfig([]byte) error
	IsEqual(config RuntimeConfig) (bool, error)
	LoadConfig(data []byte) error
	SerializeConfig() ([]byte, error)
	ResetConfig()
}

func NewRuntimeConfig(runtimeType string) (RuntimeConfig, error) {
	switch runtimeType {
	case "cri-o":
		return NewCrioConfig(), nil
	default:
		return nil, fmt.Errorf("unsupported runtime type: %s", runtimeType)
	}
}
