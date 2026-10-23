// Package genremap maps a programme's free-text categories to a single DVB
// content code using ordered, data-driven rule tables.
//
// A table is a JSON document describing one source vocabulary in one language
// (for example Schedules Direct in English). Rules are evaluated in order
// against the programme's whole category set and the first matching rule
// wins, so a table lists specific rules ("soccer" -> football) before generic
// ones ("sports event" -> sports).
//
// The mapper returns one code per programme on purpose: Tvheadend keeps a
// programme's genres sorted by code and clients such as Kodi display only the
// lowest one, so emitting several codes would let an incidental genre win.
package genremap

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/bernhardberger/xmltv-dvb-genres/etsi"
)

//go:embed tables/*.json
var embedded embed.FS

// Category is one category of a programme as found in the guide.
type Category struct {
	Text string
	Lang string // "" when the guide gives no language
}

// Rule maps a category set to a code. All, Any and None are optional but at
// least one of All or Any must be set; every condition given must hold.
type Rule struct {
	Code    string   `json:"code"`
	All     []string `json:"all,omitempty"`  // every term present
	Any     []string `json:"any,omitempty"`  // at least one term present
	None    []string `json:"none,omitempty"` // no term present
	Comment string   `json:"comment,omitempty"`

	code etsi.Code
}

// Table is one source vocabulary.
type Table struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Lang restricts the table to categories with this lang attribute;
	// categories without a lang attribute always match. Empty means any.
	Lang  string `json:"lang,omitempty"`
	Rules []Rule `json:"rules"`
	// Ignore lists terms known to carry no genre (e.g. "series"); they are
	// left out of unmapped reports.
	Ignore []string `json:"ignore,omitempty"`
}

// Normalize returns the canonical form of a category term: lower case,
// surrounding space trimmed and inner whitespace collapsed.
func Normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// Parse decodes and validates a table.
func Parse(data []byte) (*Table, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var t Table
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("genremap: decode table: %w", err)
	}
	if err := t.compile(); err != nil {
		return nil, err
	}
	return &t, nil
}

func (t *Table) compile() error {
	if t.Name == "" {
		return fmt.Errorf("genremap: table has no name")
	}
	if len(t.Rules) == 0 {
		return fmt.Errorf("genremap: table %s has no rules", t.Name)
	}
	var empty bool
	norm := func(terms []string) {
		for i := range terms {
			terms[i] = Normalize(terms[i])
			empty = empty || terms[i] == ""
		}
	}
	for i := range t.Rules {
		r := &t.Rules[i]
		c, err := etsi.Parse(r.Code)
		if err != nil {
			return fmt.Errorf("genremap: table %s rule %d: %w", t.Name, i+1, err)
		}
		r.code = c
		if len(r.All) == 0 && len(r.Any) == 0 {
			return fmt.Errorf("genremap: table %s rule %d: needs all or any", t.Name, i+1)
		}
		norm(r.All)
		norm(r.Any)
		norm(r.None)
		if empty {
			return fmt.Errorf("genremap: table %s rule %d: empty term", t.Name, i+1)
		}
	}
	norm(t.Ignore)
	if empty {
		return fmt.Errorf("genremap: table %s: empty term in ignore", t.Name)
	}
	t.Lang = strings.ToLower(strings.TrimSpace(t.Lang))
	return nil
}

// Tables returns the names of the embedded tables.
func Tables() []string {
	entries, _ := fs.ReadDir(embedded, "tables")
	var out []string
	for _, e := range entries {
		out = append(out, strings.TrimSuffix(e.Name(), ".json"))
	}
	sort.Strings(out)
	return out
}

// Load returns an embedded table by name, or reads a table file when ref
// contains a path separator or ends in ".json".
func Load(ref string) (*Table, error) {
	var (
		data []byte
		err  error
	)
	if strings.ContainsRune(ref, os.PathSeparator) || strings.HasSuffix(ref, ".json") {
		data, err = os.ReadFile(ref)
	} else {
		data, err = embedded.ReadFile(path.Join("tables", ref+".json"))
		if err != nil {
			err = fmt.Errorf("genremap: no embedded table %q (have %s)", ref, strings.Join(Tables(), ", "))
		}
	}
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Mapper evaluates one or more tables in order.
type Mapper struct {
	tables []*Table
}

// New returns a mapper that tries the tables in the given order, so a local
// override table can precede an embedded one.
func New(tables ...*Table) *Mapper { return &Mapper{tables: tables} }

// Result describes the outcome for one programme.
type Result struct {
	Code  etsi.Code
	OK    bool   // a rule matched
	Table string // table of the matching rule
	Rule  int    // 1-based index of the matching rule
}

// Map maps a programme's categories. Each table only sees categories in its
// language (or without a language).
func (m *Mapper) Map(cats []Category) Result {
	for _, t := range m.tables {
		set := make(map[string]bool, len(cats))
		for _, c := range cats {
			if t.Lang == "" || c.Lang == "" || strings.EqualFold(c.Lang, t.Lang) {
				set[Normalize(c.Text)] = true
			}
		}
		if len(set) == 0 {
			continue
		}
		for i := range t.Rules {
			if t.Rules[i].matches(set) {
				return Result{Code: t.Rules[i].code, OK: true, Table: t.Name, Rule: i + 1}
			}
		}
	}
	return Result{}
}

// MapTerms is Map for callers that hold plain terms in the tables' language,
// such as a grabber reading a provider API directly.
func (m *Mapper) MapTerms(terms ...string) Result {
	cats := make([]Category, len(terms))
	for i, t := range terms {
		cats[i] = Category{Text: t}
	}
	return m.Map(cats)
}

// Ignored reports whether any table lists term as carrying no genre.
func (m *Mapper) Ignored(term string) bool {
	term = Normalize(term)
	for _, t := range m.tables {
		for _, ig := range t.Ignore {
			if ig == term {
				return true
			}
		}
	}
	return false
}

func (r *Rule) matches(set map[string]bool) bool {
	for _, t := range r.All {
		if !set[t] {
			return false
		}
	}
	if len(r.Any) > 0 {
		found := false
		for _, t := range r.Any {
			if set[t] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, t := range r.None {
		if set[t] {
			return false
		}
	}
	return true
}
