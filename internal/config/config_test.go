package config

import "testing"

func TestLoadValidatesServiceName(t *testing.T) {
	t.Setenv("NATS_URL", "nats://ckguesser_be:test@nats:4222")
	t.Setenv("SERVICE_NAME", "ckguesser-be")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.ServiceName != "ckguesser-be" {
		t.Fatalf("ServiceName = %q, want %q", cfg.ServiceName, "ckguesser-be")
	}
	if cfg.HTTPAddr != ":8081" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8081")
	}
}

func TestLoadRejectsInvalidServiceName(t *testing.T) {
	t.Setenv("NATS_URL", "nats://ckguesser_be:test@nats:4222")
	t.Setenv("SERVICE_NAME", "wrong-service")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected error for invalid SERVICE_NAME, got nil")
	}
}
