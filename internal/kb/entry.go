package kb

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Entry is a single knowledge base document. The slug is the filename stem
// (data/kb/<slug>.md) and is never duplicated in the front matter.
type Entry struct {
	Slug      string    `json:"slug" yaml:"-"`
	Title     string    `json:"title" yaml:"title"`
	Tags      []string  `json:"tags,omitempty" yaml:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at" yaml:"created"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated"`
	Body      string    `json:"body,omitempty" yaml:"-"`
}

var validSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Slugify converts an arbitrary string into a slug: lowercase, with runs of
// non-alphanumeric characters collapsed into single hyphens. The result may
// be empty if the input contains no alphanumeric characters.
func Slugify(s string) string {
	var b strings.Builder
	prevHyphen := true // suppress a leading hyphen
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevHyphen = false
		case !prevHyphen:
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

const fence = "---"

// parseEntry decodes a Markdown file with YAML front matter.
func parseEntry(slug string, raw []byte) (Entry, error) {
	e := Entry{Slug: slug}
	content := string(raw)
	if !strings.HasPrefix(content, fence+"\n") {
		return e, fmt.Errorf("missing front matter fence")
	}
	rest := content[len(fence)+1:]
	head, body, found := strings.Cut(rest, "\n"+fence+"\n")
	if !found {
		return e, fmt.Errorf("unterminated front matter")
	}
	if err := yaml.Unmarshal([]byte(head), &e); err != nil {
		return e, fmt.Errorf("front matter: %w", err)
	}
	// Undo the normalization renderEntry applies (blank line after the
	// fence, trailing newline) so parse/render round-trips are stable.
	e.Body = strings.TrimSuffix(strings.TrimPrefix(body, "\n"), "\n")
	return e, nil
}

// renderEntry encodes an entry back into Markdown with YAML front matter.
func renderEntry(e Entry) ([]byte, error) {
	head, err := yaml.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("front matter: %w", err)
	}
	var b bytes.Buffer
	b.WriteString(fence + "\n")
	b.Write(head)
	b.WriteString(fence + "\n\n")
	b.WriteString(e.Body)
	if !strings.HasSuffix(e.Body, "\n") {
		b.WriteByte('\n')
	}
	return b.Bytes(), nil
}
