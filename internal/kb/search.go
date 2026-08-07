package kb

import (
	"slices"
	"strings"
)

// SearchResult pairs an entry (body omitted) with its relevance score and a
// snippet of the first body match.
type SearchResult struct {
	Entry   Entry  `json:"entry"`
	Score   int    `json:"score"`
	Snippet string `json:"snippet"`
}

const (
	titleWeight = 10
	tagWeight   = 5
	bodyWeight  = 1

	snippetRadius = 60
	snippetLead   = 120
)

// Search performs case-insensitive keyword search over titles, tags, and
// bodies. The query is split on whitespace; each token scores title
// occurrences highest, then exact tag matches, then body occurrences.
// Entries with zero score are excluded and results are capped at limit.
func (s *Store) Search(query string, limit int) []SearchResult {
	tokens := strings.Fields(strings.ToLower(query))
	if len(tokens) == 0 || limit <= 0 {
		return nil
	}
	var results []SearchResult
	for _, e := range s.List() {
		title := strings.ToLower(e.Title)
		body := strings.ToLower(e.Body)
		score := 0
		firstMatch := -1
		for _, tok := range tokens {
			score += titleWeight * strings.Count(title, tok)
			for _, tag := range e.Tags {
				if strings.ToLower(tag) == tok {
					score += tagWeight
				}
			}
			score += bodyWeight * strings.Count(body, tok)
			if i := strings.Index(body, tok); i >= 0 && (firstMatch < 0 || i < firstMatch) {
				firstMatch = i
			}
		}
		if score == 0 {
			continue
		}
		snippet := makeSnippet(e.Body, firstMatch)
		e.Body = ""
		results = append(results, SearchResult{Entry: e, Score: score, Snippet: snippet})
	}
	slices.SortFunc(results, func(a, b SearchResult) int {
		if a.Score != b.Score {
			return b.Score - a.Score
		}
		return strings.Compare(a.Entry.Slug, b.Entry.Slug)
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

// makeSnippet excerpts the body around the first match position, or the
// start of the body when the match was not in the body.
func makeSnippet(body string, match int) string {
	start, end := 0, min(len(body), snippetLead)
	if match >= 0 {
		start = max(0, match-snippetRadius)
		end = min(len(body), match+snippetRadius)
	}
	snippet := strings.TrimSpace(body[start:end])
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(body) {
		snippet += "…"
	}
	return snippet
}
