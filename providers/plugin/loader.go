// SPDX-License-Identifier: LGPL-3.0-or-later

package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"hypersdk/logger"
	"hypersdk/providers"
)

// Loader loads provider plugins from shared libraries
type Loader struct {
	logger logger.Logger
}

// NewLoader creates a new plugin loader
func NewLoader(log logger.Logger) *Loader {
	return &Loader{
		logger: log,
	}
}

// Load loads a plugin from a file
func (l *Loader) Load(path string) (*Info, providers.ProviderFactory, error) {
	l.logger.Info("loading plugin", "path", path)

	// Validate file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("plugin file not found: %s", path)
	}

	// Load plugin
	p, err := plugin.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Lookup metadata
	metadataSym, err := p.Lookup("PluginInfo")
	if err != nil {
		return nil, nil, fmt.Errorf("plugin missing PluginInfo: %w", err)
	}

	metadata, ok := metadataSym.(*Metadata)
	if !ok {
		return nil, nil, fmt.Errorf("PluginInfo has wrong type: %T", metadataSym)
	}

	// Validate metadata
	if err := metadata.ValidateMetadata(); err != nil {
		return nil, nil, fmt.Errorf("invalid plugin metadata: %w", err)
	}

	// Lookup provider factory
	factorySym, err := p.Lookup("NewProvider")
	if err != nil {
		return nil, nil, fmt.Errorf("plugin missing NewProvider: %w", err)
	}

	// Validate factory type
	pluginFactory, ok := factorySym.(func(providers.ProviderConfig, logger.Logger) (providers.Provider, error))
	if !ok {
		return nil, nil, fmt.Errorf("NewProvider has wrong type: %T", factorySym)
	}

	// Create plugin info
	info := &Info{
		Metadata: *metadata,
		Path:     path,
		Status:   StatusLoaded,
	}

	l.logger.Info("plugin loaded successfully",
		"name", metadata.Name,
		"version", metadata.Version,
		"type", metadata.ProviderType)

	// Wrap plugin factory to match ProviderFactory signature
	factory := func(config providers.ProviderConfig) (providers.Provider, error) {
		return pluginFactory(config, l.logger)
	}

	return info, factory, nil
}

// Discover scans a directory for plugin files
func (l *Loader) Discover(dir string) ([]string, error) {
	l.logger.Debug("discovering plugins in directory", "dir", dir)

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		l.logger.Debug("plugin directory does not exist", "dir", dir)
		return nil, nil
	}

	var pluginPaths []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check for .so extension
		if strings.HasSuffix(path, ".so") {
			pluginPaths = append(pluginPaths, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	l.logger.Debug("discovered plugins", "count", len(pluginPaths), "dir", dir)

	return pluginPaths, nil
}

// DiscoverAll scans multiple directories for plugins
func (l *Loader) DiscoverAll(dirs []string) ([]string, error) {
	var allPaths []string

	for _, dir := range dirs {
		paths, err := l.Discover(dir)
		if err != nil {
			l.logger.Warn("failed to discover plugins in directory",
				"dir", dir,
				"error", err)
			continue
		}
		allPaths = append(allPaths, paths...)
	}

	return allPaths, nil
}

// GetDefaultPluginDirs returns default plugin directories
func GetDefaultPluginDirs() []string {
	dirs := []string{
		"/usr/local/lib/hypersdk/plugins",
		"/usr/lib/hypersdk/plugins",
	}

	// Add user-specific directory
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".hypersdk", "plugins"))
	}

	// Add current directory
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, "plugins"))
	}

	return dirs
}

// ParsePluginPath parses HYPERSDK_PLUGIN_PATH environment variable
func ParsePluginPath() []string {
	pluginPath := os.Getenv("HYPERSDK_PLUGIN_PATH")
	if pluginPath == "" {
		return nil
	}

	// Split by : (Unix) or ; (Windows)
	separator := ":"
	if os.PathSeparator == '\\' {
		separator = ";"
	}

	return strings.Split(pluginPath, separator)
}
