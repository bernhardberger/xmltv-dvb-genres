// Package etsi provides the DVB content (genre) codes defined in
// ETSI EN 300 468, clause 6.2.9, Table 29 ("Content_nibble level 1 and 2
// assignments").
//
// A code is one byte: the high nibble is the broad genre (level 1), the low
// nibble the sub-genre (level 2). DVB broadcasters carry these codes in the
// EIT content descriptor; Tvheadend stores EPG genres as the same codes.
package etsi

import (
	"fmt"
	"strconv"
	"strings"
)

// Code is a DVB content code: level-1 nibble in the high four bits,
// level-2 nibble in the low four bits.
type Code uint8

// Major returns the level-1 (broad genre) nibble.
func (c Code) Major() uint8 { return uint8(c) >> 4 }

// Minor returns the level-2 (sub-genre) nibble.
func (c Code) Minor() uint8 { return uint8(c) & 0x0F }

// General returns the "general" code of c's broad genre (minor nibble 0).
func (c Code) General() Code { return c & 0xF0 }

// String formats c as a two-digit lowercase hex literal, e.g. "0x43".
// This is the form Tvheadend's "Category Code XPath" parser expects.
func (c Code) String() string { return fmt.Sprintf("0x%02x", uint8(c)) }

// Defined reports whether c is assigned a meaning in Table 29.
// Code 0x00 ("undefined content"), reserved and user-defined codes are not.
func (c Code) Defined() bool {
	_, ok := names[c]
	return ok
}

// Name returns the English description of c as worded in Table 29,
// or "" if c is not defined.
func (c Code) Name() string { return names[c] }

// Parse parses a code written as "0x43", "0X43" or "43" (hex). It accepts
// only codes that are defined in Table 29.
func Parse(s string) (Code, error) {
	t := strings.TrimSpace(s)
	t = strings.TrimPrefix(strings.TrimPrefix(t, "0x"), "0X")
	if len(t) != 2 {
		return 0, fmt.Errorf("etsi: invalid code %q: want two hex digits, e.g. 0x43", s)
	}
	v, err := strconv.ParseUint(t, 16, 8)
	if err != nil {
		return 0, fmt.Errorf("etsi: invalid code %q: %w", s, err)
	}
	c := Code(v)
	if !c.Defined() {
		return 0, fmt.Errorf("etsi: code %s is not defined in EN 300 468 Table 29", c)
	}
	return c, nil
}

// All returns every defined code in ascending order.
func All() []Code {
	out := make([]Code, 0, len(names))
	for c := 0; c <= 0xFF; c++ {
		if _, ok := names[Code(c)]; ok {
			out = append(out, Code(c))
		}
	}
	return out
}

// names follows ETSI EN 300 468 V1.17.1, Table 29.
var names = map[Code]string{
	0x10: "movie/drama (general)",
	0x11: "detective/thriller",
	0x12: "adventure/western/war",
	0x13: "science fiction/fantasy/horror",
	0x14: "comedy",
	0x15: "soap/melodrama/folkloric",
	0x16: "romance",
	0x17: "serious/classical/religious/historical movie/drama",
	0x18: "adult movie/drama",

	0x20: "news/current affairs (general)",
	0x21: "news/weather report",
	0x22: "news magazine",
	0x23: "documentary",
	0x24: "discussion/interview/debate",

	0x30: "show/game show (general)",
	0x31: "game show/quiz/contest",
	0x32: "variety show",
	0x33: "talk show",

	0x40: "sports (general)",
	0x41: "special events (Olympic Games, World Cup, etc.)",
	0x42: "sports magazines",
	0x43: "football/soccer",
	0x44: "tennis/squash",
	0x45: "team sports (excluding football)",
	0x46: "athletics",
	0x47: "motor sport",
	0x48: "water sport",
	0x49: "winter sports",
	0x4A: "equestrian",
	0x4B: "martial sports",

	0x50: "children's/youth programmes (general)",
	0x51: "pre-school children's programmes",
	0x52: "entertainment programmes for 6 to 14",
	0x53: "entertainment programmes for 10 to 16",
	0x54: "informational/educational/school programmes",
	0x55: "cartoons/puppets",

	0x60: "music/ballet/dance (general)",
	0x61: "rock/pop",
	0x62: "serious music/classical music",
	0x63: "folk/traditional music",
	0x64: "jazz",
	0x65: "musical/opera",
	0x66: "ballet",

	0x70: "arts/culture (without music, general)",
	0x71: "performing arts",
	0x72: "fine arts",
	0x73: "religion",
	0x74: "popular culture/traditional arts",
	0x75: "literature",
	0x76: "film/cinema",
	0x77: "experimental film/video",
	0x78: "broadcasting/press",
	0x79: "new media",
	0x7A: "arts/culture magazines",
	0x7B: "fashion",

	0x80: "social/political issues/economics (general)",
	0x81: "magazines/reports/documentary",
	0x82: "economics/social advisory",
	0x83: "remarkable people",

	0x90: "education/science/factual topics (general)",
	0x91: "nature/animals/environment",
	0x92: "technology/natural sciences",
	0x93: "medicine/physiology/psychology",
	0x94: "foreign countries/expeditions",
	0x95: "social/spiritual sciences",
	0x96: "further education",
	0x97: "languages",

	0xA0: "leisure hobbies (general)",
	0xA1: "tourism/travel",
	0xA2: "handicraft",
	0xA3: "motoring",
	0xA4: "fitness and health",
	0xA5: "cooking",
	0xA6: "advertisement/shopping",
	0xA7: "gardening",

	0xB0: "original language",
	0xB1: "black and white",
	0xB2: "unpublished",
	0xB3: "live broadcast",
	0xB4: "plano-stereoscopic",
	0xB5: "local or regional",

	0xC0: "adult (general)",
}
