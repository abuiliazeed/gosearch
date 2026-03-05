package progress

import (
	"testing"
)

func TestProgress_New(t *testing.T) {
	desc := "Test Progress"
	max := int64(100)
	p := NewSimple(max, desc)

	if p == nil {
		t.Fatal("NewSimple() returned nil")
	}
}

func TestProgress_Add(t *testing.T) {
	p := NewSimple(10, "Test")
	err := p.Add(1)
	if err != nil {
		t.Errorf("Add failed: %v", err)
	}
}

func TestProgress_Finish(t *testing.T) {
	p := NewSimple(10, "Test")
	err := p.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
