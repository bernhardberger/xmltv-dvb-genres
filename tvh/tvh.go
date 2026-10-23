// Package tvh holds the English genre names that Tvheadend matches XMLTV
// category text against. A category whose text is one of these names gets the
// corresponding genre in every Tvheadend version since at least 4.2, with no
// grabber setting.
package tvh

import "github.com/bernhardberger/xmltv-dvb-genres/etsi"

// names are Tvheadend's genre names for each code it can store by name,
// joined with " / " exactly as Tvheadend joins them before matching. They are
// identical in Tvheadend 4.2 and master (src/epg.c, _epg_genre_names).
// Tvheadend also names the undefined codes of each broad genre, but always
// with the name of the broad genre's general code, which wins the match;
// those entries are left out.
var names = map[etsi.Code]string{
	0x10: "Movie / Drama",
	0x11: "Detective / Thriller",
	0x12: "Adventure / Western / War",
	0x13: "Science fiction / Fantasy / Horror",
	0x14: "Comedy",
	0x15: "Soap / Melodrama / Folkloric",
	0x16: "Romance",
	0x17: "Serious / Classical / Religious / Historical movie / Drama",
	0x18: "Adult movie / Drama",
	0x20: "News / Current affairs",
	0x21: "News / Weather report",
	0x22: "News magazine",
	0x23: "Documentary",
	0x24: "Discussion / Interview / Debate",
	0x30: "Show / Game show",
	0x31: "Game show / Quiz / Contest",
	0x32: "Variety show",
	0x33: "Talk show",
	0x40: "Sports",
	0x41: "Special events (Olympic Games, World Cup, etc.)",
	0x42: "Sports magazines",
	0x43: "Football / Soccer",
	0x44: "Tennis / Squash",
	0x45: "Team sports (excluding football)",
	0x46: "Athletics",
	0x47: "Motor sport",
	0x48: "Water sport",
	0x49: "Winter sports",
	0x4a: "Equestrian",
	0x4b: "Martial sports",
	0x50: "Children's / Youth programs",
	0x51: "Pre-school children's programs",
	0x52: "Entertainment programs for 6 to 14",
	0x53: "Entertainment programs for 10 to 16",
	0x54: "Informational / Educational / School programs",
	0x55: "Cartoons / Puppets",
	0x60: "Music / Ballet / Dance",
	0x61: "Rock / Pop",
	0x62: "Serious music / Classical music",
	0x63: "Folk / Traditional music",
	0x64: "Jazz",
	0x65: "Musical / Opera",
	0x66: "Ballet",
	0x70: "Arts / Culture (without music)",
	0x71: "Performing arts",
	0x72: "Fine arts",
	0x73: "Religion",
	0x74: "Popular culture / Traditional arts",
	0x75: "Literature",
	0x76: "Film / Cinema",
	0x77: "Experimental film / Video",
	0x78: "Broadcasting / Press",
	0x79: "New media",
	0x7a: "Arts magazines / Culture magazines",
	0x7b: "Fashion",
	0x80: "Social / Political issues / Economics",
	0x81: "Magazines / Reports / Documentary",
	0x82: "Economics / Social advisory",
	0x83: "Remarkable people",
	0x90: "Education / Science / Factual topics",
	0x91: "Nature / Animals / Environment",
	0x92: "Technology / Natural sciences",
	0x93: "Medicine / Physiology / Psychology",
	0x94: "Foreign countries / Expeditions",
	0x95: "Social / Spiritual sciences",
	0x96: "Further education",
	0x97: "Languages",
	0xa0: "Leisure hobbies",
	0xa1: "Tourism / Travel",
	0xa2: "Handicraft",
	0xa3: "Motoring",
	0xa4: "Fitness and health",
	0xa5: "Cooking",
	0xa6: "Advertisement / Shopping",
	0xa7: "Gardening",
}

// Name returns the category text Tvheadend maps to c, and false if Tvheadend
// has no name of its own for c.
func Name(c etsi.Code) (string, bool) {
	s, ok := names[c]
	return s, ok
}

// Lookup returns the code Tvheadend assigns to category text by name. Like
// Tvheadend, it ignores spaces and ASCII case, and the lowest matching code
// wins.
func Lookup(text string) (etsi.Code, bool) {
	k := key(text)
	for c := etsi.Code(0x10); c <= 0xAF; c++ {
		if s, ok := names[c]; ok && key(s) == k {
			return c, true
		}
	}
	return 0, false
}

// key reproduces Tvheadend's comparison: spaces are skipped and every other
// byte gets bit 0x20 set, which folds ASCII case.
func key(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' {
			b = append(b, s[i]|0x20)
		}
	}
	return string(b)
}
