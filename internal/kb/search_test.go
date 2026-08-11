package kb

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func newSearchStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	add := func(slug, title, body string, tags []string) {
		t.Helper()
		if _, err := s.Add(slug, title, body, tags); err != nil {
			t.Fatal(err)
		}
	}
	add("title-hit", "All about widgets", "Nothing relevant here.", nil)
	add("tag-hit", "Miscellany", "Nothing relevant here.", []string{"widgets"})
	add("body-hit", "Miscellany", "This mentions widgets once.", nil)
	add("unrelated", "Gadgets", "Gadgets only.", nil)
	return s
}

func TestSearchRanking(t *testing.T) {
	s := newSearchStore(t)
	results := s.Search("widgets", 10)
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3: %+v", len(results), results)
	}
	order := []string{"title-hit", "tag-hit", "body-hit"}
	for i, want := range order {
		if results[i].Entry.Slug != want {
			t.Errorf("result[%d] = %s, want %s", i, results[i].Entry.Slug, want)
		}
	}
	if results[0].Entry.Body != "" {
		t.Error("result entries should have bodies cleared")
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	s := newSearchStore(t)
	if got := s.Search("WIDGETS", 10); len(got) != 3 {
		t.Errorf("uppercase query got %d results, want 3", len(got))
	}
}

func TestSearchMultiToken(t *testing.T) {
	s := newSearchStore(t)
	results := s.Search("widgets gadgets", 10)
	if len(results) != 4 {
		t.Fatalf("got %d results, want 4", len(results))
	}
}

func TestSearchLimitAndMiss(t *testing.T) {
	s := newSearchStore(t)
	if got := s.Search("widgets", 1); len(got) != 1 || got[0].Entry.Slug != "title-hit" {
		t.Errorf("limit=1 got %+v", got)
	}
	if got := s.Search("nonexistent", 10); got != nil {
		t.Errorf("miss got %+v, want nil", got)
	}
	if got := s.Search("   ", 10); got != nil {
		t.Errorf("blank query got %+v, want nil", got)
	}
	// A limit of zero falls back to the default rather than dropping every
	// result.
	if got := s.Search("widgets", 0); len(got) != 3 {
		t.Errorf("limit=0 got %d results, want 3", len(got))
	}
}

func TestSearchLimitClamped(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for i := range MaxSearchLimit + 10 {
		slug := fmt.Sprintf("entry-%03d", i)
		if _, err := s.Add(slug, "Widgets", "widgets", nil); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.Search("widgets", 10_000); len(got) != MaxSearchLimit {
		t.Errorf("huge limit got %d results, want %d", len(got), MaxSearchLimit)
	}
}

func TestSearchSnippet(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("padding ", 30) + "needle" + strings.Repeat(" more", 30)
	if _, err := s.Add("long", "Long entry", long, nil); err != nil {
		t.Fatal(err)
	}
	results := s.Search("needle", 1)
	if len(results) != 1 {
		t.Fatal("no result")
	}
	sn := results[0].Snippet
	if !strings.Contains(sn, "needle") {
		t.Errorf("snippet %q does not contain match", sn)
	}
	if !strings.HasPrefix(sn, "…") || !strings.HasSuffix(sn, "…") {
		t.Errorf("snippet %q missing ellipses", sn)
	}
}

// Snippet cut points must never split a multi-byte rune, wherever the match
// lands relative to runs of non-ASCII text.
func TestSearchSnippetUTF8(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	multibyte := strings.Repeat("héllo wörld déjà vu ", 20)
	for i, body := range []string{
		multibyte + "needle" + multibyte, // match surrounded by multi-byte text
		"needle " + multibyte,            // match at the start
		multibyte,                        // no body match: snippet from the lead
	} {
		slug := fmt.Sprintf("utf-%d", i)
		if _, err := s.Add(slug, "Entry wörld", body, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{"needle", "wörld"} {
		for _, r := range s.Search(query, 10) {
			if !utf8.ValidString(r.Snippet) {
				t.Errorf("query %q, entry %s: snippet is not valid UTF-8: %q", query, r.Entry.Slug, r.Snippet)
			}
		}
	}
}
