// Package viper defines a config parser implementation based on the spf13/viper pkg
package viper

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Parser wraps a viper.Viper instance to isolate parsing logic from
// application-specific types and behaviours.
type Parser struct {
	v       *viper.Viper
	mu      sync.RWMutex
	watches map[string]func()
}

// Config represents a parsed configuration
type Config struct {
	// Raw contains the unmarshaled configuration as a map
	Raw map[string]interface{}
	// Viper provides direct access to the underlying viper instance
	// for advanced use cases
	Viper *viper.Viper
}

// Option defines a function that can modify a Parser
type Option func(*Parser)

// WithEnvPrefix sets the prefix for environment variables
func WithEnvPrefix(prefix string) Option {
	return func(p *Parser) {
		p.v.SetEnvPrefix(prefix)
	}
}

// WithConfigType explicitly sets the config type
func WithConfigType(typ string) Option {
	return func(p *Parser) {
		p.v.SetConfigType(typ)
	}
}

// New creates a new parser with default settings applied
func New(opts ...Option) *Parser {
	p := &Parser{
		v:       viper.New(),
		watches: make(map[string]func()),
	}

	// Apply default settings
	p.v.SetEnvPrefix("nexen")
	p.v.AutomaticEnv()
	p.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Apply custom options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Parse reads the configuration from the specified file and unmarshals
// it into a Config struct. The file type is determined from the extension.
func (p *Parser) Parse(configFile string) (*Config, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Set config file and type
	p.v.SetConfigFile(configFile)
	if ext := filepath.Ext(configFile); ext != "" {
		p.v.SetConfigType(ext[1:]) // Remove the leading dot
	}

	// Read configuration
	if err := p.v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file %q: %w", configFile, err)
	}

	// Get all settings as a map
	settings := p.v.AllSettings()

	return &Config{
		Raw:   settings,
		Viper: p.v,
	}, nil
}

// Watch starts watching the config file for changes.
// The callback will be invoked whenever the file changes.
func (p *Parser) Watch(configFile string, callback func()) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove existing watch if any
	if stop, exists := p.watches[configFile]; exists {
		stop()
		delete(p.watches, configFile)
	}

	// Create new watcher
	p.v.WatchConfig()

	// Store callback
	p.watches[configFile] = callback

	// Set callback
	p.v.OnConfigChange(func(e fsnotify.Event) {
		if callback != nil {
			callback()
		}
	})

	return nil
}

// StopWatch stops watching the specified config file
func (p *Parser) StopWatch(configFile string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if stop, exists := p.watches[configFile]; exists {
		stop()
		delete(p.watches, configFile)
	}
}

// Get retrieves a value from the configuration using a dot-notation path
func (p *Parser) Get(path string) interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.v.Get(path)
}

// GetString retrieves a string value from the configuration
func (p *Parser) GetString(path string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.v.GetString(path)
}

// GetInt retrieves an integer value from the configuration
func (p *Parser) GetInt(path string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.v.GetInt(path)
}

// GetBool retrieves a boolean value from the configuration
func (p *Parser) GetBool(path string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.v.GetBool(path)
}

// GetStringMap retrieves a map of strings from the configuration
func (p *Parser) GetStringMap(path string) map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.v.GetStringMap(path)
}

// GetStringSlice retrieves a slice of strings from the configuration
func (p *Parser) GetStringSlice(path string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.v.GetStringSlice(path)
}

// GetEnvPrefix returns the current environment variable prefix
func (p *Parser) GetEnvPrefix() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.v.GetEnvPrefix()
}
