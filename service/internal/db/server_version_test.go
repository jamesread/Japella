package db

import "testing"

func TestValidateSupportedDBServerVersion(t *testing.T) {
	tests := []struct {
		version string
		wantErr bool
	}{
		{"10.11.6-MariaDB", false},
		{"10.5.15-MariaDB", false},
		{"10.5.0-MariaDB", false},
		{"5.5.5-10.11.6-MariaDB", false},
		{"11.4.2-MariaDB", false},
		{"10.4.32-MariaDB", true},
		{"8.0.36", true},
		{"8.0.36-MySQL Community Server", true},
		{"", true},
		{"mariadb without semver", true},
	}

	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			err := ValidateSupportedDBServerVersion(tc.version)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseMariaDBVersion(t *testing.T) {
	major, minor, ok := parseMariaDBVersion("5.5.5-10.11.6-MariaDB")
	if !ok || major != 10 || minor != 11 {
		t.Fatalf("got %d.%d ok=%v", major, minor, ok)
	}
}
