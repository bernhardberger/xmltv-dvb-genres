package genremap

import (
	"strings"
	"testing"
)

func TestLint(t *testing.T) {
	tab, err := Parse([]byte(`{"name":"t","lang":"en","rules":[
		{"code":"0x40","any":["sport","Sport "]},
		{"code":"0x43","any":["soccer","sport"]},
		{"code":"0x45","all":["sport","team"]},
		{"code":"0xb3","any":["live"]},
		{"code":"0x21","all":["news"],"none":["news"]},
		{"code":"0x22","any":["magazine"],"none":["magazine"]}
	],"ignore":["series","live"]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`warning: rule 1: term "sport" is listed more than once`,
		`warning: rule 2: term "sport" is shadowed by rule 1`,
		`warning: rule 3: unreachable: required term "sport" always matches rule 1 first`,
		`warning: rule 4: code 0xb3 has no Tvheadend name`,
		`warning: rule 5: term "news" is in both all and none`,
		`warning: rule 6: term "magazine" is in both any and none`,
		`warning: ignore term "live" is also used by a rule`,
	}
	got := Lint(tab, nil)
	if len(got) != len(want) {
		t.Errorf("got %d issues, want %d:\n%v", len(got), len(want), got)
	}
	for _, w := range want {
		found := false
		for _, is := range got {
			if strings.HasPrefix(is.String(), w) {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %q in\n%v", w, got)
		}
	}
}

func TestLintAgainstGuide(t *testing.T) {
	tab, err := Parse([]byte(`{"name":"t","lang":"en","rules":[
		{"code":"0x43","any":["soccer","socer"]}
	],"ignore":["series"]}`))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[Category]int{
		{Text: "soccer", Lang: "en"}:  5,
		{Text: "series", Lang: "en"}:  9,
		{Text: "tennis", Lang: ""}:    2,
		{Text: "fußball", Lang: "de"}: 7, // other language: not this table's business
	}
	var lines []string
	for _, is := range Lint(tab, seen) {
		if is.Level != Info {
			t.Errorf("unexpected %s", is)
		}
		lines = append(lines, is.String())
	}
	all := strings.Join(lines, "\n")
	for _, w := range []string{`never occur in the guide (typo, or not in this guide): "socer"`, `guide term "tennis" (2 categories)`} {
		if !strings.Contains(all, w) {
			t.Errorf("missing %q in\n%s", w, all)
		}
	}
	if strings.Contains(all, "fußball") || strings.Contains(all, `"series"`) {
		t.Errorf("reported foreign or ignored term:\n%s", all)
	}
}
