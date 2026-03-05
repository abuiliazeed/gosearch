package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type commandRunner interface {
	Run() error
}

type commandFactory func(name string, args ...string) commandRunner

var browserCommandFactory commandFactory = func(name string, args ...string) commandRunner {
	return exec.Command(name, args...)
}

func openURLInBrowser(url string) error {
	return openURLInBrowserForOS(runtime.GOOS, url, browserCommandFactory)
}

func openURLInBrowserForOS(goos string, url string, factory commandFactory) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("cannot open empty URL")
	}

	commandName, commandArgs, err := browserCommandForOS(goos, url)
	if err != nil {
		return err
	}

	if factory == nil {
		return fmt.Errorf("browser command factory is not configured")
	}

	if err := factory(commandName, commandArgs...).Run(); err != nil {
		return fmt.Errorf("failed to open URL %q: %w", url, err)
	}

	return nil
}

func browserCommandForOS(goos string, url string) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{url}, nil
	case "linux", "freebsd":
		return "xdg-open", []string{url}, nil
	case "windows":
		return "cmd", []string{"/c", "start", "", url}, nil
	default:
		return "", nil, fmt.Errorf("opening URLs is not supported on %s", goos)
	}
}
