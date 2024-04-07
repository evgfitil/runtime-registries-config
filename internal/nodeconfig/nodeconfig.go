package nodeconfig

import (
	"context"
	"fmt"
	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/evgfitil/runtime-registries-config/internal/logger"
	"github.com/evgfitil/runtime-registries-config/internal/runtimeconfig"
	"os"
	"path"
	"time"
)

type NodeConfig struct {
	Config runtimeconfig.RuntimeConfig
}

const (
	defaultDirectoryPermissions  = 0755
	defaultFilePermissions       = 0644
	defaultFilePrefix            = `# This configuration is managed by runtime-registries-config`
	defaultServiceRestartTimeout = 2
)

func NewNodeConfig(config runtimeconfig.RuntimeConfig) *NodeConfig {
	return &NodeConfig{
		Config: config,
	}
}

func (nc *NodeConfig) LoadNodeRuntimeConfig() error {
	if _, err := os.Stat(nc.Config.GetConfigFilePath()); os.IsNotExist(err) {
		err = createRuntimeConfigDirectory(nc.Config.GetConfigFilePath())
		if err != nil {
			logger.Sugar.Fatalf("error: %v", err)
		}
	}

	fullFilePath := path.Join(nc.Config.GetConfigFilePath(), nc.Config.GetConfigFileName())
	if _, err := os.Stat(fullFilePath); os.IsNotExist(err) {
		err = createNodeRuntimeConfig(fullFilePath)
		if err != nil {
			logger.Sugar.Fatalf("error: %v", err)
		}
	}

	data, err := os.ReadFile(fullFilePath)
	if err != nil {
		return err
	}

	if err = nc.Config.DeserializeConfig(data); err != nil {
		logger.Sugar.Errorf("error deserializing config data: %v", err)
		return err
	}

	return nil
}

func (nc *NodeConfig) TruncateRuntimeConfigFile() error {
	fullFilePath := path.Join(nc.Config.GetConfigFilePath(), nc.Config.GetConfigFileName())
	if err := os.Truncate(fullFilePath, 0); err != nil {
		logger.Sugar.Errorf("error truncate runtime config file: %v", err)
		return err
	}
	prefix := []byte(defaultFilePrefix + "\n")
	err := os.WriteFile(fullFilePath, prefix, 0644)
	if err != nil {
		logger.Sugar.Errorf("error writing prefix data: %v", err)
		return err
	}
	if err = nc.reloadOrRestartRuntimeService(); err != nil {
		logger.Sugar.Errorf("error restarting runtime: %v", err)
	}
	nc.Config.ResetConfig()
	return nil
}

func (nc *NodeConfig) UpdateNodeRuntimeConfig(incomingConfig runtimeconfig.RuntimeConfig) error {
	data, err := incomingConfig.SerializeConfig()
	if err != nil {
		logger.Sugar.Errorf("error updating node config: %v", err)
		return err
	}

	fullFilePath := path.Join(nc.Config.GetConfigFilePath(), nc.Config.GetConfigFileName())
	prefix := []byte(defaultFilePrefix + "\n")
	newDataWithPrefix := append(prefix, data...)
	err = os.WriteFile(fullFilePath, newDataWithPrefix, defaultFilePermissions)
	if err != nil {
		logger.Sugar.Errorf("error writing config: %v", err)
		return err
	}
	if err = nc.reloadOrRestartRuntimeService(); err != nil {
		logger.Sugar.Errorf("error apply new config: %v", err)
	}
	return nil
}

func createNodeRuntimeConfig(filePath string) error {
	err := os.WriteFile(filePath, []byte(defaultFilePrefix), defaultFilePermissions)
	if err != nil {
		logger.Sugar.Fatalf("error creating config file: %v", err)
		return err
	}
	return nil
}

func createRuntimeConfigDirectory(configFilePath string) error {
	err := os.MkdirAll(configFilePath, defaultDirectoryPermissions)
	if err != nil {
		logger.Sugar.Fatalf("error creating config directory: %v", err)
		return err
	}
	return nil
}

func (nc *NodeConfig) reloadOrRestartRuntimeService() error {
	conn, err := dbus.NewSystemdConnectionContext(context.TODO())
	if err != nil {
		logger.Sugar.Errorf("failed to connect to systemd: %v", err)
		return err
	}
	defer conn.Close()

	serviceName := nc.Config.GetServiceName()
	resultChannel := make(chan string)
	defer close(resultChannel)

	if _, err = conn.ReloadOrRestartUnitContext(context.TODO(), serviceName, "replace", resultChannel); err != nil {
		logger.Sugar.Errorf("failed to restart %s service: %s", serviceName, err)
		return err
	}

	select {
	case result := <-resultChannel:
		logger.Sugar.Infof("service %s restart result: %s", serviceName, result)
	case <-time.After(defaultServiceRestartTimeout * time.Second):
		logger.Sugar.Errorf("timeout waiting for %s to restart", serviceName)
		return fmt.Errorf("timeout waiting for service %s to restart", serviceName)
	}
	return nil
}
