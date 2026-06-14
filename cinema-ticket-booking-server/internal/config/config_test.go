package config_test

import (
    "testing"
    "cinema-ticket-booking/internal/config"
)

func TestConfigDefaults(t *testing.T) {
    cfg := config.Load()
    if cfg.Port == "" {
        t.Error("Port should default to 8080")
    }
    if cfg.MongoDB == "" {
        t.Error("MongoDB should default to 'cinema'")
    }
}
