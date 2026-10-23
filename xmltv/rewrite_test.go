package xmltv

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/bernhardberger/xmltv-dvb-genres/genremap"
)

func mapper(t *testing.T) *genremap.Mapper {
	t.Helper()
	tab, err := genremap.Load("schedulesdirect-en")
	if err != nil {
		t.Fatal(err)
	}
	return genremap.New(tab)
}

const doc = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE tv SYSTEM "xmltv.dtd">
<tv generator-info-name="test">
  <channel id="a"><display-name>A</display-name></channel>
  <programme start="20261022120000 +0200" stop="20261022130000 +0200" channel="a">
    <title lang="de">Bundesliga &amp; mehr</title>
    <category lang="en">Series</category>
    <category lang="en">Soccer</category>
    <category lang="en">Sports non-event</category>
  </programme>
  <programme start="20261022130000 +0200" stop="20261022140000 +0200" channel="a">
    <title>Nothing known</title>
    <category lang="en">Series</category>
    <category lang="en">Mystery box</category>
  </programme>
  <programme start="20261022140000 +0200" stop="20261022150000 +0200" channel="a">
    <title>Already coded</title>
    <category lang="en" eit="0x10">News</category>
  </programme>
  <programme start="20261022150000 +0200" stop="20261022160000 +0200" channel="a"><title>No categories</title></programme>
  <programme start="20261022160000 +0200" stop="20261022170000 +0200" channel="a">
    <!-- <category>Soccer</category> -->
    <category lang='en'  >Hip-Hop &amp; Rap</category><category>Music</category>
  </programme>
</tv>
`

const want = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE tv SYSTEM "xmltv.dtd">
<tv generator-info-name="test">
  <channel id="a"><display-name>A</display-name></channel>
  <programme start="20261022120000 +0200" stop="20261022130000 +0200" channel="a">
    <title lang="de">Bundesliga &amp; mehr</title>
    <category eit="0x43" lang="en">Series</category>
    <category eit="0x43" lang="en">Soccer</category>
    <category eit="0x43" lang="en">Sports non-event</category>
  </programme>
  <programme start="20261022130000 +0200" stop="20261022140000 +0200" channel="a">
    <title>Nothing known</title>
    <category lang="en">Series</category>
    <category lang="en">Mystery box</category>
  </programme>
  <programme start="20261022140000 +0200" stop="20261022150000 +0200" channel="a">
    <title>Already coded</title>
    <category lang="en" eit="0x10">News</category>
  </programme>
  <programme start="20261022150000 +0200" stop="20261022160000 +0200" channel="a"><title>No categories</title></programme>
  <programme start="20261022160000 +0200" stop="20261022170000 +0200" channel="a">
    <!-- <category>Soccer</category> -->
    <category eit="0x61" lang='en'  >Hip-Hop &amp; Rap</category><category eit="0x61">Music</category>
  </programme>
</tv>
`

func TestRewrite(t *testing.T) {
	for name, r := range map[string]func() io.Reader{
		"whole":    func() io.Reader { return strings.NewReader(doc) },
		"one-byte": func() io.Reader { return iotest.OneByteReader(strings.NewReader(doc)) },
	} {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			st, err := Rewrite(r(), &out, mapper(t), Options{CollectUnmapped: true})
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != want {
				t.Errorf("output mismatch:\n%s", out.String())
			}
			if st.Programmes != 5 || st.Mapped != 2 || st.Coded != 1 || st.NoCategory != 1 || st.Unmapped != 1 {
				t.Errorf("stats: %+v", st)
			}
			if st.Codes[0x43] != 1 || st.Codes[0x61] != 1 {
				t.Errorf("codes: %v", st.Codes)
			}
			if len(st.UnmappedTerms) != 1 || st.UnmappedTerms["mystery box"] != 1 {
				t.Errorf("unmapped terms: %v", st.UnmappedTerms)
			}
		})
	}
}

const wantName = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE tv SYSTEM "xmltv.dtd">
<tv generator-info-name="test">
  <channel id="a"><display-name>A</display-name></channel>
  <programme start="20261022120000 +0200" stop="20261022130000 +0200" channel="a">
    <title lang="de">Bundesliga &amp; mehr</title>
    <category lang="en">Football / Soccer</category>
  </programme>
  <programme start="20261022130000 +0200" stop="20261022140000 +0200" channel="a">
    <title>Nothing known</title>
    <category lang="en">Series</category>
    <category lang="en">Mystery box</category>
  </programme>
  <programme start="20261022140000 +0200" stop="20261022150000 +0200" channel="a">
    <title>Already coded</title>
    <category lang="en" eit="0x10">News</category>
  </programme>
  <programme start="20261022150000 +0200" stop="20261022160000 +0200" channel="a"><title>No categories</title></programme>
  <programme start="20261022160000 +0200" stop="20261022170000 +0200" channel="a">
    <!-- <category>Soccer</category> -->
    <category lang="en">Rock / Pop</category>
  </programme>
</tv>
`

func TestRewriteName(t *testing.T) {
	for name, r := range map[string]func() io.Reader{
		"whole":    func() io.Reader { return strings.NewReader(doc) },
		"one-byte": func() io.Reader { return iotest.OneByteReader(strings.NewReader(doc)) },
	} {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			st, err := Rewrite(r(), &out, mapper(t), Options{Format: FormatName})
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != wantName {
				t.Errorf("output mismatch:\n%s", out.String())
			}
			if st.Programmes != 5 || st.Mapped != 2 || st.Coded != 1 || st.NoCategory != 1 || st.Unmapped != 1 || st.Unnamed != 0 {
				t.Errorf("stats: %+v", st)
			}
		})
	}
}

// A programme whose only category already is a Tvheadend genre name is
// left alone, in any spelling Tvheadend accepts.
func TestRewriteNameKeepsTvheadendNames(t *testing.T) {
	in := `<tv>
<programme channel="a"><category>Comedy</category></programme>
<programme channel="a"><category lang="en">football/soccer</category></programme>
<programme channel="a"><category>Comedy</category><category>Sitcom</category></programme>
</tv>`
	var out bytes.Buffer
	st, err := Rewrite(strings.NewReader(in), &out, mapper(t), Options{Format: FormatName})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(in, `<category>Comedy</category><category>Sitcom</category>`, `<category lang="en">Comedy</category>`, 1)
	if out.String() != want {
		t.Errorf("output mismatch:\n%s", out.String())
	}
	if st.Coded != 2 || st.Mapped != 1 {
		t.Errorf("stats: %+v", st)
	}
}

// Codes without a Tvheadend name (0xB0 and up) cannot be written as a name;
// such programmes keep their categories.
func TestRewriteNameUnnamedCode(t *testing.T) {
	tab, err := genremap.Parse([]byte(`{"name":"t","rules":[{"code":"0xb1","any":["original language"]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	in := `<tv><programme channel="a"><category>Original language</category><category>Drama</category></programme></tv>`
	var out bytes.Buffer
	st, err := Rewrite(strings.NewReader(in), &out, genremap.New(tab), Options{Format: FormatName})
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != in || st.Unnamed != 1 || st.Mapped != 0 {
		t.Errorf("stats %+v, output:\n%s", st, out.String())
	}
}

func TestRewriteIsIdempotent(t *testing.T) {
	for _, f := range []Format{FormatEIT, FormatName} {
		var once, twice bytes.Buffer
		if _, err := Rewrite(strings.NewReader(doc), &once, mapper(t), Options{Format: f}); err != nil {
			t.Fatal(err)
		}
		st, err := Rewrite(bytes.NewReader(once.Bytes()), &twice, mapper(t), Options{Format: f})
		if err != nil {
			t.Fatal(err)
		}
		if once.String() != twice.String() {
			t.Errorf("format %d: second pass changed the output", f)
		}
		if st.Mapped != 0 {
			t.Errorf("format %d: second pass mapped %d programmes", f, st.Mapped)
		}
	}
}

func TestParseFormat(t *testing.T) {
	for s, want := range map[string]Format{"eit": FormatEIT, "name": FormatName} {
		if f, err := ParseFormat(s); err != nil || f != want {
			t.Errorf("ParseFormat(%q) = %d, %v", s, f, err)
		}
	}
	if _, err := ParseFormat("EIT"); err == nil {
		t.Error("ParseFormat accepted EIT")
	}
}

// A broken document must come out at least as complete as it went in: the
// part before the error is mapped, the rest copied verbatim.
func TestRewriteFailsOpen(t *testing.T) {
	broken := strings.Replace(doc, "<title>Nothing known</title>", "<title>Nothing known</titel>", 1)
	var out bytes.Buffer
	_, err := Rewrite(iotest.OneByteReader(strings.NewReader(broken)), &out, mapper(t), Options{})
	if err == nil {
		t.Fatal("no error for broken input")
	}
	got := out.String()
	if strings.Replace(got, ` eit="0x43"`, "", -1) != broken {
		t.Errorf("fail-open output differs from input beyond the mapped prefix:\n%s", got)
	}
	if strings.Count(got, `eit="0x43"`) != 3 {
		t.Errorf("mapped prefix lost:\n%s", got)
	}
}

func TestRewriteLargeInputStreams(t *testing.T) {
	var in bytes.Buffer
	in.WriteString("<tv>\n")
	prog := `<programme start="20261022120000 +0200" channel="a"><title>x</title><category lang="en">News</category></programme>` + "\n"
	for i := 0; i < 20000; i++ {
		in.WriteString(prog)
	}
	in.WriteString("</tv>\n")
	var out bytes.Buffer
	st, err := Rewrite(bytes.NewReader(in.Bytes()), &out, mapper(t), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if st.Mapped != 20000 || out.Len() != in.Len()+20000*len(` eit="0x21"`) {
		t.Errorf("mapped %d, out %d bytes", st.Mapped, out.Len())
	}
}
