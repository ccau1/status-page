package envutil

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadDotEnv searches for a .env file starting in the current working directory
// and traversing up the directory tree until it finds one. If found, it populates
// any unset environment variables.
func LoadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, ".env")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			parseAndLoadFile(candidate)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

// LoadFile reads and sets environment variables from a specific file path.
func LoadFile(path string) error {
	return parseAndLoadFile(path)
}

func parseAndLoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var currentKey, currentValue string
	inMultiline := false
	quoteChar := byte(0)

	for scanner.Scan() {
		line := scanner.Text()

		if inMultiline {
			currentValue += "\n" + line
			trimmed := strings.TrimRight(line, " \t\r")
			if len(trimmed) > 0 && trimmed[len(trimmed)-1] == quoteChar {
				// End of multiline
				inMultiline = false
				currentValue = strings.TrimSuffix(currentValue, string(quoteChar))
				if os.Getenv(currentKey) == "" {
					os.Setenv(currentKey, currentValue)
				}
				currentKey, currentValue = "", ""
				quoteChar = 0
			}
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Split on first '='
		idx := strings.Index(trimmed, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(trimmed[:idx])
		val := strings.TrimSpace(trimmed[idx+1:])

		// Check for quoted multiline values (e.g. JSON payloads)
		if len(val) > 0 && (val[0] == '"' || val[0] == '\'') {
			q := val[0]
			if len(val) > 1 && val[len(val)-1] == q {
				// Single-line quoted value
				val = val[1 : len(val)-1]
			} else {
				// Starts multiline quoted value
				inMultiline = true
				quoteChar = q
				currentKey = key
				currentValue = val[1:]
				continue
			}
		}

		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}

	return scanner.Err()
}
