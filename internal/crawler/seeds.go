// Package crawler provides seed URL management and predefined seed sets.
package crawler

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// SeedSetType represents the type of predefined seed set.
type SeedSetType string

const (
	// SeedSetGeneral is a general purpose seed set covering diverse topics.
	SeedSetGeneral SeedSetType = "general"
	// SeedSetProgramming is a programming-focused seed set.
	SeedSetProgramming SeedSetType = "programming"
	// SeedSetAcademic is an academic/research focused seed set.
	SeedSetAcademic SeedSetType = "academic"
)

// SeedSet represents a collection of seed URLs with metadata.
type SeedSet struct {
	// Name is the identifier for this seed set.
	Name string `json:"name"`
	// Description explains what this seed set contains.
	Description string `json:"description"`
	// URLs is the list of seed URLs in this set.
	URLs []string `json:"urls"`
}

// SeedConfig represents seed configuration loaded from a file.
type SeedConfig struct {
	// Sets is a map of named seed sets.
	Sets map[string]*SeedSet `json:"sets"`
	// Default specifies which seed set to use by default.
	Default string `json:"default"`
}

// predefinedSeedSets contains all built-in seed sets.
var predefinedSeedSets = map[SeedSetType]*SeedSet{
	SeedSetGeneral: {
		Name:        "general",
		Description: "General purpose seed set covering diverse topics and regions",
		URLs: []string{
			"https://example.com",
			"https://httpbin.org",
			"https://www.wikipedia.org",
			"https://github.com",
			"https://stackoverflow.com",
			"https://reddit.com",
			"https://news.ycombinator.com",
			"https://mozilla.org",
			"https://w3.org",
			"https://www.json.org",
		},
	},
	SeedSetProgramming: {
		Name:        "programming",
		Description: "Programming and development focused seed set",
		URLs: []string{
			"https://github.com",
			"https://stackoverflow.com",
			"https://developer.mozilla.org",
			"https://docs.rs",
			"https://pkg.go.dev",
			"https://python.org",
			"https://nodejs.org",
			"https://rust-lang.org",
			"https://go.dev",
			"https://www typescriptlang.org",
			"https://react.dev",
			"https://nextjs.org",
			"https://vuejs.org",
			"https://lodash.com",
			"https://expressjs.com",
			"https://fastify.io",
			"https://www.freecodecamp.org",
			"https://www.theodinproject.com",
		},
	},
	SeedSetAcademic: {
		Name:        "academic",
		Description: "Academic and research focused seed set",
		URLs: []string{
			"https://scholar.google.com",
			"https://arxiv.org",
			"https://www.researchgate.net",
			"https://www.jstor.org",
			"https://www.nature.com",
			"https://www.science.org",
			"https://www ieee.org",
			"https://dl.acm.org",
			"https://www.springer.com",
			"https://onlinelibrary.wiley.com",
			"https://www.ncbi.nlm.nih.gov",
			"https://www.semanticscholar.org",
			"https://www.doaj.org",
			"https://www.coursera.org",
			"https://www.edx.org",
			"https://www mit.edu",
			"https://www.stanford.edu",
		},
	},
}

// GetSeedSet returns a predefined seed set by type.
func GetSeedSet(setType SeedSetType) (*SeedSet, error) {
	set, ok := predefinedSeedSets[setType]
	if !ok {
		return nil, fmt.Errorf("unknown seed set type: %s (valid: general, programming, academic)", setType)
	}
	return set, nil
}

// ListSeedSets returns information about all available seed sets.
func ListSeedSets() []*SeedSet {
	sets := make([]*SeedSet, 0, len(predefinedSeedSets))
	for _, set := range predefinedSeedSets {
		sets = append(sets, set)
	}
	return sets
}

// LoadSeedConfig loads seed sets from a configuration file.
func LoadSeedConfig(path string) (*SeedConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open seed config file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("failed to close seed config file: %v\n", closeErr)
		}
	}()

	config := &SeedConfig{
		Sets: make(map[string]*SeedSet),
	}

	scanner := bufio.NewScanner(file)
	currentSet := &SeedSet{}
	inSet := false
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse set directives: [set_name]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// Save previous set if exists
			if inSet && len(currentSet.URLs) > 0 {
				config.Sets[currentSet.Name] = currentSet
			}

			setName := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			currentSet = &SeedSet{
				Name: setName,
				URLs: []string{},
			}
			inSet = true
			continue
		}

		// Parse URL lines
		if inSet && !strings.HasPrefix(line, "#") {
			url := strings.TrimSpace(line)
			if url != "" {
				currentSet.URLs = append(currentSet.URLs, url)
			}
		}

		// Parse default directive
		if strings.HasPrefix(strings.ToLower(line), "default=") {
			defaultSet := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "default="))
			config.Default = defaultSet
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading seed config: %w", err)
	}

	// Save last set
	if inSet && len(currentSet.URLs) > 0 {
		config.Sets[currentSet.Name] = currentSet
	}

	if len(config.Sets) == 0 {
		return nil, fmt.Errorf("no seed sets found in config file")
	}

	return config, nil
}

// GetSeedFromConfig retrieves a specific seed set from configuration.
func GetSeedFromConfig(config *SeedConfig, name string) (*SeedSet, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	set, ok := config.Sets[name]
	if !ok {
		available := make([]string, 0, len(config.Sets))
		for setName := range config.Sets {
			available = append(available, setName)
		}
		return nil, fmt.Errorf("seed set '%s' not found (available: %s)", name, strings.Join(available, ", "))
	}

	return set, nil
}

// GetDefaultSeedFromConfig returns the default seed set from configuration.
func GetDefaultSeedFromConfig(config *SeedConfig) (*SeedSet, error) {
	if config == nil || config.Default == "" {
		return nil, fmt.Errorf("no default seed set configured")
	}
	return GetSeedFromConfig(config, config.Default)
}

// MergeSeeds combines predefined and custom seed URLs.
func MergeSeeds(predefined *SeedSet, customURLs []string) []string {
	if predefined == nil {
		return customURLs
	}

	urls := make([]string, 0, len(predefined.URLs)+len(customURLs))
	urls = append(urls, predefined.URLs...)
	urls = append(urls, customURLs...)

	return urls
}

// ValidateSeedURL checks if a URL is valid for seeding.
func ValidateSeedURL(url string) error {
	if url == "" {
		return fmt.Errorf("seed URL cannot be empty")
	}

	url = strings.TrimSpace(url)

	// Basic URL validation
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("seed URL must start with http:// or https://: %s", url)
	}

	return nil
}
