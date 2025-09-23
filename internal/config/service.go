package config

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type AsenaConfigService struct {
	cfg            *AsenaConfig
	logg           *zap.Logger
	configFilePath string
}

func NewAsenaConfigService(configFilePath string, logg *zap.Logger) (*AsenaConfigService, error) {
	acs := &AsenaConfigService{
		configFilePath: configFilePath,
		logg:           logg,
	}

	if err := acs.load(); err != nil {
		return nil, err
	}

	return acs, nil
}

func (acs *AsenaConfigService) load() error {
	data, err := os.ReadFile(acs.configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read asena config file: %s: %w", acs.configFilePath, err)
	}

	var cfg AsenaConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse asena config file: %s: %w", acs.configFilePath, err)
	}

	//	Set and normalize configurations
	setAsenaConfigs(&cfg)

	acs.cfg = &cfg

	return nil
}

func (acs *AsenaConfigService) Get() *AsenaConfig {
	return acs.cfg
}
