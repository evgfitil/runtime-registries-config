package runtimeconfig

import (
	"github.com/caarlos0/env/v10"
	"github.com/evgfitil/runtime-registries-config/internal/client"
	"github.com/evgfitil/runtime-registries-config/internal/logger"
	"github.com/pelletier/go-toml"
	"sort"
)

type RegistryMirror struct {
	Prefix   string `toml:"prefix"`
	Location string `toml:"location"`
	Insecure bool   `toml:"insecure"`
}

const (
	crioServiceName = "crio.service"
)

type CrioConfig struct {
	RegistryMirrors       []RegistryMirror `toml:"registry"`
	RuntimeConfigFileName string           `env:"NODE_CONFIG_NAME" envDefault:"99-registries.conf"`
	RuntimeConfigFilePath string           `env:"NODE_CONFIG_PATH" envDefault:"/etc/crio/crio.conf.d"`
}

func NewCrioConfig() *CrioConfig {
	crioConfig := &CrioConfig{}
	if err := env.Parse(crioConfig); err != nil {
		logger.Sugar.Fatalf("error to parse environment variables: %v", err)
	}
	return crioConfig
}

func (c *CrioConfig) BuildConfig(data []client.ConfigMapData) error {
	c.RegistryMirrors = []RegistryMirror{}
	for _, item := range data {
		c.RegistryMirrors = append(c.RegistryMirrors, RegistryMirror{
			Prefix:   item.Original,
			Location: item.Mirror,
			Insecure: item.Insecure,
		})
	}
	return nil
}

func (c *CrioConfig) DeserializeConfig(data []byte) error {
	if err := toml.Unmarshal(data, c); err != nil {
		logger.Sugar.Errorf("error unmarshaling crio config: %v", err)
		return err
	}
	return nil
}

func (c *CrioConfig) GetServiceName() string {
	return crioServiceName
}

func (c *CrioConfig) GetConfigFilePath() string {
	return c.RuntimeConfigFilePath
}

func (c *CrioConfig) GetConfigFileName() string {
	return c.RuntimeConfigFileName
}

func (c *CrioConfig) IsEqual(incoming RuntimeConfig) (bool, error) {
	incomingConfig, ok := incoming.(*CrioConfig)
	if !ok {
		logger.Sugar.Error("cannot compare, something wrong with incoming config")
		return false, nil
	}
	if len(c.RegistryMirrors) != len(incomingConfig.RegistryMirrors) {
		return false, nil
	}
	currentConfigMirrors := c.RegistryMirrors
	incomingConfigMirrors := incomingConfig.RegistryMirrors
	sortRegistryMirrors(currentConfigMirrors)
	sortRegistryMirrors(incomingConfigMirrors)

	for i := 0; i < len(currentConfigMirrors); i++ {
		if currentConfigMirrors[i] != incomingConfigMirrors[i] {
			return false, nil
		}
	}
	return true, nil
}

func (c *CrioConfig) LoadConfig(data []byte) error {
	if err := toml.Unmarshal(data, c); err != nil {
		logger.Sugar.Errorf("error unmarshaling crio config: %v", err)
		return err
	}
	return nil
}

func (c *CrioConfig) ResetConfig() {
	c.RegistryMirrors = []RegistryMirror{}
}

func (c *CrioConfig) SerializeConfig() ([]byte, error) {
	wrapper := struct {
		RegistryMirrors []RegistryMirror `toml:"registry"`
	}{
		RegistryMirrors: c.RegistryMirrors,
	}
	data, err := toml.Marshal(wrapper)
	if err != nil {
		logger.Sugar.Errorf("error marshaling: %v", err)
		return nil, err
	}

	return data, nil
}

func sortRegistryMirrors(mirrors []RegistryMirror) {
	sort.Slice(mirrors, func(i, j int) bool {
		return mirrors[i].Prefix < mirrors[j].Prefix
	})
}
