package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const guide = `<tv><programme channel="a"><category lang="en">News</category><category lang="en">Odd</category></programme>` +
	`<programme channel="a"><category lang="en">Odd</category></programme></tv>`

func TestRun(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-report"}, strings.NewReader(guide), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `<category eit="0x21" lang="en">News</category>`) {
		t.Errorf("stdout: %s", out.String())
	}
	for _, s := range []string{"programmes 2: mapped 1 (50.0%)", "0x21       1", "1  odd"} {
		if !strings.Contains(errOut.String(), s) {
			t.Errorf("report lacks %q:\n%s", s, errOut.String())
		}
	}
}

func TestRunFormatName(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-format", "name"}, strings.NewReader(guide), &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	want := strings.Replace(guide, `<category lang="en">News</category><category lang="en">Odd</category>`, `<category lang="en">News / Weather report</category>`, 1)
	if out.String() != want {
		t.Errorf("stdout: %s", out.String())
	}
}

func TestRunBadArgsPassThrough(t *testing.T) {
	for _, args := range [][]string{{"-table", "missing"}, {"-format", "bogus"}} {
		var out, errOut bytes.Buffer
		if code := run(args, strings.NewReader(guide), &out, &errOut); code != 1 {
			t.Errorf("%v: exit %d, want 1", args, code)
		}
		if out.String() != guide {
			t.Errorf("%v: guide not passed through: %s", args, out.String())
		}
		if errOut.Len() == 0 {
			t.Errorf("%v: no error reported", args)
		}
	}
}

func TestRunListings(t *testing.T) {
	for _, flag := range []string{"-list-tables", "-list-codes", "-version"} {
		var out, errOut bytes.Buffer
		if code := run([]string{flag}, nil, &out, &errOut); code != 0 || out.Len() == 0 {
			t.Errorf("%s: exit %d, stdout %q, stderr %q", flag, code, out.String(), errOut.String())
		}
	}
}

func TestRunCheck(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"name":"bad","rules":[{"code":"0x40","any":["news"]},{"code":"0x21","any":["news"]}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	g := filepath.Join(dir, "guide.xml")
	titled := strings.Replace(guide, `<programme channel="a">`, `<programme channel="a"><title>Odd Show</title>`, 1)
	if err := os.WriteFile(g, []byte(titled), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if code := run([]string{"-check", "schedulesdirect-en"}, nil, &out, &out); code != 0 {
		t.Errorf("embedded table: exit %d\n%s", code, out.String())
	}
	out.Reset()
	if code := run([]string{"-check", bad, g}, nil, &out, &out); code != 1 {
		t.Errorf("bad table: exit %d", code)
	}
	for _, s := range []string{`term "news" is shadowed by rule 1`, `guide term "odd" (2 categories) is in no rule and not ignored, e.g. "Odd Show"`, "1 of 2 programmes", "1 warnings"} {
		if !strings.Contains(out.String(), s) {
			t.Errorf("check output lacks %q:\n%s", s, out.String())
		}
	}
	out.Reset()
	if code := run([]string{"-check", filepath.Join(dir, "missing.json")}, nil, &out, &out); code != 1 || !strings.Contains(out.String(), "error:") {
		t.Errorf("missing table: exit %d\n%s", code, out.String())
	}
}
