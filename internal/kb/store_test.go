package kb

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Getting Started", "getting-started"},
		{"  Hello,   World!  ", "hello-world"},
		{"already-a-slug", "already-a-slug"},
		{"MCP: Tools & Resources", "mcp-tools-resources"},
		{"héllo wörld", "h-llo-w-rld"},
		{"!!!", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := Slugify(tt.in); got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	e, err := s.Add("", "My First Entry: A Test", "Some **markdown** body.", []string{"test", "demo"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if e.Slug != "my-first-entry-a-test" {
		t.Errorf("derived slug = %q", e.Slug)
	}
	if _, err := os.Stat(filepath.Join(dir, e.Slug+".md")); err != nil {
		t.Errorf("entry file not on disk: %v", err)
	}

	if _, err := s.Add(e.Slug, "Dup", "x", nil); !errors.Is(err, ErrExists) {
		t.Errorf("duplicate Add error = %v, want ErrExists", err)
	}
	if _, err := s.Add("Bad Slug!", "t", "b", nil); err == nil {
		t.Error("Add with invalid slug should fail")
	}

	got, ok := s.Get(e.Slug)
	if !ok || got.Title != e.Title || got.Body != e.Body {
		t.Errorf("Get = %+v, ok=%v", got, ok)
	}

	newTitle := "Renamed"
	upd, err := s.Update(e.Slug, Update{Title: &newTitle, Tags: &[]string{}})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if upd.Title != "Renamed" || len(upd.Tags) != 0 || upd.Body != e.Body {
		t.Errorf("Update result = %+v", upd)
	}
	if _, err := s.Update("missing", Update{}); !errors.Is(err, ErrNotFound) {
		t.Errorf("Update missing error = %v, want ErrNotFound", err)
	}

	// Reopen to prove persistence round-trips, including the colon in the title.
	s2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got2, ok := s2.Get(e.Slug)
	if !ok || got2.Title != "Renamed" || got2.Body != e.Body || len(got2.Tags) != 0 {
		t.Errorf("after reopen: %+v, ok=%v", got2, ok)
	}
	if got2.CreatedAt.IsZero() || got2.UpdatedAt.Before(got2.CreatedAt) {
		t.Errorf("timestamps not persisted: %+v", got2)
	}

	if err := s2.Delete(e.Slug); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, e.Slug+".md")); !os.IsNotExist(err) {
		t.Error("entry file still on disk after Delete")
	}
	if err := s2.Delete(e.Slug); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete error = %v, want ErrNotFound", err)
	}
}

func TestOpenSkipsMalformed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.md"), []byte("no front matter"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Bad Name.md"), []byte("---\ntitle: x\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if n := len(s.List()); n != 0 {
		t.Errorf("List after malformed files = %d entries, want 0", n)
	}
}

func TestList(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"charlie", "alpha", "bravo"} {
		if _, err := s.Add(slug, slug, "body", nil); err != nil {
			t.Fatal(err)
		}
	}
	list := s.List()
	if len(list) != 3 || list[0].Slug != "alpha" || list[1].Slug != "bravo" || list[2].Slug != "charlie" {
		t.Errorf("List not sorted by slug: %+v", list)
	}
}
