package shared

import "os"

// EnvDefault returns the value of the environment variable or defaultValue.
func EnvDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
