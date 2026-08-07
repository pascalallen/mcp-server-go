package kb

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("kb: entry not found")
	ErrExists   = errors.New("kb: entry already exists")
)

// Store is a knowledge base backed by Markdown files in a single directory,
// with an in-memory index for reads and searches. All writes persist to disk
// before returning.
type Store struct {
	dir     string
	mu      sync.RWMutex
	entries map[string]Entry // keyed by slug
}

// Open loads every *.md file in dir into memory, creating dir if needed.
// Malformed files are skipped with a log message rather than failing startup.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("kb: create dir: %w", err)
	}
	s := &Store{dir: dir, entries: make(map[string]Entry)}
	paths, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("kb: scan dir: %w", err)
	}
	for _, p := range paths {
		slug := strings.TrimSuffix(filepath.Base(p), ".md")
		if !validSlug.MatchString(slug) {
			log.Printf("kb: skipping %s: filename is not a valid slug", p)
			continue
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("kb: read %s: %w", p, err)
		}
		e, err := parseEntry(slug, raw)
		if err != nil {
			log.Printf("kb: skipping %s: %v", p, err)
			continue
		}
		s.entries[slug] = e
	}
	return s, nil
}

func (s *Store) Get(slug string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[slug]
	return e, ok
}

// List returns all entries sorted by slug.
func (s *Store) List() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		out = append(out, e)
	}
	slices.SortFunc(out, func(a, b Entry) int { return strings.Compare(a.Slug, b.Slug) })
	return out
}

// Add creates a new entry and persists it. An empty slug is derived from the
// title via Slugify.
func (s *Store) Add(slug, title, body string, tags []string) (Entry, error) {
	if slug == "" {
		slug = Slugify(title)
	}
	if !validSlug.MatchString(slug) {
		return Entry{}, fmt.Errorf("kb: invalid slug %q: must match %s", slug, validSlug)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[slug]; ok {
		return Entry{}, fmt.Errorf("%w: %s", ErrExists, slug)
	}
	now := time.Now().UTC().Truncate(time.Second)
	e := Entry{Slug: slug, Title: title, Tags: tags, CreatedAt: now, UpdatedAt: now, Body: body}
	if err := s.persist(e); err != nil {
		return Entry{}, err
	}
	s.entries[slug] = e
	return e, nil
}

// Update holds partial changes for an entry. Nil fields keep the current
// value; a pointer to an empty slice clears the tags.
type Update struct {
	Title *string
	Body  *string
	Tags  *[]string
}

func (s *Store) Update(slug string, upd Update) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[slug]
	if !ok {
		return Entry{}, fmt.Errorf("%w: %s", ErrNotFound, slug)
	}
	if upd.Title != nil {
		e.Title = *upd.Title
	}
	if upd.Body != nil {
		e.Body = *upd.Body
	}
	if upd.Tags != nil {
		e.Tags = *upd.Tags
	}
	e.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if err := s.persist(e); err != nil {
		return Entry{}, err
	}
	s.entries[slug] = e
	return e, nil
}

func (s *Store) Delete(slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[slug]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, slug)
	}
	if err := os.Remove(s.path(slug)); err != nil {
		return fmt.Errorf("kb: delete %s: %w", slug, err)
	}
	delete(s.entries, slug)
	return nil
}

func (s *Store) path(slug string) string {
	return filepath.Join(s.dir, slug+".md")
}

// persist atomically writes an entry's file via a temp file and rename.
// Callers must hold the write lock.
func (s *Store) persist(e Entry) error {
	data, err := renderEntry(e)
	if err != nil {
		return fmt.Errorf("kb: render %s: %w", e.Slug, err)
	}
	tmp, err := os.CreateTemp(s.dir, e.Slug+".*.tmp")
	if err != nil {
		return fmt.Errorf("kb: write %s: %w", e.Slug, err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("kb: write %s: %w", e.Slug, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("kb: write %s: %w", e.Slug, err)
	}
	if err := os.Rename(tmp.Name(), s.path(e.Slug)); err != nil {
		return fmt.Errorf("kb: write %s: %w", e.Slug, err)
	}
	return nil
}
