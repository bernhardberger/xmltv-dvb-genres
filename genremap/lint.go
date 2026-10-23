package genremap

import (
	"fmt"
	"sort"
	"strings"
)

// Level is the severity of a lint issue.
type Level int

const (
	// Warning marks a mistake: the table loads but does not do what it says.
	Warning Level = iota
	// Info marks something worth a look, such as guide terms no rule covers.
	Info
)

func (l Level) String() string {
	if l == Warning {
		return "warning"
	}
	return "info"
}

// Issue is one lint finding. Rule is the 1-based rule index, or 0 when the
// issue concerns the table as a whole.
type Issue struct {
	Level Level
	Rule  int
	Msg   string
}

func (i Issue) String() string {
	if i.Rule > 0 {
		return fmt.Sprintf("%s: rule %d: %s", i.Level, i.Rule, i.Msg)
	}
	return fmt.Sprintf("%s: %s", i.Level, i.Msg)
}

// Lint checks a loaded table for rules that are valid but cannot work as
// written. When seen is non-nil it holds the category counts of a real guide
// (keys normalized, Lang lower case), and Lint also reports rule terms that
// never occur in it and guide terms that no rule or ignore entry covers.
func Lint(t *Table, seen map[Category]int) []Issue {
	var out []Issue
	warn := func(rule int, format string, a ...any) {
		out = append(out, Issue{Warning, rule, fmt.Sprintf(format, a...)})
	}

	// catchAll[term] = first rule that fires whenever term is present.
	catchAll := map[string]int{}
	used := map[string]bool{}
	for i, r := range t.Rules {
		n := i + 1
		if r.code < 0x10 || r.code > 0xAF {
			warn(n, "code %s has no Tvheadend name; clients will show no genre", r.code)
		}
		for _, list := range [][]string{r.All, r.Any, r.None} {
			dups := map[string]bool{}
			for _, term := range list {
				if dups[term] {
					warn(n, "term %q is listed more than once", term)
				}
				dups[term] = true
				used[term] = true
			}
		}
		none := toSet(r.None)
		for _, term := range r.All {
			if none[term] {
				warn(n, "term %q is in both all and none; the rule can never match", term)
			}
		}
		for _, term := range r.Any {
			if none[term] {
				warn(n, "term %q is in both any and none; it can never match", term)
			}
		}

		unreachable := false
		for _, term := range r.All {
			if prev, ok := catchAll[term]; ok {
				warn(n, "unreachable: required term %q always matches rule %d first", term, prev)
				unreachable = true
				break
			}
		}
		if !unreachable {
			for _, term := range r.Any {
				if prev, ok := catchAll[term]; ok {
					warn(n, "term %q is shadowed by rule %d and never decides this rule", term, prev)
				}
			}
		}
		if len(r.All) == 0 && len(r.None) == 0 {
			for _, term := range r.Any {
				if _, ok := catchAll[term]; !ok {
					catchAll[term] = n
				}
			}
		}
	}
	ignored := map[string]bool{}
	for _, term := range t.Ignore {
		if ignored[term] {
			warn(0, "ignore term %q is listed more than once", term)
		}
		ignored[term] = true
		if used[term] {
			warn(0, "ignore term %q is also used by a rule", term)
		}
	}

	if seen == nil {
		return out
	}
	counts := map[string]int{}
	for c, n := range seen {
		if t.Lang == "" || c.Lang == "" || c.Lang == t.Lang {
			counts[c.Text] += n
		}
	}
	var missing []string
	for term := range used {
		if counts[term] == 0 {
			missing = append(missing, term)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		out = append(out, Issue{Info, 0, fmt.Sprintf("%d rule terms never occur in the guide (typo, or not in this guide): %s",
			len(missing), quoteJoin(missing))})
	}
	var uncovered []string
	for term := range counts {
		if !used[term] && !ignored[term] {
			uncovered = append(uncovered, term)
		}
	}
	sort.Slice(uncovered, func(i, j int) bool {
		if a, b := counts[uncovered[i]], counts[uncovered[j]]; a != b {
			return a > b
		}
		return uncovered[i] < uncovered[j]
	})
	for _, term := range uncovered {
		out = append(out, Issue{Info, 0, fmt.Sprintf("guide term %q (%d categories) is in no rule and not ignored", term, counts[term])})
	}
	return out
}

func toSet(terms []string) map[string]bool {
	s := make(map[string]bool, len(terms))
	for _, t := range terms {
		s[t] = true
	}
	return s
}

func quoteJoin(terms []string) string {
	q := make([]string, len(terms))
	for i, t := range terms {
		q[i] = fmt.Sprintf("%q", t)
	}
	return strings.Join(q, ", ")
}
