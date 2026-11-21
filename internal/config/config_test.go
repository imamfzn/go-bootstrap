package config_test

import (
	"testing"
	"time"

	"github.com/imamfzn/go-bootstrap/internal/config"
)

func TestLoadAPI(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://app:app@localhost:5432/app")
	t.Setenv("PORT", "9090")
	t.Setenv("ORDER_PENDING_TTL", "2h")

	cfg, err := config.LoadAPI()
	if err != nil {
		t.Fatalf("LoadAPI: %v", err)
	}

	if cfg.DatabaseURL != "postgres://app:app@localhost:5432/app" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.OrderPendingTTL != 2*time.Hour {
		t.Errorf("OrderPendingTTL = %v, want 2h", cfg.OrderPendingTTL)
	}
	if cfg.ShutdownTimeout != 8*time.Second {
		t.Errorf("ShutdownTimeout = %v, want default 8s", cfg.ShutdownTimeout)
	}
	if cfg.TopicOrderCreated != "order-created" {
		t.Errorf("TopicOrderCreated = %q, want default", cfg.TopicOrderCreated)
	}
}

func TestLoadAPI_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := config.LoadAPI(); err == nil {
		t.Fatal("LoadAPI succeeded without DATABASE_URL")
	}
}

func TestLoadWorker_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/app")

	cfg, err := config.LoadWorker()
	if err != nil {
		t.Fatalf("LoadWorker: %v", err)
	}
	if cfg.Concurrency != 4 {
		t.Errorf("Concurrency = %d, want 4", cfg.Concurrency)
	}
	if cfg.SubscriptionOrderCreated != "order-created-worker" {
		t.Errorf("SubscriptionOrderCreated = %q", cfg.SubscriptionOrderCreated)
	}
}
