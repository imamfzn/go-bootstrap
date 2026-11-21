// Package config reads environment variables. It is the only package that
// does so. Each binary loads only the struct it needs.
package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Base holds what every binary needs.
type Base struct {
	ServiceName string `envconfig:"SERVICE_NAME" default:"go-bootstrap"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`

	DatabaseURL              string `envconfig:"DATABASE_URL"`
	DBInstanceConnectionName string `envconfig:"DB_INSTANCE_CONNECTION_NAME"`
	DBIAMAuth                bool   `envconfig:"DB_IAM_AUTH" default:"false"`
	DBMaxConns               int32  `envconfig:"DB_MAX_CONNS" default:"5"`

	PubSubProjectID   string `envconfig:"PUBSUB_PROJECT_ID"`
	TopicOrderCreated string `envconfig:"TOPIC_ORDER_CREATED" default:"order-created"`

	PaymentBaseURL string        `envconfig:"PAYMENT_BASE_URL"`
	PaymentAPIKey  string        `envconfig:"PAYMENT_API_KEY"`
	PaymentTimeout time.Duration `envconfig:"PAYMENT_TIMEOUT" default:"10s"`

	OrderPendingTTL time.Duration `envconfig:"ORDER_PENDING_TTL" default:"24h"`
}

// API is the configuration of cmd/api.
type API struct {
	Base

	Port            int           `envconfig:"PORT" default:"8080"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"8s"`
	AuthDevInsecure bool          `envconfig:"AUTH_DEV_INSECURE" default:"false"`
}

// Worker is the configuration of cmd/worker.
type Worker struct {
	Base

	SubscriptionOrderCreated string        `envconfig:"SUBSCRIPTION_ORDER_CREATED" default:"order-created-worker"`
	Concurrency              int           `envconfig:"WORKER_CONCURRENCY" default:"4"`
	HandlerTimeout           time.Duration `envconfig:"WORKER_HANDLER_TIMEOUT" default:"30s"`
}

// Job is the configuration of cmd/job.
type Job struct {
	Base
}

// validate rejects a configuration no binary can start with. envconfig's own
// "required" only checks that a variable exists, so a variable set to an
// empty string, which is what an unset Terraform input renders as, would
// pass it.
func (b Base) validate() error {
	if b.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	return nil
}

// LoadAPI reads the API configuration from the environment.
func LoadAPI() (API, error) {
	var cfg API
	if err := envconfig.Process("", &cfg); err != nil {
		return API{}, fmt.Errorf("api config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return API{}, fmt.Errorf("api config: %w", err)
	}
	return cfg, nil
}

// LoadWorker reads the worker configuration from the environment.
func LoadWorker() (Worker, error) {
	var cfg Worker
	if err := envconfig.Process("", &cfg); err != nil {
		return Worker{}, fmt.Errorf("worker config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Worker{}, fmt.Errorf("worker config: %w", err)
	}
	return cfg, nil
}

// LoadJob reads the job configuration from the environment.
func LoadJob() (Job, error) {
	var cfg Job
	if err := envconfig.Process("", &cfg); err != nil {
		return Job{}, fmt.Errorf("job config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Job{}, fmt.Errorf("job config: %w", err)
	}
	return cfg, nil
}
