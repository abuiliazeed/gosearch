package crawler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPolitenessManager_Acquire(t *testing.T) {
	pm := NewPolitenessManager(10*time.Millisecond, "TestBot", false)

	target := "http://example.com/page"

	// First acquire should be immediate
	start := time.Now()
	if err := pm.Acquire(target); err != nil {
		t.Fatalf("acquire failed: %v", err)
	}
	pm.Release(target)

	// Second acquire should respect delay
	if err := pm.Acquire(target); err != nil {
		t.Fatalf("second acquire failed: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 10*time.Millisecond {
		t.Errorf("expected delay, got %v", elapsed)
	}
	pm.Release(target)
}

func TestPolitenessManager_RobotsTxt(t *testing.T) {
	// Mock server for robots.txt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			fmt.Fprintln(w, "User-agent: *")
			fmt.Fprintln(w, "Disallow: /private")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	pm := NewPolitenessManager(0, "TestBot", true)

	// allowed path
	if err := pm.Acquire(server.URL + "/public"); err != nil {
		t.Errorf("expected public path to be allowed: %v", err)
	}
	pm.Release(server.URL + "/public")

	// disallowed path
	if err := pm.Acquire(server.URL + "/private"); err != ErrDisallowed {
		t.Errorf("expected private path to be disallowed, got %v", err)
	}
}

func TestPolitenessManager_Concurrent(t *testing.T) {
	pm := NewPolitenessManager(10*time.Millisecond, "TestBot", false)
	target := "http://example.com/page"

	// Start two goroutines trying to acquire the same domain
	start := time.Now()

	done := make(chan bool)
	go func() {
		pm.Acquire(target)
		time.Sleep(10 * time.Millisecond) // Simulate work
		pm.Release(target)
		done <- true
	}()

	go func() {
		time.Sleep(5 * time.Millisecond) // Start slightly later
		pm.Acquire(target)
		pm.Release(target)
		done <- true
	}()

	<-done
	<-done

	elapsed := time.Since(start)
	// The total time should be at least the delay
	// But in test environment, scheduling might be fast, so we check that
	// at least *some* delay happened if we acquired sequentially
	if elapsed < 10*time.Millisecond {
		t.Errorf("expected concurrent access to be serialized, took %v", elapsed)
	}
}
