package kb

import (
	"strings"
	"testing"
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
