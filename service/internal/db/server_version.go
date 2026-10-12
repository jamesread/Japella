package db

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	minMariaDBMajor = 10
	minMariaDBMinor = 5
)

// Matches N.N.N-MariaDB, optionally followed by a packaging suffix (e.g. -ubu2404).
var mariaDBVersionSuffix = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)-[Mm]aria[Dd][Bb]`)

// ValidateSupportedDBServerVersion ensures the server is MariaDB 10.5+ (required by SQL migrations).
func ValidateSupportedDBServerVersion(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("unsupported database server: empty VERSION()")
	}

	if !strings.Contains(strings.ToLower(version), "mariadb") {
		return fmt.Errorf("unsupported database server: Japella requires MariaDB %d.%d or newer (connected to %q)", minMariaDBMajor, minMariaDBMinor, version)
	}

	major, minor, ok := parseMariaDBVersion(version)
	if !ok {
		return fmt.Errorf("unsupported database server: could not parse MariaDB version from %q", version)
	}

	if major < minMariaDBMajor || (major == minMariaDBMajor && minor < minMariaDBMinor) {
		return fmt.Errorf("unsupported database server: Japella requires MariaDB %d.%d or newer (connected to %q)", minMariaDBMajor, minMariaDBMinor, version)
	}

	return nil
}

func parseMariaDBVersion(version string) (major, minor int, ok bool) {
	m := mariaDBVersionSuffix.FindStringSubmatch(version)
	if len(m) != 4 {
		return 0, 0, false
	}
	major, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, false
	}
	minor, err = strconv.Atoi(m[2])
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}
