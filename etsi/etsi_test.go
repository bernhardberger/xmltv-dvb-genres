package etsi

import "testing"

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Code
	}{
		{"0x43", 0x43},
		{"0X4a", 0x4A},
		{" a5 ", 0xA5},
		{"0xc0", 0xC0},
	} {
		got, err := Parse(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("Parse(%q) = %v, %v; want %v", tc.in, got, err, tc.want)
		}
	}
	for _, in := range []string{"", "0x4", "0x043", "zz", "0x00", "0x19", "0x4f", "0xd0", "0xff"} {
		if c, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) = %v; want error", in, c)
		}
	}
}

func TestString(t *testing.T) {
	// Tvheadend reads exactly two hex digits after "0x".
	if got := Code(0x05).String(); got != "0x05" {
		t.Errorf("String() = %q; want 0x05", got)
	}
	if got := Code(0xA5).String(); got != "0xa5" {
		t.Errorf("String() = %q; want 0xa5", got)
	}
}

func TestNibbles(t *testing.T) {
	c := Code(0x4B)
	if c.Major() != 0x4 || c.Minor() != 0xB || c.General() != 0x40 {
		t.Errorf("nibbles of %v: %x %x %v", c, c.Major(), c.Minor(), c.General())
	}
}

func TestAll(t *testing.T) {
	all := All()
	if len(all) != len(names) {
		t.Fatalf("All() returned %d codes; want %d", len(all), len(names))
	}
	for i, c := range all {
		if i > 0 && all[i-1] >= c {
			t.Fatalf("All() not ascending at %d", i)
		}
		if c.Name() == "" {
			t.Errorf("%v has no name", c)
		}
		// Every broad genre except special characteristics (0xB) has a
		// general entry; sub-genres must belong to one.
		if c.Major() != 0xB && !c.General().Defined() {
			t.Errorf("%v has no general code", c)
		}
	}
}
