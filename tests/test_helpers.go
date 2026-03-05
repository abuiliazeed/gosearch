// Package tests provides integration test helpers and utilities for gosearch.
package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestConfig holds configuration for integration tests.
type TestConfig struct {
	// DataDir is the temporary directory for test data.
	DataDir string
	// BinPath is the path to the gosearch binary.
	BinPath string
	// RedisAddr is the Redis server address.
	RedisAddr string
	// APIPort is the port for the API server during tests.
	APIPort int
	// TestTimeout is the timeout for test operations.
	TestTimeout time.Duration
}

// TestFixture holds sample data for testing.
type TestFixture struct {
	// SampleURL is a test URL for crawling.
	SampleURL string
	// SampleHTML is sample HTML content for testing.
	SampleHTML string
	// SampleQueries is a list of test queries.
	SampleQueries []string
	// ExpectedResults is expected search results.
	ExpectedResults map[string][]string
}

// NewTestConfig creates a new test configuration with defaults.
func NewTestConfig(t *testing.T) *TestConfig {
	tmpDir, err := os.MkdirTemp("", "gosearch_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Find the binary path
	binPath := filepath.Join("..", "bin", "gosearch")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	return &TestConfig{
		DataDir:     tmpDir,
		BinPath:     binPath,
		RedisAddr:   "localhost:6379",
		APIPort:     18080, // Use different port to avoid conflicts
		TestTimeout: 30 * time.Second,
	}
}

// Cleanup removes the temporary test data directory.
func (c *TestConfig) Cleanup() {
	if c.DataDir != "" {
		os.RemoveAll(c.DataDir)
	}
}

// IndexDir returns the path to the index directory.
func (c *TestConfig) IndexDir() string {
	return filepath.Join(c.DataDir, "index")
}

// PagesDir returns the path to the pages directory.
func (c *TestConfig) PagesDir() string {
	return filepath.Join(c.DataDir, "pages")
}

// BackupPath returns the path to a backup file.
func (c *TestConfig) BackupPath(name string) string {
	return filepath.Join(c.DataDir, name+".bin")
}

// ServerAddr returns the server address for the API.
func (c *TestConfig) ServerAddr() string {
	return fmt.Sprintf(":%d", c.APIPort)
}

// ServerURL returns the server URL for the API.
func (c *TestConfig) ServerURL() string {
	return fmt.Sprintf("http://localhost:%d", c.APIPort)
}

// CreateContext returns a context with test timeout.
func (c *TestConfig) CreateContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.TestTimeout)
}

// SampleFixture returns a sample test fixture.
func SampleFixture() *TestFixture {
	return &TestFixture{
		SampleURL: "https://example.com",
		SampleHTML: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Example Domain</title>
</head>
<body>
    <h1>Example Domain</h1>
    <p>This domain is for use in illustrative examples in documents.</p>
    <p>You may use this domain in literature without prior coordination or asking permission.</p>
    <a href="/page1">Page 1</a>
    <a href="/page2">Page 2</a>
</body>
</html>`,
		SampleQueries: []string{
			"example",
			"domain",
			"illustrative",
		},
		ExpectedResults: map[string][]string{
			"example": {"Example Domain"},
			"domain":  {"Example Domain"},
		},
	}
}

// SkipIfMissingBinary skips the test if the binary is not found.
func SkipIfMissingBinary(t *testing.T, binPath string) {
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Skipf("binary not found: %s (run 'make build' first)", binPath)
	}
}

// SkipIfNoRedis skips the test if Redis is not available.
func SkipIfNoRedis(t *testing.T, _ string) {
	// Simple check - try to connect to Redis
	// In a real scenario, we'd check if Redis is running
	// For now, we'll skip if the REDIS_TEST environment variable is not set
	if os.Getenv("REDIS_TEST") != "1" {
		t.Skip("skipping Redis-dependent tests (set REDIS_TEST=1 to enable)")
	}
}

// SetupTestDir creates the test directory structure.
func SetupTestDir(dataDir string) error {
	dirs := []string{
		filepath.Join(dataDir, "index"),
		filepath.Join(dataDir, "pages"),
		filepath.Join(dataDir, "cache"),
	}

	for _, dir := range dirs {
		// #nosec G301 -- Test code, temp directory with known permissions
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// CleanupTestDir removes the test directory structure.
func CleanupTestDir(dataDir string) error {
	if dataDir == "" {
		return nil
	}
	return os.RemoveAll(dataDir)
}

// WriteSampleHTML writes sample HTML to a file for testing.
func WriteSampleHTML(path string, html string) error {
	// #nosec G306 -- Test code, temp file with known permissions
	return os.WriteFile(path, []byte(html), 0644)
}

// ReadTestdata reads a file from the testdata directory.
func ReadTestdata(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("testdata", name)
	// #nosec G304 -- Test code, reading from known testdata directory
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read testdata file %s: %v", path, err)
	}

	return data
}

// MockHTTPServerConfig holds configuration for a mock HTTP server.
type MockHTTPServerConfig struct {
	Port         int
	ResponseCode int
	ResponseBody string
	Headers      map[string]string
}

// DefaultMockServerConfig returns default configuration for a mock server.
func DefaultMockServerConfig() *MockHTTPServerConfig {
	return &MockHTTPServerConfig{
		Port:         9999,
		ResponseCode: 200,
		ResponseBody: SampleFixture().SampleHTML,
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
	}
}

// BlockResponseConfig returns configuration for a server that returns 429.
func BlockResponseConfig() *MockHTTPServerConfig {
	return &MockHTTPServerConfig{
		Port:         9999,
		ResponseCode: 429,
		ResponseBody: `<html><body>Too Many Requests</body></html>`,
		Headers: map[string]string{
			"Content-Type": "text/html",
			"Retry-After":  "5",
		},
	}
}

// AssertContains asserts that a string contains a substring.
func AssertContains(t *testing.T, s, substr string) {
	t.Helper()

	if !contains(s, substr) {
		t.Errorf("expected string to contain %q, but got: %s", substr, s)
	}
}

// AssertNotContains asserts that a string does not contain a substring.
func AssertNotContains(t *testing.T, s, substr string) {
	t.Helper()

	if contains(s, substr) {
		t.Errorf("expected string NOT to contain %q, but got: %s", substr, s)
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
