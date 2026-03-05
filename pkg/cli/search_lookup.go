package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/abuiliazeed/gosearch/internal/search"
	"github.com/abuiliazeed/gosearch/internal/storage"
)

type rankedSearchDocument struct {
	Query        string
	Rank         int
	TotalResults int
	Result       *search.Result
	Document     *storage.Document
}

func fetchRankedSearchDocument(query string, rankPos int, noCache bool) (*rankedSearchDocument, error) {
	if rankPos < 1 {
		return nil, fmt.Errorf("--rank must be >= 1")
	}

	runtime, err := newSearchRuntime(noCache)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := runtime.Search(ctx, query, rankPos)
	if err != nil {
		return nil, err
	}

	if response.TotalCount == 0 {
		return nil, fmt.Errorf("no results found for query %q", query)
	}

	if rankPos > len(response.Results) {
		return nil, fmt.Errorf("requested rank %d, but only %d results are available", rankPos, len(response.Results))
	}

	selected := response.Results[rankPos-1]

	doc, err := runtime.GetDocument(selected.DocID)
	if err != nil {
		return nil, fmt.Errorf("failed to load stored page for doc %s: %w", selected.DocID, err)
	}

	return &rankedSearchDocument{
		Query:        query,
		Rank:         rankPos,
		TotalResults: response.TotalCount,
		Result:       selected,
		Document:     doc,
	}, nil
}
