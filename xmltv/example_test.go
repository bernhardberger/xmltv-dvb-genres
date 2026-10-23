package xmltv_test

import (
	"os"
	"strings"

	"github.com/bernhardberger/xmltv-dvb-genres/genremap"
	"github.com/bernhardberger/xmltv-dvb-genres/xmltv"
)

func ExampleRewrite() {
	tab, err := genremap.Load("schedulesdirect-en")
	if err != nil {
		panic(err)
	}
	guide := `<tv>
<programme start="20261024190000 +0200" channel="orf1">
  <title>Fußball: Bundesliga</title>
  <category lang="en">Series</category>
  <category lang="en">Soccer</category>
</programme>
</tv>
`
	if _, err := xmltv.Rewrite(strings.NewReader(guide), os.Stdout, genremap.New(tab), xmltv.Options{}); err != nil {
		panic(err)
	}
	// Output:
	// <tv>
	// <programme start="20261024190000 +0200" channel="orf1">
	//   <title>Fußball: Bundesliga</title>
	//   <category eit="0x43" lang="en">Series</category>
	//   <category eit="0x43" lang="en">Soccer</category>
	// </programme>
	// </tv>
}

func ExampleRewrite_name() {
	tab, err := genremap.Load("schedulesdirect-en")
	if err != nil {
		panic(err)
	}
	guide := `<tv>
<programme start="20261024190000 +0200" channel="orf1">
  <title>Fußball: Bundesliga</title>
  <category lang="en">Series</category>
  <category lang="en">Soccer</category>
</programme>
</tv>
`
	opt := xmltv.Options{Format: xmltv.FormatName}
	if _, err := xmltv.Rewrite(strings.NewReader(guide), os.Stdout, genremap.New(tab), opt); err != nil {
		panic(err)
	}
	// Output:
	// <tv>
	// <programme start="20261024190000 +0200" channel="orf1">
	//   <title>Fußball: Bundesliga</title>
	//   <category lang="en">Football / Soccer</category>
	// </programme>
	// </tv>
}
