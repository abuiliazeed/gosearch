# Seeding Strategies for gosearch

> This document researches and outlines seeding strategies for web crawlers, with specific recommendations for the gosearch project.

---

## What is a Seed URL?

A **seed URL** is the starting point for a web crawler to begin discovering and indexing web content. Crawlers use seed URLs as initial addresses to visit, then follow links to discover new pages. The quality and diversity of seed URLs directly impacts:

- **Coverage** - How much of the web is reachable
- **Efficiency** - How quickly high-quality pages are found
- **Relevance** - How well the crawled content matches the target domain

---

## Core Seeding Strategies

### 1. High Hub/Authority Pages

Select seed URLs with high **hub scores** (pages that link to many authoritative pages) and **authority scores** (pages linked by many hubs).

**Rationale:** Pages with high hub scores connect to many other web pages, assuring higher recall and precision for focused crawlers. High-authority pages serve as quality indicators.

**Implementation:**
```go
// Example: Prioritize domains with high hub scores
type HubScore struct {
    Domain  string
    Score   float64
    Links   int
}

// Priority: Wikipedia, major news sites, university domains
highHubSeeds := []string{
    "https://en.wikipedia.org/",
    "https://news.ycombinator.com/",
    "https://reddit.com/",
    "https://stackoverflow.com/",
}
```

**Sources:**
- [A Survey on Focused Crawler for Seed URL Selection](https://www.ijaict.com/journals/ijaict/ijaict_pdf/2017_volume04/2017_v4i8/ijaict%25202014120807.pdf)
- [Focused Crawling Based Upon Tf-Idf Semantics and Hub Score](http://www.jetwi.us/uploadfile/2014/1219/20141219053135595.pdf)

### 2. Domain Diversity Strategy

Distribute seed URLs across different top-level domains (TLDs), geographic regions, and content types to ensure broad coverage.

**Rationale:** Domain diversity prevents the crawler from being trapped in a "island" of interconnected pages and ensures representation of different web communities.

**Implementation:**
```go
// Diverse seed set covering multiple dimensions
diverseSeeds := []string{
    // Reference/Knowledge
    "https://en.wikipedia.org/",
    "https://www.britannica.com/",

    // News (multiple regions)
    "https://www.bbc.com/",
    "https://www.nytimes.com/",
    "https://www.aljazeera.com/",

    // Tech/Developer
    "https://github.com/",
    "https://stackoverflow.com/",
    "https://dev.to/",

    // Academic
    "https://scholar.google.com/",
    "https://arxiv.org/",

    // Social/Forums
    "https://reddit.com/",
    "https://news.ycombinator.com/",
}
```

### 3. PageRank-Based Selection

Select seed URLs based on PageRank scores, prioritizing pages that are frequently cited across the web.

**Rationale:** Pages with high PageRank are generally more important and link to other important pages, creating a cascade effect for discovering quality content.

**Implementation Strategy:**
- Use existing PageRank data (if available) to score potential seeds
- Prioritize domains with known high PageRank
- Combine with link graph analysis for new domains

### 4. Community-Based Seed Generation

Use community detection algorithms to identify clusters of highly interconnected pages and select representative seeds from each community.

**Rationale:** Bipartite cores have high hub or authority ranks. Pages with high hub-rank typically link to pages with high PageRank. This ensures coverage of distinct web communities.

**Implementation:**
1. Build link graph from existing crawl data
2. Detect communities using algorithms like Louvain or Infomap
3. Select top pages (by centrality) from each community as seeds

**Sources:**
- [A Fast Community Based Algorithm for Generating Web Crawler Seeds Set](https://www.researchgate.net/publication/220724572_A_Fast_Community_Based_Algorithm_for_Generating_Web_Crawler_Seeds_Set)
- [A fast community based algorithm for generating web crawler seeds (PDF)](https://scispace.com/pdf/a-fast-community-based-algorithm-for-generating-web-crawler-3qyoqupacw.pdf)

### 5. Topic-Driven Focused Crawling

For topic-specific search engines, select seed URLs that are highly relevant to the target topic using TF-IDF semantics and topic similarity.

**Rationale:** Focused crawlers selectively seek out pages that are relevant to a pre-defined set of topics. Seed selection is critical for recall and precision.

**Implementation:**
```go
// Topic-focused seed selection
type TopicSeed struct {
    URL        string
    Relevance  float64  // TF-IDF similarity score
    Category   string
}

// For programming-focused search, start with developer sites
programmingSeeds := []string{
    "https://github.com/trending",
    "https://stackoverflow.com/questions",
    "https://dev.to/trending",
    "https://www.reddit.com/r/programming/",
}
```

**Sources:**
- [Focused Crawling: A New Approach to Topic-Specific Web Resource Discovery](https://www.ra.ethz.ch/cdstore/www8/data/2178/pdf/pd1.pdf)
- [A Survey of Focused Web Crawling Algorithms](https://aile3.ijs.si/dunja/SiKDD2004/Papers/BlazNovak-FocusedCrawling.pdf)

### 6. Search Engine Results as Seeds

Use search engine results for specific queries as seed URLs to discover relevant pages.

**Rationale:** Search engines have already done the work of finding relevant, high-quality pages for given queries. This can jump-start a focused crawl.

**Considerations:**
- May violate some search engines' Terms of Service
- Can be rate-limited or blocked
- Best for research/personal projects, not production

---

## Crawling Strategies

### Breadth-First Crawling (Recommended)

BFS explores all pages at the same depth before diving deeper. This is generally preferred for web crawling because:

- **Balanced Exploration:** Explores the web uniformly
- **High-Quality Pages:** Research shows BFS yields higher average page quality
- **Complete Coverage:** Reduces risk of missing important branches

**Sources:**
- [Breadth-first crawling yields high-quality pages (ACM)](https://dl.acm.org/doi/pdf/10.1145/371920.371965)
- [Measuring the Search Effectiveness of a Breadth-First Crawl](https://www.microsoft.com/en-us/research/wp-content/uploads/2009/04/p388-fetterly.pdf)
- [Understanding DFS vs BFS in Web Crawling](https://medium.com/@seelylook95/understanding-dfs-vs-bfs-in-web-crawling-a-practical-perspective-8129c93bfb02)

### Best-First Crawling

Prioritize URLs based on relevance scores (combining PageRank and topic similarity). Best for focused crawls where relevance matters more than coverage.

---

## Recommended Seed Sets for gosearch

### General Web Search Seed Set

```go
var generalSeeds = []string{
    // Knowledge bases (high authority)
    "https://en.wikipedia.org/wiki/Main_Page",
    "https://www.wikidata.org/",

    // Major news sources (fresh content, diverse links)
    "https://www.bbc.com/news",
    "https://www.reuters.com/",
    "https://apnews.com/",

    // Tech hubs (developer content)
    "https://news.ycombinator.com/",
    "https://slashdot.org/",
    "https://techcrunch.com/",

    // Reference sites
    "https://www.reddit.com/",
    "https://stackoverflow.com/",

    // Open directory alternative
    "https://dmoztools.net/",
}
```

### Programming-Focused Seed Set

```go
var programmingSeeds = []string{
    "https://github.com/explore",
    "https://stackoverflow.com/questions",
    "https://dev.to/",
    "https://medium.com/tag/programming",
    "https://www.reddit.com/r/programming/",
    "https://www.reddit.com/r/golang/",
    "https://www.reddit.com/r/rust/",
    "https://www.reddit.com/r/python/",
}
```

### Academic/Research Seed Set

```go
var academicSeeds = []string{
    "https://arxiv.org/",
    "https://scholar.google.com/",
    "https://www.researchgate.net/",
    "https://www.semanticscholar.org/",
    "https://doaj.org/",  // Directory of Open Access Journals
}
```

---

## Implementation in gosearch

### Adding Seed Configuration

Add seed sets to `internal/crawler/seeds.go`:

```go
package crawler

// SeedSet represents a collection of seed URLs
type SeedSet struct {
    Name   string
    Seeds  []string
    Topics []string  // Topics for focused crawling
}

// PredefinedSeedSets returns available seed collections
func PredefinedSeedSets() map[string]*SeedSet {
    return map[string]*SeedSet{
        "general": {
            Name: "General Web",
            Seeds: generalSeeds,
        },
        "programming": {
            Name: "Programming",
            Seeds: programmingSeeds,
            Topics: []string{"programming", "software", "development"},
        },
        "academic": {
            Name: "Academic Research",
            Seeds: academicSeeds,
            Topics: []string{"research", "academic", "science"},
        },
    }
}
```

### CLI Enhancement

Add seed selection to crawl command:

```go
// In pkg/cli/crawl.go
var crawlCmd = &cobra.Command{
    // ...
    RunE: func(cmd *cobra.Command, args []string) error {
        // If no URLs provided, use seed set
        if len(args) == 0 {
            seedSet, _ := cmd.Flags().GetString("seed-set")
            seeds := crawler.PredefinedSeedSets()[seedSet]
            args = seeds.Seeds
        }
        // ... rest of crawl logic
    },
}

func init() {
    rootCmd.AddCommand(crawlCmd)

    crawlCmd.Flags().StringP("seed-set", "s", "general",
        "predefined seed set: general, programming, academic")
}
```

---

## Best Practices Summary

1. **Start Small, Expand Gradually:** Begin with 10-20 high-quality seeds, expand based on results
2. **Prioritize Domain Diversity:** Ensure seeds cover multiple TLDs and regions
3. **Use High-Hub Pages:** Wikipedia, Reddit, and major news sites link to diverse content
4. **Monitor Crawl Health:** Track coverage, avoid getting trapped in link farms
5. **Refresh Seeds Periodically:** Update seed lists as web evolves
6. **Combine Strategies:** Use multiple seed sources for comprehensive coverage
7. **Respect robots.txt:** Always check and respect robots.txt directives
8. **Implement Politeness:** Rate limiting and respectful crawling behavior

---

## References

1. [What is a seed URL? | Firecrawl Glossary](https://www.firecrawl.dev/glossary/web-crawling-apis/what-is-seed-url)
2. [How do web crawlers decide which pages to access and index? | Tencent Cloud](https://www.tencentcloud.com/techpedia/131313)
3. [A Survey on Focused Crawler for Seed URL Selection](https://www.ijaict.com/journals/ijaict/ijaict_pdf/2017_volume04/2017_v4i8/ijaict%25202014120807.pdf)
4. [Design A Web Crawler - ByteByteGo](https://bytebytego.com/courses/system-design-interview/design-a-web-crawler)
5. [Google's search engine web crawlers are not what you think | Segment SEO](https://www.segmentseo.com/blog/crawlers)
6. [Graph-based seed selection for web-scale crawlers | C. Lee Giles](https://clgiles.ist.psu.edu/pubs/CIKM2009-crawler-seeds.pdf)
7. [URL Seeding - Crawl4AI Documentation](https://docs.crawl4ai.com/core/url-seeding/)
8. [Focused Crawling: A New Approach to Topic-Specific Web Resource Discovery](https://www.ra.ethz.ch/cdstore/www8/data/2178/pdf/pd1.pdf)
9. [Breadth-first crawling yields high-quality pages | ACM Digital Library](https://dl.acm.org/doi/pdf/10.1145/371920.371965)
10. [Measuring the Search Effectiveness of a Breadth-First Crawl | Microsoft Research](https://www.microsoft.com/en-us/research/wp-content/uploads/2009/04/p388-fetterly.pdf)

---

*Document created: 2026-02-11*
*Last updated: 2026-02-11*
