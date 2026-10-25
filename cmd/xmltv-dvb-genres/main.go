// Command xmltv-dvb-genres reads an XMLTV document and adds a DVB genre code
// to every programme it can map. With -format name (the default) it replaces
// the categories with one Tvheadend genre name, which any Tvheadend version
// recognises without a setting. With -format eit it adds eit="0xNN" to each
// category, for Tvheadend's "Category Code XPath" setting; see
// docs/tvheadend.md for the Tvheadend builds that handle it safely.
//
// Usage:
//
//	tv_grab_zz_sdjson | xmltv-dvb-genres [-format name|eit] [-table name|file.json]... [-report] > guide.xml
//
// The tool fails open: if the input is malformed, or the tables cannot be
// loaded, the guide is still written to stdout (mapped up to the error or not
// at all) and the problem is reported on stderr.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/bernhardberger/xmltv-dvb-genres/etsi"
	"github.com/bernhardberger/xmltv-dvb-genres/genremap"
	"github.com/bernhardberger/xmltv-dvb-genres/xmltv"
)

type tableList []string

func (l *tableList) String() string     { return strings.Join(*l, ",") }
func (l *tableList) Set(v string) error { *l = append(*l, v); return nil }

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("xmltv-dvb-genres", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var tables tableList
	fs.Var(&tables, "table", "rule table: embedded name or path to a .json file; repeat to chain, first match wins (default schedulesdirect-en)")
	format := fs.String("format", "name", "output format: name (replace categories with one Tvheadend genre name) or eit (add eit attributes, keep categories)")
	report := fs.Bool("report", false, "print mapping statistics and unmapped category terms to stderr")
	list := fs.Bool("list-tables", false, "list embedded tables and exit")
	codes := fs.Bool("list-codes", false, "list the DVB genre codes and exit")
	check := fs.String("check", "", "lint a rule table (embedded name or .json file) and exit; with an input guide, also compare its terms with the guide")
	version := fs.Bool("version", false, "print version and exit")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: xmltv-dvb-genres [flags] [input.xml] < guide.xml > guide-with-codes.xml\n")
		fmt.Fprintf(stderr, "       xmltv-dvb-genres -check table.json [guide.xml]\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	switch {
	case *version:
		fmt.Fprintln(stdout, buildVersion())
		return 0
	case *list:
		for _, n := range genremap.Tables() {
			t, err := genremap.Load(n)
			if err != nil {
				fmt.Fprintf(stderr, "xmltv-dvb-genres: %v\n", err)
				return 1
			}
			fmt.Fprintf(stdout, "%s\t%d rules\t%s\n", n, len(t.Rules), t.Description)
		}
		return 0
	case *codes:
		for _, c := range etsi.All() {
			fmt.Fprintf(stdout, "%s\t%s\n", c, c.Name())
		}
		return 0
	}

	if *check != "" {
		if fs.NArg() > 1 {
			fs.Usage()
			return 2
		}
		return runCheck(*check, fs.Arg(0), stdout, stderr)
	}

	in := stdin
	switch fs.NArg() {
	case 0:
	case 1:
		f, err := os.Open(fs.Arg(0))
		if err != nil {
			fmt.Fprintf(stderr, "xmltv-dvb-genres: %v\n", err)
			return 1
		}
		defer f.Close()
		in = f
	default:
		fs.Usage()
		return 2
	}

	passThrough := func(err error) int {
		fmt.Fprintf(stderr, "xmltv-dvb-genres: %v; passing guide through unchanged\n", err)
		if _, cerr := io.Copy(stdout, in); cerr != nil {
			fmt.Fprintf(stderr, "xmltv-dvb-genres: %v\n", cerr)
		}
		return 1
	}
	outFormat, err := xmltv.ParseFormat(*format)
	if err != nil {
		return passThrough(err)
	}
	if len(tables) == 0 {
		tables = tableList{"schedulesdirect-en"}
	}
	loaded := make([]*genremap.Table, 0, len(tables))
	for _, ref := range tables {
		t, err := genremap.Load(ref)
		if err != nil {
			return passThrough(err)
		}
		loaded = append(loaded, t)
	}

	st, err := xmltv.Rewrite(in, stdout, genremap.New(loaded...), xmltv.Options{Format: outFormat, CollectUnmapped: *report})
	if *report {
		printReport(stderr, st)
	}
	if err != nil {
		fmt.Fprintf(stderr, "xmltv-dvb-genres: %v; rest of guide passed through unchanged\n", err)
		return 1
	}
	return 0
}

// runCheck lints one table, optionally against a guide file. It exits 1 on
// load errors and warnings; info findings alone do not fail.
func runCheck(ref, guide string, stdout, stderr io.Writer) int {
	t, err := genremap.Load(ref)
	if err != nil {
		fmt.Fprintf(stdout, "%s: error: %v\n", ref, err)
		return 1
	}
	var (
		seen     map[genremap.Category]int
		examples map[string][]string
	)
	if guide != "" {
		f, err := os.Open(guide)
		if err != nil {
			fmt.Fprintf(stderr, "xmltv-dvb-genres: %v\n", err)
			return 1
		}
		st, err := xmltv.Rewrite(f, io.Discard, genremap.New(t), xmltv.Options{CollectTerms: true})
		f.Close()
		if err != nil {
			fmt.Fprintf(stderr, "xmltv-dvb-genres: %s: %v\n", guide, err)
			return 1
		}
		seen, examples = st.Terms, st.Examples
		fmt.Fprintf(stdout, "%s: %d of %d programmes in %s mapped by this table alone\n", ref, st.Mapped, st.Programmes, guide)
	}
	issues := genremap.Lint(t, seen)
	warnings := 0
	for _, is := range issues {
		if is.Level == genremap.Warning {
			warnings++
		}
		line := is.String()
		if ex := examples[is.Term]; is.Term != "" && len(ex) > 0 {
			q := make([]string, len(ex))
			for i, title := range ex {
				q[i] = fmt.Sprintf("%q", title)
			}
			line += ", e.g. " + strings.Join(q, ", ")
		}
		fmt.Fprintf(stdout, "%s: %s\n", ref, line)
	}
	fmt.Fprintf(stdout, "%s: %d rules, %d warnings\n", ref, len(t.Rules), warnings)
	if warnings > 0 {
		return 1
	}
	return 0
}

func printReport(w io.Writer, st xmltv.Stats) {
	bw := bufio.NewWriter(w)
	defer bw.Flush()
	pct := func(n int) float64 {
		if st.Programmes == 0 {
			return 0
		}
		return 100 * float64(n) / float64(st.Programmes)
	}
	fmt.Fprintf(bw, "programmes %d: mapped %d (%.1f%%), unmapped %d (%.1f%%), already coded %d, no category %d\n",
		st.Programmes, st.Mapped, pct(st.Mapped), st.Unmapped, pct(st.Unmapped), st.Coded, st.NoCategory)
	if st.Unnamed > 0 {
		fmt.Fprintf(bw, "left alone because Tvheadend has no name for their code: %d\n", st.Unnamed)
	}

	byCode := make([]etsi.Code, 0, len(st.Codes))
	for c := range st.Codes {
		byCode = append(byCode, c)
	}
	sort.Slice(byCode, func(i, j int) bool { return byCode[i] < byCode[j] })
	for _, c := range byCode {
		fmt.Fprintf(bw, "  %s %7d  %s\n", c, st.Codes[c], c.Name())
	}

	if len(st.UnmappedTerms) == 0 {
		return
	}
	terms := make([]string, 0, len(st.UnmappedTerms))
	for t := range st.UnmappedTerms {
		terms = append(terms, t)
	}
	sort.Slice(terms, func(i, j int) bool {
		if a, b := st.UnmappedTerms[terms[i]], st.UnmappedTerms[terms[j]]; a != b {
			return a > b
		}
		return terms[i] < terms[j]
	})
	fmt.Fprintln(bw, "unmapped terms (programmes):")
	for _, t := range terms {
		fmt.Fprintf(bw, "  %7d  %s\n", st.UnmappedTerms[t], t)
	}
}

func buildVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "(devel)"
}
