package viper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want string
	}{
		{
			name: "default settings",
			opts: nil,
			want: "nexen",
		},
		{
			name: "custom env prefix",
			opts: []Option{WithEnvPrefix("custom")},
			want: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.opts...)
			if got := p.v.GetEnvPrefix(); got != tt.want {
				t.Errorf("New() env prefix = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name       string
		configType string
		content    []byte
		wantErr    bool
	}{
		{
			name:       "valid json",
			configType: "json",
			content:    []byte(`{"key": "value", "number": 42, "nested": {"inner": true}}`),
			wantErr:    false,
		},
		{
			name:       "valid yaml",
			configType: "yaml",
			content: []byte(`
key: value
number: 42
nested:
  inner: true
`),
			wantErr: false,
		},
		{
			name:       "invalid json",
			configType: "json",
			content:    []byte(`{invalid`),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config."+tt.configType)
			if err := os.WriteFile(configFile, tt.content, 0644); err != nil {
				t.Fatal(err)
			}

			p := New()
			cfg, err := p.Parse(configFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the parsed content
				if cfg == nil {
					t.Error("Parse() returned nil Config")
					return
				}

				// Test some basic values
				if v, ok := cfg.Raw["key"].(string); !ok || v != "value" {
					t.Errorf("Parse() key = %v, want 'value'", v)
				}
				// Check for both float64 and int types since viper might return either
				if v, ok := cfg.Raw["number"].(float64); ok {
					if int(v) != 42 {
						t.Errorf("Parse() number = %v, want 42", v)
					}
				} else if v, ok := cfg.Raw["number"].(int); ok {
					if v != 42 {
						t.Errorf("Parse() number = %v, want 42", v)
					}
				} else {
					t.Errorf("Parse() number is neither float64 nor int, got type %T", cfg.Raw["number"])
				}
			}
		})
	}
}

func TestParser_Watch(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	initialContent := []byte(`{"key": "initial"}`)
	if err := os.WriteFile(configFile, initialContent, 0644); err != nil {
		t.Fatal(err)
	}

	p := New()
	if _, err := p.Parse(configFile); err != nil {
		t.Fatal(err)
	}

	// Channel to receive watch notifications
	changes := make(chan struct{}, 1)

	// Start watching
	if err := p.Watch(configFile, func() {
		changes <- struct{}{}
	}); err != nil {
		t.Fatal(err)
	}

	// Modify the file
	newContent := []byte(`{"key": "modified"}`)
	if err := os.WriteFile(configFile, newContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for change notification
	select {
	case <-changes:
		// Verify the content was updated
		if got := p.GetString("key"); got != "modified" {
			t.Errorf("Watch() updated value = %v, want 'modified'", got)
		}
	case <-time.After(time.Second):
		t.Error("Watch() timeout waiting for change notification")
	}

	// Test stopping the watch
	p.StopWatch(configFile)
}

func TestParser_GetMethods(t *testing.T) {
	content := []byte(`{
		"string": "value",
		"int": 42,
		"bool": true,
		"stringSlice": ["a", "b", "c"],
		"stringMap": {
			"key1": "value1",
			"key2": "value2"
		}
	}`)

	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	p := New()
	if _, err := p.Parse(configFile); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		getFunc  func() interface{}
		want     interface{}
		typeName string
	}{
		{
			name:     "GetString",
			getFunc:  func() interface{} { return p.GetString("string") },
			want:     "value",
			typeName: "string",
		},
		{
			name:     "GetInt",
			getFunc:  func() interface{} { return p.GetInt("int") },
			want:     42,
			typeName: "int",
		},
		{
			name:     "GetBool",
			getFunc:  func() interface{} { return p.GetBool("bool") },
			want:     true,
			typeName: "bool",
		},
		{
			name:     "GetStringSlice",
			getFunc:  func() interface{} { return p.GetStringSlice("stringSlice") },
			want:     []string{"a", "b", "c"},
			typeName: "[]string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.getFunc()
			if !jsonEqual(got, tt.want) {
				t.Errorf("%s() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}

	// Test GetStringMap separately due to type comparison complexity
	t.Run("GetStringMap", func(t *testing.T) {
		got := p.GetStringMap("stringMap")
		want := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		if !jsonEqual(got, want) {
			t.Errorf("GetStringMap() = %v, want %v", got, want)
		}
	})
}

// jsonEqual compares two values by marshalling them to JSON
func jsonEqual(a, b interface{}) bool {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}
