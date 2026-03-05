package crawler

import (
	"net/url"
	"sort"
	"testing"
)

func TestExtractEmbeddedLinks_JSONTargets(t *testing.T) {
	baseURL, err := url.Parse("https://www.sephora.com/")
	if err != nil {
		t.Fatalf("failed to parse base URL: %v", err)
	}

	raw := `
	<script>
		window.Sephora = {
			items: [
				{"targetUrl":"/product/lip-gloss-P123?skuId=1"},
				{"targetUrl":"\/shop/fragrance-value-sets-gifts"},
				{"targetUrl":"https://www.sephora.com/beauty/best-sellers"},
				{"image":"\/contentimages/banner.jpg"},
				{"chunk":"\/js/app.chunk.js"}
			]
		}
	</script>
	`

	got := extractEmbeddedLinks(raw, baseURL)
	sort.Strings(got)

	want := []string{
		"https://www.sephora.com/beauty/best-sellers",
		"https://www.sephora.com/product/lip-gloss-P123?skuId=1",
		"https://www.sephora.com/shop/fragrance-value-sets-gifts",
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("unexpected number of links: got %d want %d\nlinks=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected link at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestExtractEmbeddedLinks_DeduplicatesAndFiltersAssets(t *testing.T) {
	baseURL, err := url.Parse("https://example.com/")
	if err != nil {
		t.Fatalf("failed to parse base URL: %v", err)
	}

	raw := `
	<script>
		const data = {
			a: "/shop/all",
			b: "/shop/all",
			c: "/images/logo.svg",
			d: "/productimages/sku/x-main-zoom.jpg",
			e: "//example.com/beauty/new-arrivals",
			f: "mailto:test@example.com"
		};
	</script>
	`

	got := extractEmbeddedLinks(raw, baseURL)
	sort.Strings(got)

	want := []string{
		"https://example.com/beauty/new-arrivals",
		"https://example.com/shop/all",
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("unexpected number of links: got %d want %d\nlinks=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected link at %d: got %q want %q", i, got[i], want[i])
		}
	}
}
