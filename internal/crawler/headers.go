// Package crawler provides web crawling functionality for gosearch.
package crawler

import (
	"net/http"
)

// HeaderProfiles returns a map of browser header profiles.
func HeaderProfiles() map[string]HeaderProfile {
	return map[string]HeaderProfile{
		"chrome": {
			Name:            "Chrome on Windows",
			UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
			Accept:          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
			AcceptLanguage:  "en-US,en;q=0.9",
			AcceptEncoding:  "gzip, deflate, br",
			SecCHUA:         `"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"`,
			SecCHUAMobile:   "?0",
			SecCHUAPlatform: `"Windows"`,
			SecFetchDest:    "document",
			SecFetchMode:    "navigate",
			SecFetchSite:    "none",
			SecFetchUser:    "?1",
			UpgradeInsecure: "1",
		},
		"firefox": {
			Name:           "Firefox on Windows",
			UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
			Accept:         "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
			AcceptLanguage: "en-US,en;q=0.5",
			AcceptEncoding: "gzip, deflate, br",
			SecFetchDest:   "document",
			SecFetchMode:   "navigate",
			SecFetchSite:   "none",
			SecFetchUser:   "?1",
		},
		"safari": {
			Name:           "Safari on macOS",
			UserAgent:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15",
			Accept:         "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			AcceptLanguage: "en-US,en;q=0.9",
			AcceptEncoding: "gzip, deflate, br",
			SecFetchDest:   "document",
			SecFetchMode:   "navigate",
			SecFetchSite:   "none",
			SecFetchUser:   "?1",
		},
		"edge": {
			Name:            "Edge on Windows",
			UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0",
			Accept:          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
			AcceptLanguage:  "en-US,en;q=0.9",
			AcceptEncoding:  "gzip, deflate, br",
			SecCHUA:         `"Chromium";v="122", "Not(A:Brand";v="24", "Microsoft Edge";v="122"`,
			SecCHUAMobile:   "?0",
			SecCHUAPlatform: `"Windows"`,
			SecFetchDest:    "document",
			SecFetchMode:    "navigate",
			SecFetchSite:    "none",
			SecFetchUser:    "?1",
			UpgradeInsecure: "1",
		},
	}
}

// GetHeaderProfile returns a header profile by name.
// Returns the Chrome profile if the name is not found.
func GetHeaderProfile(name string) HeaderProfile {
	profiles := HeaderProfiles()
	if profile, exists := profiles[name]; exists {
		return profile
	}
	return profiles["chrome"]
}

// ApplyHeaders applies a header profile to an HTTP request.
func ApplyHeaders(req *http.Request, profile HeaderProfile) {
	req.Header.Set("User-Agent", profile.UserAgent)
	req.Header.Set("Accept", profile.Accept)
	req.Header.Set("Accept-Language", profile.AcceptLanguage)
	req.Header.Set("Accept-Encoding", profile.AcceptEncoding)

	// Sec-CH-UA headers (Client Hints)
	if profile.SecCHUA != "" {
		req.Header.Set("Sec-CH-UA", profile.SecCHUA)
	}
	if profile.SecCHUAMobile != "" {
		req.Header.Set("Sec-CH-UA-Mobile", profile.SecCHUAMobile)
	}
	if profile.SecCHUAPlatform != "" {
		req.Header.Set("Sec-CH-UA-Platform", profile.SecCHUAPlatform)
	}

	// Sec-Fetch headers
	if profile.SecFetchDest != "" {
		req.Header.Set("Sec-Fetch-Dest", profile.SecFetchDest)
	}
	if profile.SecFetchMode != "" {
		req.Header.Set("Sec-Fetch-Mode", profile.SecFetchMode)
	}
	if profile.SecFetchSite != "" {
		req.Header.Set("Sec-Fetch-Site", profile.SecFetchSite)
	}
	if profile.SecFetchUser != "" {
		req.Header.Set("Sec-Fetch-User", profile.SecFetchUser)
	}

	// Upgrade header
	if profile.UpgradeInsecure != "" {
		req.Header.Set("Upgrade-Insecure-Requests", profile.UpgradeInsecure)
	}

	// Additional headers for realism
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
}
