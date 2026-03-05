// Package progress provides progress tracking for long-running operations in gosearch.
//
// It wraps the progressbar library to provide a consistent interface
// for displaying progress during crawling, indexing, and other operations.
package progress

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Progress represents a progress bar for tracking operation progress.
type Progress struct {
	bar *progressbar.ProgressBar
}

// Config holds configuration for a progress bar.
type Config struct {
	Total       int64     // Total items to process
	Description string    // Description of the operation
	Writer      io.Writer // Output writer (defaults to stderr)
	ShowBytes   bool      // Show bytes instead of count
	ShowPercent bool      // Show percentage completion
	ShowCount   bool      // Show current/total count
	ShowIters   bool      // Show iterations per second
}

// DefaultConfig returns the default progress bar configuration.
func DefaultConfig() *Config {
	return &Config{
		Total:       100,
		Description: "Processing",
		Writer:      os.Stderr,
		ShowBytes:   false,
		ShowPercent: true,
		ShowCount:   true,
		ShowIters:   true,
	}
}

// New creates a new progress bar with the given configuration.
func New(cfg *Config) *Progress {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Determine if we should show the progress bar
	// Only show if stdout is a terminal (not piped)
	showBar := isTerminal()

	bar := progressbar.NewOptions64(
		cfg.Total,
		progressbar.OptionSetDescription(cfg.Description),
		progressbar.OptionSetWriter(cfg.Writer),
		progressbar.OptionShowBytes(cfg.ShowBytes),
		progressbar.OptionSetVisibility(showBar),
		progressbar.OptionOnCompletion(func() {
			if showBar && cfg.Writer == os.Stderr {
				fmt.Fprintln(cfg.Writer)
			}
		}),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionFullWidth(),
	)

	return &Progress{bar: bar}
}

// NewSimple creates a simple progress bar with default options.
func NewSimple(total int64, description string) *Progress {
	return New(&Config{
		Total:       total,
		Description: description,
	})
}

// Add increments the progress bar by the given amount.
func (p *Progress) Add(n int64) error {
	if p.bar != nil {
		return p.bar.Add64(n)
	}
	return nil
}

// Set sets the current progress to the given value.
func (p *Progress) Set(n int64) error {
	if p.bar != nil {
		return p.bar.Set64(n)
	}
	return nil
}

// Close finishes the progress bar.
func (p *Progress) Close() error {
	if p.bar != nil {
		return p.bar.Finish()
	}
	return nil
}

// Describe changes the description of the progress bar.
func (p *Progress) Describe(description string) {
	if p.bar != nil {
		p.bar.Describe(description)
	}
}

// IsTerminal returns true if stdout is a terminal.
func isTerminal() bool {
	return isTerminalFile(os.Stdout) || isTerminalFile(os.Stderr)
}

// isTerminalFile returns true if the file is a terminal.
func isTerminalFile(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}

	// Check if this is a character device
	return (info.Mode() & os.ModeCharDevice) != 0
}

// CrawlerProgress tracks crawling progress with statistics.
type CrawlerProgress struct {
	total      *Progress
	urlsQueued *Progress
}

// NewCrawlerProgress creates a new crawler progress tracker.
func NewCrawlerProgress(expectedURLs int64) *CrawlerProgress {
	return &CrawlerProgress{
		total:      NewSimple(expectedURLs, "Crawling"),
		urlsQueued: NewSimple(expectedURLs, "Queueing URLs"),
	}
}

// AddCrawled increments the crawled counter.
func (cp *CrawlerProgress) AddCrawled(n int64) {
	if cp.total != nil {
		_ = cp.total.Add(n)
	}
}

// AddQueued increments the queued counter.
func (cp *CrawlerProgress) AddQueued(n int64) {
	if cp.urlsQueued != nil {
		_ = cp.urlsQueued.Add(n)
	}
}

// Close closes all progress bars.
func (cp *CrawlerProgress) Close() {
	if cp.total != nil {
		cp.total.Close()
	}
	if cp.urlsQueued != nil {
		cp.urlsQueued.Close()
	}
}

// IndexerProgress tracks indexing progress with statistics.
type IndexerProgress struct {
	docs  *Progress
	terms *Progress
}

// NewIndexerProgress creates a new indexer progress tracker.
func NewIndexerProgress(totalDocs, totalTerms int64) *IndexerProgress {
	return &IndexerProgress{
		docs:  NewSimple(totalDocs, "Indexing documents"),
		terms: NewSimple(totalTerms, "Indexing terms"),
	}
}

// AddDoc increments the document counter.
func (ip *IndexerProgress) AddDoc(n int64) {
	if ip.docs != nil {
		_ = ip.docs.Add(n)
	}
}

// AddTerm increments the term counter.
func (ip *IndexerProgress) AddTerm(n int64) {
	if ip.terms != nil {
		_ = ip.terms.Add(n)
	}
}

// Close closes all progress bars.
func (ip *IndexerProgress) Close() {
	if ip.docs != nil {
		ip.docs.Close()
	}
	if ip.terms != nil {
		ip.terms.Close()
	}
}

// SilentProgress is a no-op progress tracker for non-interactive environments.
type SilentProgress struct{}

// NewSilent creates a silent progress tracker.
func NewSilent() *SilentProgress {
	return &SilentProgress{}
}

// Add does nothing.
func (sp *SilentProgress) Add(_ int64) error {
	return nil
}

// Set does nothing.
func (sp *SilentProgress) Set(_ int64) error {
	return nil
}

// Close does nothing.
func (sp *SilentProgress) Close() error {
	return nil
}

// Describe does nothing.
func (sp *SilentProgress) Describe(_ string) {}
