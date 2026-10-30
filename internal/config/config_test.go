package config

import (
	"log/slog"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := LoadFrom(env(nil))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Addr != ":8080" || cfg.Env != "dev" || cfg.LogFormat != "text" ||
		cfg.LogLevel != slog.LevelInfo || cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
		check   func(t *testing.T, cfg Config)
	}{
		{
			name: "prod defaults to json logs",
			env:  map[string]string{"APP_ENV": "prod", "DATABASE_URL": "postgres://x"},
			check: func(t *testing.T, cfg Config) {
				if cfg.LogFormat != "json" {
					t.Errorf("LogFormat = %q, want json", cfg.LogFormat)
				}
			},
		},
		{
			name: "cors origins are split and trimmed",
			env:  map[string]string{"CORS_ALLOWED_ORIGINS": " https://a.example , ,https://b.example"},
			check: func(t *testing.T, cfg Config) {
				if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[1] != "https://b.example" {
					t.Errorf("CORSOrigins = %q", cfg.CORSOrigins)
				}
			},
		},
		{name: "prod requires database", env: map[string]string{"APP_ENV": "prod"}, wantErr: true},
		{name: "unknown env", env: map[string]string{"APP_ENV": "staging"}, wantErr: true},
		{name: "bad log level", env: map[string]string{"LOG_LEVEL": "loud"}, wantErr: true},
		{name: "bad duration", env: map[string]string{"SHUTDOWN_TIMEOUT": "soon"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadFrom(env(tt.env))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}
