package genremap_test

import (
	"fmt"

	"github.com/bernhardberger/xmltv-dvb-genres/genremap"
)

func ExampleMapper_MapTerms() {
	tab, err := genremap.Load("schedulesdirect-en")
	if err != nil {
		panic(err)
	}
	m := genremap.New(tab)

	res := m.MapTerms("Series", "Soccer", "Sports event")
	fmt.Println(res.OK, res.Code, res.Code.Name())

	// Schedules Direct "Football" is American football.
	res = m.MapTerms("Football", "Sports event")
	fmt.Println(res.OK, res.Code, res.Code.Name())

	res = m.MapTerms("Series", "tvshow")
	fmt.Println(res.OK)
	// Output:
	// true 0x43 football/soccer
	// true 0x45 team sports (excluding football)
	// false
}

func ExampleNew_override() {
	// A local table listed first overrides individual rules of the
	// embedded one; everything it does not match falls through.
	local, err := genremap.Parse([]byte(`{
		"name": "local",
		"rules": [{"code": "0x45", "any": ["cricket"]}, {"code": "0x40", "any": ["darts"]}]
	}`))
	if err != nil {
		panic(err)
	}
	sd, err := genremap.Load("schedulesdirect-en")
	if err != nil {
		panic(err)
	}
	m := genremap.New(local, sd)

	for _, terms := range [][]string{{"Darts", "Sports event"}, {"Cooking", "Series"}} {
		res := m.MapTerms(terms...)
		fmt.Println(terms, res.Code, res.Table)
	}
	// Output:
	// [Darts Sports event] 0x40 local
	// [Cooking Series] 0xa5 schedulesdirect-en
}

func ExampleLint() {
	tab, err := genremap.Parse([]byte(`{
		"name": "example",
		"rules": [
			{"code": "0x40", "any": ["sport"]},
			{"code": "0x43", "all": ["sport"], "any": ["soccer"]}
		]
	}`))
	if err != nil {
		panic(err)
	}
	for _, issue := range genremap.Lint(tab, nil) {
		fmt.Println(issue)
	}
	// Output:
	// warning: rule 2: unreachable: required term "sport" always matches rule 1 first
}
