package config

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ============================== Static ==============================

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
	err = setAsenaConfigs(&cfg, acs.configFilePath)
	if err != nil {
		return err
	}

	acs.cfg = &cfg

	return nil
}

func (acs *AsenaConfigService) Get() *AsenaConfig {
	return acs.cfg
}

// ============================== Dynamic ==============================

type DynamicConfigService struct {
	cfg            *DynamicConfig
	configFilePath string
	logg           *zap.Logger
	updates        chan *DynamicConfig
	mu             sync.RWMutex
	hash           []byte
}

func NewDynamicConfigService(ctx context.Context, configFilePath string, logg *zap.Logger) (*DynamicConfigService, error) {
	dcs := &DynamicConfigService{
		configFilePath: configFilePath,
		logg:           logg,
		updates:        make(chan *DynamicConfig, 1),
	}

	if err := dcs.reload(); err != nil {
		return nil, err
	}

	go dcs.watch(ctx)

	return dcs, nil
}

func (dcs *DynamicConfigService) reload() error {
	data, err := os.ReadFile(dcs.configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read dynamic config file: %s: %w", dcs.configFilePath, err)
	}

	var cfg DynamicConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse dynamic config file: %s: %w", dcs.configFilePath, err)
	}

	//	set/normalize/validate configurations
	if err := setDynamicConfigs(&cfg); err != nil {
		return err
	}

	newHash := sha256sum(data)
	if dcs.hash != nil && bytes.Equal(dcs.hash, newHash) {
		return nil
	}

	dcs.mu.Lock()
	dcs.cfg = &cfg
	dcs.hash = newHash
	dcs.mu.Unlock()

	select {
	case dcs.updates <- &cfg:
	default:
	}

	return nil
}

func (dcs *DynamicConfigService) watch(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		dcs.logg.Error("failed to create watcher for dynamic config", zap.Error(err))
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			dcs.logg.Error("failed to close watcher for dynamic config", zap.Error(err))
		}
	}()

	if err := watcher.Add(dcs.configFilePath); err != nil {
		dcs.logg.Error("failed to start watcher for dynamic config", zap.Error(err))
		return
	}

	var debounceMu sync.Mutex
	var debounceTimer *time.Timer

	stopTimer := func() {
		debounceMu.Lock()
		if debounceTimer != nil {
			if debounceTimer.Stop() {
				// We are using time.AfterFunc, which runs a callback directly instead of sending on a channel.
				// Unlike time.NewTimer, there is no channel to drain after Stop() — calling Stop() is enough.
			}
			debounceTimer = nil
		}
		debounceMu.Unlock()
	}

	for {
		select {
		case <-ctx.Done():
			stopTimer()
			return
		case event, ok := <-watcher.Events:
			if !ok {
				stopTimer()
				return
			}

			if filepath.Clean(event.Name) != filepath.Clean(dcs.configFilePath) {
				continue
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Chmod) != 0 {
				debounceMu.Lock()
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
					if err := dcs.reload(); err != nil {
						dcs.logg.Error("failed to read dynamic config file", zap.Error(err))
					}
				})
				debounceMu.Unlock()
			}
		case err := <-watcher.Errors:
			dcs.logg.Error("failed to watch dynamic config file", zap.Error(err))
		}
	}
}

func (dcs *DynamicConfigService) Get() *DynamicConfig {
	dcs.mu.RLock()
	defer dcs.mu.RUnlock()
	return dcs.cfg
}

func (dcs *DynamicConfigService) Updates() <-chan *DynamicConfig {
	return dcs.updates
}

func sha256sum(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}
