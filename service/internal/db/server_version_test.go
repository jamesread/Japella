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
		{"11.8.9-MariaDB-ubu2404", false},
		{"13.0.2-MariaDB-ubu2604", false},
		{"11.8.9-MariaDB-log", false},
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
	tests := []struct {
		version      string
		wantMajor    int
		wantMinor    int
		wantOK       bool
	}{
		{"5.5.5-10.11.6-MariaDB", 10, 11, true},
		{"11.8.9-MariaDB-ubu2404", 11, 8, true},
		{"13.0.2-MariaDB-ubu2604", 13, 0, true},
		{"10.11.15-MariaDB", 10, 11, true},
	}
	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			major, minor, ok := parseMariaDBVersion(tc.version)
			if ok != tc.wantOK || major != tc.wantMajor || minor != tc.wantMinor {
				t.Fatalf("got %d.%d ok=%v, want %d.%d ok=%v", major, minor, ok, tc.wantMajor, tc.wantMinor, tc.wantOK)
			}
		})
	}
}
