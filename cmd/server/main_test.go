package main

import "testing"

func TestValidateStartupConfigRejectsShortSecret(t *testing.T) {
	err := validateStartupConfig("short", "", "development")
	if err == nil {
		t.Fatal("expected short SECRET_KEY to be rejected")
	}
}

func TestValidateStartupConfigRequiresProductionAuthURL(t *testing.T) {
	err := validateStartupConfig("01234567890123456789012345678901", "", "production")
	if err == nil {
		t.Fatal("expected production AUTH_URL to be required")
	}
}

func TestValidateStartupConfigAllowsDevelopmentWithoutAuthURL(t *testing.T) {
	err := validateStartupConfig("01234567890123456789012345678901", "", "development")
	if err != nil {
		t.Fatalf("expected development config to be accepted: %v", err)
	}
}
