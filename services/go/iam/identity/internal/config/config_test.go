package config

import "testing"

func TestValidateJWTAccessSecretAllowsLocalDevDefaults(t *testing.T) {
	if err := validateJWTAccessSecret("local", defaultJWTAccessSecret); err != nil {
		t.Fatalf("expected local env to allow default secret, got %v", err)
	}
	if err := validateJWTAccessSecret("test", ""); err != nil {
		t.Fatalf("expected test env to allow empty secret fallback behavior, got %v", err)
	}
}

func TestValidateJWTAccessSecretRejectsDefaultInProduction(t *testing.T) {
	if err := validateJWTAccessSecret("production", defaultJWTAccessSecret); err == nil {
		t.Fatal("expected production env to reject default jwt secret")
	}
	if err := validateJWTAccessSecret("staging", ""); err == nil {
		t.Fatal("expected non-local env to reject empty jwt secret")
	}
	if err := validateJWTAccessSecret("production", "real-secret-value"); err != nil {
		t.Fatalf("expected production env to accept custom secret, got %v", err)
	}
}
