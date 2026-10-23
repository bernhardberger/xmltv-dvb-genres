package tvh

import (
	"testing"

	"github.com/bernhardberger/xmltv-dvb-genres/etsi"
)

// Every DVB-defined code that Tvheadend can display must have a name that
// Tvheadend maps back to the same code.
func TestNamesRoundTrip(t *testing.T) {
	for _, c := range etsi.All() {
		if c < 0x10 || c > 0xAF {
			continue
		}
		name, ok := Name(c)
		if !ok {
			t.Errorf("%s (%s): no Tvheadend name", c, c.Name())
			continue
		}
		if got, ok := Lookup(name); !ok || got != c {
			t.Errorf("%s: Lookup(%q) = %s, %v", c, name, got, ok)
		}
	}
	for c := range names {
		if !c.Defined() {
			t.Errorf("%s: name for a code DVB does not define", c)
		}
	}
}

func TestLookup(t *testing.T) {
	for text, want := range map[string]etsi.Code{
		"Football / Soccer":           0x43,
		"football/soccer":             0x43,
		"Newsmagazine":                0x22,
		"COMEDY":                      0x14,
		"Children's / Youth Programs": 0x50,
		"Advertisement / Shopping":    0xA6,
	} {
		if got, ok := Lookup(text); !ok || got != want {
			t.Errorf("Lookup(%q) = %s, %v; want %s", text, got, ok, want)
		}
	}
	for _, text := range []string{"Soccer", "News", "Football", "", "Comedy drama"} {
		if got, ok := Lookup(text); ok {
			t.Errorf("Lookup(%q) = %s, want no match", text, got)
		}
	}
	if _, ok := Name(0xB1); ok {
		t.Error("Name(0xb1) should not exist")
	}
}
