// Package xmltv adds DVB genre codes to the categories of an XMLTV document.
//
// The document is streamed and copied byte for byte. Only the categories of
// mapped programmes change, in one of two formats:
//
//   - FormatEIT inserts an eit="0xNN" attribute into each <category> start
//     tag. Tvheadend reads it when its XMLTV grabber module has "Category
//     Code XPath" set to "@eit".
//   - FormatName replaces the programme's categories with a single category
//     holding Tvheadend's English name for the code. Tvheadend matches that
//     name without any setting, but the original category text is lost.
package xmltv

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/bernhardberger/xmltv-dvb-genres/etsi"
	"github.com/bernhardberger/xmltv-dvb-genres/genremap"
	"github.com/bernhardberger/xmltv-dvb-genres/tvh"
)

// Attr is the attribute name Tvheadend's documentation uses for category codes.
const Attr = "eit"

// Stats summarises a rewrite.
type Stats struct {
	Programmes int               // programmes seen
	Mapped     int               // programmes that received a code
	Coded      int               // programmes left alone because a category already had a code (or, in FormatName, a Tvheadend genre name)
	NoCategory int               // programmes without categories
	Unmapped   int               // programmes with categories but no matching rule
	Unnamed    int               // FormatName: mapped programmes left alone because Tvheadend has no name for the code
	Codes      map[etsi.Code]int // programmes per assigned code
	// UnmappedTerms counts the categories of unmapped programmes, excluding
	// terms the tables ignore. Filled only when Options.CollectUnmapped is set.
	UnmappedTerms map[string]int
	// Terms counts every category in the guide (text normalized, lang lower
	// case). Filled only when Options.CollectTerms is set.
	Terms map[genremap.Category]int
	// Examples holds up to MaxExamples distinct programme titles per
	// normalized category text. Filled only when Options.CollectTerms is set.
	Examples map[string][]string
}

// MaxExamples is the number of programme titles Stats.Examples keeps per term.
const MaxExamples = 3

// Format selects how a mapped programme's code is written.
type Format int

const (
	FormatEIT  Format = iota // eit attribute on every category; original text kept
	FormatName               // categories replaced by one Tvheadend genre name
)

// ParseFormat parses "eit" or "name".
func ParseFormat(s string) (Format, error) {
	switch s {
	case "eit":
		return FormatEIT, nil
	case "name":
		return FormatName, nil
	}
	return 0, fmt.Errorf("xmltv: unknown format %q (want eit or name)", s)
}

// Options controls a rewrite.
type Options struct {
	Format          Format
	CollectUnmapped bool
	CollectTerms    bool
}

// Rewrite copies an XMLTV document from r to w, writing the code chosen by m
// for each mapped programme in opt.Format. Programmes are left untouched when
// no rule matches or when any of their categories already carries an eit
// attribute. In FormatName, a programme whose only category is already a
// Tvheadend genre name is left untouched too, so a second pass changes
// nothing.
//
// If the input is not well-formed, Rewrite stops mapping at the error, copies
// the rest of the input to w unchanged and returns the error, so the output is
// never less complete than the input.
func Rewrite(r io.Reader, w io.Writer, m *genremap.Mapper, opt Options) (Stats, error) {
	st := Stats{Codes: map[etsi.Code]int{}}
	if opt.CollectUnmapped {
		st.UnmappedTerms = map[string]int{}
	}
	if opt.CollectTerms {
		st.Terms = map[genremap.Category]int{}
		st.Examples = map[string][]string{}
	}
	bw := bufio.NewWriterSize(w, 1<<16)
	rec := &recorder{r: r}
	err := rewrite(rec, bw, m, opt, &st)
	if err != nil && !errors.Is(err, errWrite) {
		// Fail open: everything the decoder has read but we have not yet
		// written is still in the recorder; send it and the rest verbatim.
		if _, werr := bw.Write(rec.buf); werr == nil {
			_, werr = io.Copy(bw, r)
			if werr != nil {
				err = errors.Join(err, werr)
			}
		} else {
			err = errors.Join(err, werr)
		}
	}
	if ferr := bw.Flush(); ferr != nil && err == nil {
		err = ferr
	}
	return st, err
}

var errWrite = errors.New("xmltv: write failed")

type category struct {
	off  int64 // offset of '<' of the start tag
	end  int64 // offset just past the end tag
	lang string
	text strings.Builder
}

func rewrite(rec *recorder, w *bufio.Writer, m *genremap.Mapper, opt Options, st *Stats) error {
	d := xml.NewDecoder(rec)
	d.Entity = xml.HTMLEntity

	var (
		inProg    bool
		progStart int64
		cats      []*category
		cur       *category
		coded     bool
		inTitle   bool
		title     strings.Builder // the programme's first title, for Stats.Examples
	)
	write := func(b []byte) error {
		if _, err := w.Write(b); err != nil {
			return fmt.Errorf("%w: %v", errWrite, err)
		}
		return nil
	}

	for {
		off := d.InputOffset()
		tok, err := d.Token()
		if err == io.EOF {
			return write(rec.take(rec.end()))
		}
		if err != nil {
			return fmt.Errorf("xmltv: offset %d: %w", d.InputOffset(), err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case t.Name.Local == "programme" && !inProg:
				// Flush everything before the programme so the recorder only
				// ever holds one programme.
				if err := write(rec.take(off)); err != nil {
					return err
				}
				inProg, progStart, cats, cur, coded = true, off, cats[:0], nil, false
				title.Reset()
			case t.Name.Local == "title" && inProg && opt.CollectTerms && cur == nil && title.Len() == 0:
				inTitle = true
			case t.Name.Local == "category" && inProg && cur == nil:
				cur = &category{off: off}
				for _, a := range t.Attr {
					switch {
					case a.Name.Space == "" && a.Name.Local == "lang":
						cur.lang = a.Value
					case a.Name.Space == "" && a.Name.Local == Attr:
						coded = true
					}
				}
			}
		case xml.CharData:
			switch {
			case cur != nil:
				cur.text.Write(t)
			case inTitle:
				title.Write(t)
			}
		case xml.EndElement:
			switch {
			case t.Name.Local == "title" && inTitle:
				inTitle = false
			case t.Name.Local == "category" && cur != nil:
				cur.end = d.InputOffset()
				cats = append(cats, cur)
				cur = nil
			case t.Name.Local == "programme" && inProg:
				inProg = false
				block := rec.take(d.InputOffset())
				out, err := stamp(block, progStart, cats, coded, strings.TrimSpace(title.String()), m, opt, st)
				if err != nil {
					// The block has left the recorder; write it unchanged
					// before failing open on the rest.
					return errors.Join(err, write(block))
				}
				if err := write(out); err != nil {
					return err
				}
			}
		}
		if !inProg && rec.buffered() > 1<<16 {
			if err := write(rec.take(d.InputOffset())); err != nil {
				return err
			}
		}
	}
}

// stamp returns block (the raw bytes of one programme starting at offset
// base) with the chosen code written in opt.Format.
func stamp(block []byte, base int64, cats []*category, coded bool, title string, m *genremap.Mapper, opt Options, st *Stats) ([]byte, error) {
	st.Programmes++
	if opt.CollectTerms {
		for _, c := range cats {
			text := genremap.Normalize(c.text.String())
			st.Terms[genremap.Category{Text: text, Lang: strings.ToLower(c.lang)}]++
			if ex := st.Examples[text]; title != "" && len(ex) < MaxExamples && !slices.Contains(ex, title) {
				st.Examples[text] = append(ex, title)
			}
		}
	}
	switch {
	case len(cats) == 0:
		st.NoCategory++
		return block, nil
	case coded:
		st.Coded++
		return block, nil
	case opt.Format == FormatName && len(cats) == 1:
		if _, ok := tvh.Lookup(cats[0].text.String()); ok {
			st.Coded++
			return block, nil
		}
	}
	in := make([]genremap.Category, len(cats))
	for i, c := range cats {
		in[i] = genremap.Category{Text: c.text.String(), Lang: c.lang}
	}
	res := m.Map(in)
	if !res.OK {
		st.Unmapped++
		if opt.CollectUnmapped {
			for _, c := range in {
				if t := genremap.Normalize(c.Text); t != "" && !m.Ignored(t) {
					st.UnmappedTerms[t]++
				}
			}
		}
		return block, nil
	}
	if opt.Format == FormatName {
		name, ok := tvh.Name(res.Code)
		if !ok {
			st.Unnamed++
			return block, nil
		}
		st.Mapped++
		st.Codes[res.Code]++
		return replace(block, base, cats, name)
	}
	st.Mapped++
	st.Codes[res.Code]++

	attr := []byte(fmt.Sprintf(` %s="%s"`, Attr, res.Code))
	const tag = "<category"
	out := make([]byte, 0, len(block)+len(cats)*len(attr))
	last := 0
	for _, c := range cats {
		at := int(c.off - base)
		if at < last || at+len(tag) > len(block) || !bytes.HasPrefix(block[at:], []byte(tag)) {
			return nil, fmt.Errorf("xmltv: offset %d: category start tag not found", c.off)
		}
		at += len(tag)
		out = append(out, block[last:at]...)
		out = append(out, attr...)
		last = at
	}
	return append(out, block[last:]...), nil
}

// replace returns block with the first category replaced by a category
// holding name and every other category removed, together with the
// whitespace before it.
func replace(block []byte, base int64, cats []*category, name string) ([]byte, error) {
	out := make([]byte, 0, len(block))
	last := 0
	for i, c := range cats {
		at, end := int(c.off-base), int(c.end-base)
		if at < last || end > len(block) || !bytes.HasPrefix(block[at:], []byte("<category")) {
			return nil, fmt.Errorf("xmltv: offset %d: category element not found", c.off)
		}
		if i == 0 {
			out = append(out, block[last:at]...)
			out = append(out, `<category lang="en">`...)
			out = append(out, textEscaper.Replace(name)...)
			out = append(out, "</category>"...)
		} else {
			keep := at
			for keep > last && isSpace(block[keep-1]) {
				keep--
			}
			out = append(out, block[last:keep]...)
		}
		last = end
	}
	return append(out, block[last:]...), nil
}

var textEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func isSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }

// recorder keeps every byte read from r that has not yet been taken, so the
// output can reuse the exact input bytes.
type recorder struct {
	r    io.Reader
	buf  []byte
	base int64 // input offset of buf[0]
}

func (rc *recorder) Read(p []byte) (int, error) {
	n, err := rc.r.Read(p)
	rc.buf = append(rc.buf, p[:n]...)
	return n, err
}

func (rc *recorder) end() int64    { return rc.base + int64(len(rc.buf)) }
func (rc *recorder) buffered() int { return len(rc.buf) }

// take removes and returns the bytes before input offset off.
func (rc *recorder) take(off int64) []byte {
	n := int(off - rc.base)
	out := make([]byte, n)
	copy(out, rc.buf[:n])
	rc.buf = rc.buf[:copy(rc.buf, rc.buf[n:])]
	rc.base = off
	return out
}
