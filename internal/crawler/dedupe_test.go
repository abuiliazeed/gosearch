package crawler

import (
	"testing"
)

func TestDeduplicator_Seen(t *testing.T) {
	// Test without bloom filter first (deterministic)
	d := NewDeduplicator(false, 100)

	url1 := "http://example.com/1"
	url2 := "http://example.com/2"

	if d.Seen(url1) {
		t.Error("expected url1 to not be seen yet")
	}

	d.Add(url1)

	if !d.Seen(url1) {
		t.Error("expected url1 to be seen")
	}

	if d.Seen(url2) {
		t.Error("expected url2 to not be seen")
	}
}

func TestDeduplicator_Bloom(t *testing.T) {
	// Test with bloom filter
	d := NewDeduplicator(true, 100)

	url1 := "http://example.com/1"

	if d.Seen(url1) {
		t.Error("expected url1 to not be seen yet")
	}

	d.Add(url1)

	if !d.Seen(url1) {
		t.Error("expected url1 to be seen")
	}
}

func TestDeduplicator_Clear(t *testing.T) {
	d := NewDeduplicator(false, 100)
	url := "http://example.com"

	d.Add(url)
	if !d.Seen(url) {
		t.Fatal("setup failed: url not seen")
	}

	d.Clear()
	if d.Seen(url) {
		t.Error("expected url to differ after clear")
	}
}
