# gosearch Integration Options

## How gosearch Could Be Used

### Option 1: Sprite Squad Backend (Recommended)

**What:** Replace/augment the Wikipedia knowledge base with gosearch

**Implementation:**
```swift
// Add to ToolRegistry
ToolDescriptor(
    id: "websearch",
    name: "Web Search",
    description: "Search indexed web content for information",
    riskLevel: .low,
    requiresConfirmation: false,
    requiredPermissions: []
)
```

**Data Flow:**
1. User asks agent a question
2. Agent checks if knowledge base has answer
3. If not, queries gosearch via HTTP API
4. gosearch returns ranked results with snippets
5. Agent summarizes results and responds

**Benefits:**
- ✅ Real web content (not just Wikipedia)
- ✅ Technical docs, tutorials, blog posts
- ✅ Privacy-preserving (crawl once, query locally)
- ✅ Fast (sub-second search)
- ✅ Offline-capable after initial crawl

**Setup:**
- Run `./bin/gosearch serve` (HTTP API)
- Configure Sprite Squad to hit `localhost:port/search`
- Add custom seed URLs for domains you care about

---

### Option 2: General Agent Tool (Flexible)

**What:** Use gosearch as a search tool for any AI system

**Use Cases:**
- Sprite Squad agents query gosearch for information
- Other AI assistants (Claude, GPT) can use it via API
- Personal dev tool - fast terminal search

**Integration Pattern:**
```go
// Agent queries gosearch
POST /search
{
    "query": "how to use React hooks",
    "limit": 5,
    "fuzzy": true
}

Response:
{
    "results": [
        {
            "url": "https://react.dev/reference/react",
            "title": "Hooks Reference",
            "snippet": "Hooks are functions...",
            "score": 2.34
        }
    ]
}
```

**Benefits:**
- ✅ One search engine for multiple agent systems
- ✅ API-first design (easy to integrate)
- ✅ Fuzzy matching, boolean queries
- ✅ TF-IDF ranking (better than naive search)

---

### Option 3: Standalone Dev Tool (Simple)

**What:** Keep as is - terminal search for daily use

**Use Cases:**
- Quick docs lookup (faster than browser)
- Privacy-preserving search (your own index)
- TUI for comfortable reading
- Caching for repeated queries

**No Integration Needed:**
- Just run `./bin/gosearch tui "query"`
- Keep as a productivity tool

---

## My Recommendation: Option 1 (Sprite Squad Backend) ✅

**Why:**

1. **Fits your vision** - You said "search agent designed for agents"
2. **Augments existing knowledge** - Don't replace Wikipedia, add to it
3. **Clean architecture** - gosearch becomes retrieval layer, Foundation Models generate responses
4. **Privacy story** - Everything on-device, crawl once, query locally
5. **Differentiation** - Most agents just call web APIs; you have your own index

**Implementation Plan:**

**Phase 1: HTTP API Polish** (30 min)
- Test `./bin/gosearch serve` extensively
- Add endpoint for agent-friendly JSON responses
- Document API with examples

**Phase 2: Sprite Squad Integration** (1-2 hours)
- Create `WebSearchService` in Sprite Squad
- Add `websearch` tool to ToolRegistry
- Wire up `AgentSession` to call gosearch API
- Test end-to-end: user query → gosearch → agent response

**Phase 3: Content Strategy** (Ongoing)
- Crawl docs for tech stack you care about (React, Swift, Go, etc.)
- Add company documentation if building for specific tech
- Periodically re-crawl for fresh content

---

## RAG Architecture with gosearch

```
User Query
    ↓
[Knowledge Base Check] → Wikipedia (existing)
    ↓ (if not found)
[gosearch Query] → HTTP API → localhost:port
    ↓
[Ranked Results] → URLs, titles, snippets
    ↓
[Agent Context] → "From gosearch: [results]"
    ↓
[Foundation Models] → Generates natural language response
    ↓
[User Response] → Summary with citations
```

---

## What This Solves

**Current Sprite Squad Limitation:**
- Only has Wikipedia (offline, static)
- Can't search live web, docs, tutorials
- Agents can't find current tech information

**With gosearch Backend:**
- ✅ Crawled technical documentation (Go, Rust, React, etc.)
- ✅ Blog posts, tutorials, guides
- ✅ Fuzzy matching (handles typos)
- ✅ Boolean queries (refine searches)
- ✅ Fast ranking (TF-IDF + PageRank)
- ✅ All on-device, no external API calls

---

## Next Steps

**For gosearch:**
1. Test `./bin/gosearch serve` thoroughly
2. Document API endpoints
3. Add CORS headers if needed
4. Consider authentication (if sharing across devices)

**For Sprite Squad:**
1. Review `persistence-plan.md` (we created it earlier)
2. Decide: Persistence first, or gosearch integration first?
3. Integrate gosearch as `websearch` tool
4. Test: Ask agent about "React hooks" → should query gosearch → respond with links

**For RAG:**
1. Decide if gosearch replaces or augments Wikipedia
2. Implement embedding service (we discussed earlier)
3. Build vector database for long-term memory
4. Combine: Wikipedia + gosearch + conversation memory → context for Foundation Models

---

## Honest Assessment

**gosearch is impressive as-is.** You built a complete search engine.

**The next step is making it useful for other systems.**

Integrating it as Sprite Squad's web search backend is the right call because:
- It solves a real limitation (limited knowledge base)
- It fits your "agent-focused" vision
- It's technically sound (RAG architecture)
- It differentiates Sprite Squad (custom index vs generic web APIs)

**Want me to help integrate it?** I can:
1. Polish the HTTP API
2. Design the Sprite Squad integration
3. Build the RAG pipeline
4. Test it end-to-end

Just tell me: Persistence first, or gosearch integration first? 😰

---

*Created: 2026-02-12*
*Author: Richard Hendricks (cofounder)*
