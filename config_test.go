package seekdb

import (
	"testing"
)

func TestVersionConstants(t *testing.T) {
	if Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", Version, "1.0.0")
	}
	if Name != "seekdb-go" {
		t.Errorf("Name = %q, want %q", Name, "seekdb-go")
	}
}

func TestDistanceMetricConstants(t *testing.T) {
	tests := []struct {
		name     string
		metric   DistanceMetric
		expected string
	}{
		{"cosine", DistanceCosine, "cosine"},
		{"l2", DistanceL2, "l2"},
		{"ip", DistanceIP, "ip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.metric) != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.metric, tt.expected)
			}
		})
	}
}

func TestClientConfig(t *testing.T) {
	config := ClientConfig{
		Path:     "/data/seekdb",
		Host:     "127.0.0.1",
		Port:     2881,
		User:     "root",
		Password: "secret",
		Tenant:   "test",
		Database: "mydb",
	}

	if config.Path != "/data/seekdb" {
		t.Errorf("Path = %q, want %q", config.Path, "/data/seekdb")
	}
	if config.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want %q", config.Host, "127.0.0.1")
	}
	if config.Port != 2881 {
		t.Errorf("Port = %d, want %d", config.Port, 2881)
	}
	if config.User != "root" {
		t.Errorf("User = %q, want %q", config.User, "root")
	}
	if config.Password != "secret" {
		t.Errorf("Password = %q, want %q", config.Password, "secret")
	}
	if config.Tenant != "test" {
		t.Errorf("Tenant = %q, want %q", config.Tenant, "test")
	}
	if config.Database != "mydb" {
		t.Errorf("Database = %q, want %q", config.Database, "mydb")
	}
}

func TestAdminConfig(t *testing.T) {
	config := AdminConfig{
		Path:     "/data/seekdb",
		Host:     "127.0.0.1",
		Port:     2881,
		User:     "admin",
		Password: "adminpass",
		Tenant:   "sys",
	}

	if config.Path != "/data/seekdb" {
		t.Errorf("Path = %q, want %q", config.Path, "/data/seekdb")
	}
	if config.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want %q", config.Host, "127.0.0.1")
	}
	if config.Port != 2881 {
		t.Errorf("Port = %d, want %d", config.Port, 2881)
	}
	if config.User != "admin" {
		t.Errorf("User = %q, want %q", config.User, "admin")
	}
	if config.Password != "adminpass" {
		t.Errorf("Password = %q, want %q", config.Password, "adminpass")
	}
	if config.Tenant != "sys" {
		t.Errorf("Tenant = %q, want %q", config.Tenant, "sys")
	}
}

func TestClientConfigDefaults(t *testing.T) {
	// Test that empty config fields have expected defaults in server mode
	config := ClientConfig{
		Host: "localhost",
	}

	// Note: defaults are applied in newServerClient, not in the struct itself
	// This test documents the expected behavior
	if config.Port != 0 {
		t.Error("Port should default to 0 in struct (2881 applied later)")
	}
	if config.User != "" {
		t.Error("User should default to empty in struct (root applied later)")
	}
	if config.Database != "" {
		t.Error("Database should default to empty in struct (test applied later)")
	}
}