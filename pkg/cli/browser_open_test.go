package cli

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestBrowserCommandForOS(t *testing.T) {
	tests := []struct {
		name      string
		goos      string
		url       string
		wantCmd   string
		wantArgs  []string
		wantError bool
	}{
		{
			name:     "darwin",
			goos:     "darwin",
			url:      "https://example.com",
			wantCmd:  "open",
			wantArgs: []string{"https://example.com"},
		},
		{
			name:     "linux",
			goos:     "linux",
			url:      "https://example.com",
			wantCmd:  "xdg-open",
			wantArgs: []string{"https://example.com"},
		},
		{
			name:     "freebsd",
			goos:     "freebsd",
			url:      "https://example.com",
			wantCmd:  "xdg-open",
			wantArgs: []string{"https://example.com"},
		},
		{
			name:     "windows",
			goos:     "windows",
			url:      "https://example.com",
			wantCmd:  "cmd",
			wantArgs: []string{"/c", "start", "", "https://example.com"},
		},
		{
			name:      "unsupported",
			goos:      "plan9",
			url:       "https://example.com",
			wantError: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			gotCmd, gotArgs, err := browserCommandForOS(testCase.goos, testCase.url)
			if testCase.wantError {
				if err == nil {
					t.Fatalf("expected error for %s, got nil", testCase.goos)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotCmd != testCase.wantCmd {
				t.Fatalf("expected command %q, got %q", testCase.wantCmd, gotCmd)
			}
			if !reflect.DeepEqual(gotArgs, testCase.wantArgs) {
				t.Fatalf("expected args %v, got %v", testCase.wantArgs, gotArgs)
			}
		})
	}
}

func TestOpenURLInBrowserForOS_CommandFailure(t *testing.T) {
	commandErr := errors.New("boom")

	factory := func(name string, args ...string) commandRunner {
		return fakeCommandRunner{runErr: commandErr}
	}

	err := openURLInBrowserForOS("darwin", "https://example.com", factory)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, commandErr) {
		t.Fatalf("expected wrapped command error %v, got %v", commandErr, err)
	}
	if !strings.Contains(err.Error(), "failed to open URL") {
		t.Fatalf("expected actionable message, got %q", err.Error())
	}
}

type fakeCommandRunner struct {
	runErr error
}

func (f fakeCommandRunner) Run() error {
	return f.runErr
}
