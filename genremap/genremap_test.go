package genremap

import (
	"strings"
	"testing"

	"github.com/bernhardberger/xmltv-dvb-genres/etsi"
)

func TestEmbeddedTablesLoad(t *testing.T) {
	names := Tables()
	if len(names) == 0 {
		t.Fatal("no embedded tables")
	}
	for _, n := range names {
		tab, err := Load(n)
		if err != nil {
			t.Fatalf("Load(%q): %v", n, err)
		}
		if tab.Name != n {
			t.Errorf("table file %s.json is named %q", n, tab.Name)
		}
		for _, is := range Lint(tab, nil) {
			t.Errorf("%s: %s", n, is)
		}
	}
}

func TestSchedulesDirectMapping(t *testing.T) {
	tab, err := Load("schedulesdirect-en")
	if err != nil {
		t.Fatal(err)
	}
	m := New(tab)
	cases := []struct {
		terms []string
		want  etsi.Code
	}{
		{[]string{"News", "Series", "tvshow"}, 0x21},
		{[]string{"News", "Series", "Weather"}, 0x21},
		{[]string{"Newsmagazine", "Series"}, 0x22},
		{[]string{"Series", "Soccer", "Sports non-event"}, 0x43},
		{[]string{"Classic Sport Event", "Series", "Soccer", "Sports event"}, 0x43},
		{[]string{"Football", "Series", "Sports event"}, 0x45},
		{[]string{"Auto racing", "Classic Sport Event", "Motorsports", "Series", "Sports event"}, 0x47},
		{[]string{"Sports talk", "Series"}, 0x42},
		{[]string{"Sports event", "Track/field", "series"}, 0x46},
		{[]string{"Figure skating", "Sports event", "series"}, 0x49},
		{[]string{"Olympics", "Sports event", "Summer Olympics", "series"}, 0x41},
		{[]string{"Olympics", "Skiing", "Sports event"}, 0x49},
		{[]string{"Adventure", "Animated", "Children", "Comedy", "Series"}, 0x55},
		{[]string{"Children", "Series"}, 0x50},
		{[]string{"Animated", "Series", "Sitcom"}, 0x14},
		{[]string{"Animated", "Sitcom", "Comedy", "Dark comedy", "Adventure", "Series"}, 0x14},
		{[]string{"Historical drama", "Series", "Sitcom", "War"}, 0x14},
		{[]string{"Action", "Adventure", "Crime drama", "Series"}, 0x11},
		{[]string{"Crime", "Documentary", "Series"}, 0x23},
		{[]string{"Documentary", "Nature", "Animals"}, 0x91},
		{[]string{"Drama", "Medical", "Series"}, 0x10},
		{[]string{"Health", "Medical", "Series"}, 0x93},
		{[]string{"Law", "Reality", "Series"}, 0x30},
		{[]string{"Law", "Public affairs", "Series"}, 0x20},
		{[]string{"Cooking", "Competition reality"}, 0xA5},
		{[]string{"Series", "Shopping", "tvshow"}, 0xA6},
		{[]string{"Consumer", "Series", "Shopping", "tvshow"}, 0xA6},
		{[]string{"Comedy", "Talk"}, 0x33},
		{[]string{"Feature Film", "Comedy"}, 0x14},
		{[]string{"Feature Film"}, 0x10},
		{[]string{"Feature Film", "Musical comedy", "movie"}, 0x65},
		{[]string{"Entertainment", "Series", "tvshow"}, 0x30},
		{[]string{"Hip-Hop & Rap", "Music"}, 0x61},
	}
	for _, c := range cases {
		got := m.MapTerms(c.terms...)
		if !got.OK || got.Code != c.want {
			t.Errorf("%v: got %s (ok=%v, rule %d), want %s", c.terms, got.Code, got.OK, got.Rule, c.want)
		}
	}
	for _, terms := range [][]string{{"Series", "tvshow"}, {"Off Air"}, {"Unknown thing"}, {"Anthology", "Series"}} {
		if got := m.MapTerms(terms...); got.OK {
			t.Errorf("%v: unexpected match %s (rule %d)", terms, got.Code, got.Rule)
		}
	}
	if !m.Ignored(" Series ") || m.Ignored("soccer") {
		t.Error("Ignored is wrong")
	}
}

func TestRuleConditions(t *testing.T) {
	tab, err := Parse([]byte(`{"name":"t","lang":"en","rules":[
		{"code":"0x43","all":["a","b"],"none":["c"]},
		{"code":"0x40","any":["a"]}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	m := New(tab)
	if r := m.MapTerms("A", "b"); r.Code != 0x43 || r.Rule != 1 {
		t.Errorf("all: got %+v", r)
	}
	if r := m.MapTerms("a", "b", "c"); r.Code != 0x40 || r.Rule != 2 {
		t.Errorf("none: got %+v", r)
	}
	if r := m.Map([]Category{{Text: "a", Lang: "de"}}); r.OK {
		t.Errorf("lang: foreign category matched: %+v", r)
	}
	if r := m.Map([]Category{{Text: "a", Lang: "EN"}}); !r.OK {
		t.Errorf("lang: case-insensitive lang did not match")
	}
}

func TestTableOrderOverride(t *testing.T) {
	local, err := Parse([]byte(`{"name":"local","rules":[{"code":"0x40","any":["soccer"]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	sd, err := Load("schedulesdirect-en")
	if err != nil {
		t.Fatal(err)
	}
	if r := New(local, sd).MapTerms("soccer"); r.Code != 0x40 || r.Table != "local" {
		t.Errorf("override: got %+v", r)
	}
	if r := New(local, sd).MapTerms("news"); r.Code != 0x21 || r.Table != "schedulesdirect-en" {
		t.Errorf("fallthrough: got %+v", r)
	}
}

func TestParseRejects(t *testing.T) {
	for _, doc := range []string{
		`{"name":"t","rules":[]}`,
		`{"rules":[{"code":"0x40","any":["a"]}]}`,
		`{"name":"t","rules":[{"code":"0x40"}]}`,
		`{"name":"t","rules":[{"code":"0x00","any":["a"]}]}`,
		`{"name":"t","rules":[{"code":"0x40","any":["a"],"bogus":1}]}`,
		`{"name":"t","rules":[{"code":"0x40","any":["a","  "]}]}`,
		`{"name":"t","rules":[{"code":"0x40","any":["a"]}],"ignore":[""]}`,
	} {
		if _, err := Parse([]byte(doc)); err == nil {
			t.Errorf("accepted %s", doc)
		}
	}
	if _, err := Load("nope"); err == nil || !strings.Contains(err.Error(), "schedulesdirect-en") {
		t.Errorf("Load unknown: %v", err)
	}
}
