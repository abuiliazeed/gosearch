package ranker

import (
	"context"
	"testing"

	"github.com/abuiliazeed/gosearch/internal/indexer"
)

func TestTFIDF_Score(t *testing.T) {
	// Setup in-memory index
	idx := indexer.NewIndex()
	tokenizer := indexer.NewTokenizer(indexer.DefaultTokenizerConfig())

	// Index documents
	docs := []*indexer.DocumentInput{
		{DocID: "1", Title: "A", Content: "apple banana"},
		{DocID: "2", Title: "B", Content: "banana cherry"},
		{DocID: "3", Title: "C", Content: "apple cherry date"},
	}

	ctx := context.Background()
	for _, doc := range docs {
		idx.IndexDocument(ctx, tokenizer, doc)
	}

	tfidf := NewTFIDF(idx.GetIndex())

	// Calculate scores for "apple"
	// Doc 1: tf=1, df=2, N=3 => idf = log(3/2) = 0.405
	// Doc 3: tf=1, df=2, N=3 => idf = 0.405
	// Doc 2: tf=0

	scores := tfidf.ScoreDocuments([]string{"apple"})

	if scores["1"] <= 0 {
		t.Error("expected positive score for doc 1")
	}
	if scores["2"] != 0 {
		t.Errorf("expected 0 score for doc 2, got %f", scores["2"])
	}

	// Calculate scores for "banana"
	// Doc 1: tf=1
	// Doc 2: tf=1
	scoresBanana := tfidf.ScoreDocuments([]string{"banana"})
	if scoresBanana["1"] <= 0 {
		t.Error("expected positive score for doc 1")
	}
}

func TestPageRank_Compute(t *testing.T) {
	pr := DefaultPageRank()
	graph := NewLinkGraph()

	// A -> B
	// B -> C
	// C -> A
	graph.AddLink("A", "B")
	graph.AddLink("B", "C")
	graph.AddLink("C", "A")

	ctx := context.Background()
	if err := pr.Compute(ctx, graph); err != nil {
		t.Fatalf("pagerank compute failed: %v", err)
	}

	// In a ring, all should have equal probability (approx 0.33)
	scoreA := pr.GetScore("A")
	scoreB := pr.GetScore("B")
	scoreC := pr.GetScore("C")

	if scoreA < 0.3 || scoreA > 0.4 {
		t.Errorf("expected score around 0.33, got %f", scoreA)
	}

	// Check relative equality
	if scoreA != scoreB || scoreB != scoreC { // Precision issues might make this fail, but for simple ring it might converge ideally
		// Let's use epsilon
		if abs(scoreA-scoreB) > 0.001 || abs(scoreB-scoreC) > 0.001 {
			t.Errorf("expected equal scores, got A=%f B=%f C=%f", scoreA, scoreB, scoreC)
		}
	}
}

func TestScorer_NavigationalQueryPrefersHomepage(t *testing.T) {
	idx := indexer.NewIndex()
	tokenizer := indexer.NewTokenizer(indexer.DefaultTokenizerConfig())

	docs := []*indexer.DocumentInput{
		{
			DocID:   "home",
			URL:     "https://examplesite.com/",
			Title:   "Example Site",
			Content: "examplesite official store",
		},
		{
			DocID:   "product",
			URL:     "https://examplesite.com/products/test-item",
			Title:   "Test Item - Example Site",
			Content: "examplesite test item",
		},
		{
			DocID:   "other",
			URL:     "https://other.com/examplesite-review",
			Title:   "ExampleSite Review",
			Content: "examplesite review",
		},
	}

	ctx := context.Background()
	for _, doc := range docs {
		if err := idx.IndexDocument(ctx, tokenizer, doc); err != nil {
			t.Fatalf("failed to index test document %s: %v", doc.DocID, err)
		}
	}

	tfidf := NewTFIDF(idx.GetIndex())
	pr := DefaultPageRank()
	scorer := NewScorer(tfidf, pr, nil)

	results := scorer.RankDocuments([]string{"examplesite"}, []string{"home", "product", "other"})
	if len(results) < 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].DocID != "home" {
		t.Fatalf("expected homepage to rank first for navigational query, got %s", results[0].DocID)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
