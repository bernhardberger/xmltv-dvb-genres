# xmltv-dvb-genres

Map XMLTV category text to DVB (EIT) genre codes for Tvheadend.

Tvheadend stores EPG genres as the numeric content codes from ETSI EN 300 468.
For XMLTV guides it derives them from `<category>` text, and only text that
exactly matches one of its built-in English genre names counts. Most real guides
use their own vocabulary, so most genres are lost. In one Schedules Direct
guide, Tvheadend matched categories on 801 of 3,000 sampled programmes.

`xmltv-dvb-genres` sits between a grabber and Tvheadend. It picks one DVB
code per programme from a rule table for the guide's vocabulary and writes it
in one of two formats.

By default (`-format name`) it replaces the programme's categories with a
single category holding Tvheadend's own name for the code. Every Tvheadend
version matches that name without any setting:

```xml
<category lang="en">Football / Soccer</category>
```

With `-format eit` it keeps the categories and adds the code to each of them
as `eit="0xNN"`, which Tvheadend reads through its Category Code XPath
setting:

```xml
<category eit="0x43" lang="en">Soccer</category>
```

That setting needs a recent Tvheadend build, and until two fixes are merged
upstream it can crash Tvheadend. Read [Choosing a format](#choosing-a-format)
before you use `eit`.

Nothing else in the document changes. Output is byte-identical to the input
apart from the inserted attributes or replaced categories.

This first release ships one table, for the Schedules Direct vocabulary.
Guides from other sources use other terms and need their own table. New
tables are welcome; see [Tables](#tables).

## Install

The [releases page](https://github.com/bernhardberger/xmltv-dvb-genres/releases)
has binaries for Linux on amd64, arm64 and armv7. Each is a single static
binary with no runtime dependencies. Install it wherever the grabber runs, for
example on a Raspberry Pi:

```sh
base=https://github.com/bernhardberger/xmltv-dvb-genres/releases/latest/download
curl -LO "$base/xmltv-dvb-genres-linux-arm64" -LO "$base/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
sudo install -m 755 xmltv-dvb-genres-linux-arm64 /usr/local/bin/xmltv-dvb-genres
```

With Go 1.22 or later you can build it yourself:

```sh
go install github.com/bernhardberger/xmltv-dvb-genres/cmd/xmltv-dvb-genres@latest
```

## Use

```sh
tv_grab_something --days 7 | xmltv-dvb-genres > guide.xml
tv_grab_something --days 7 | xmltv-dvb-genres -format eit > guide.xml   # patched Tvheadend only, see below
xmltv-dvb-genres -report guide-in.xml > guide.xml   # statistics and unmapped terms on stderr
xmltv-dvb-genres -check ./mine.json guide.xml       # lint a table, optionally against a real guide
xmltv-dvb-genres -list-tables
xmltv-dvb-genres -list-codes
```

| Flag | Meaning |
| --- | --- |
| `-format name\|eit` | How to write the code: one Tvheadend genre name (default) or `eit` attributes. See [Choosing a format](#choosing-a-format). |
| `-table name\|file.json` | Rule table to use. Repeat to chain tables; the first table with a matching rule wins. Default `schedulesdirect-en`. |
| `-report` | Print per-code counts and the category terms of unmapped programmes to stderr. |
| `-check table [guide.xml]` | Lint a table and exit (see below). |
| `-list-tables`, `-list-codes`, `-version` | Print information and exit. |

The tool **fails open**. If the input is malformed, it maps everything up to
the error and copies the rest through unchanged. If a table cannot be loaded,
it passes the guide through untouched. The same applies to an unknown
`-format`. Either way it reports the problem on stderr and exits with 1. A
guide is never lost because of this tool.

Running the tool twice is harmless. Programmes that already have an `eit`
attribute on any category are left alone, and so are programmes whose only
category is already a Tvheadend genre name.

### Choosing a format

Use `name` unless your Tvheadend build has the two fixes listed below.

With `name`, Tvheadend gets exactly the mapped genre for every mapped
programme, except for the few codes without a Tvheadend name (see below).
It works with any Tvheadend version and no setting, and the output stays
valid XMLTV. The original category text is gone, though: "Crime drama",
"Mystery" and "Series" all become "Detective / Thriller". Kodi does not
notice, because it shows the genre code and only falls back to category text
when there is none. Anything else that reads the same guide loses the source
vocabulary too.

`eit` keeps the original categories. With the grabber module's **Scrape
credits and extra information** option on, Tvheadend stores their text as
well, and its web UI shows and filters on it. `eit` is a Tvheadend convention,
not part of the XMLTV DTD, so strict validators such as `tv_validate_file`
will flag it. It needs the Category Code XPath setting, which has problems in
upstream Tvheadend:

- The setting reached Tvheadend master on 26 October 2025
  ([#1962](https://github.com/tvheadend/tvheadend/pull/1962)). Tvheadend 4.2
  and older 4.3 builds don't have it and ignore the attribute.
- Without [#2345](https://github.com/tvheadend/tvheadend/pull/2345), Tvheadend
  can crash when one XMLTV module has a Category Code XPath and another XMLTV
  module without one imports a guide
  ([#2344](https://github.com/tvheadend/tvheadend/issues/2344)). Clearing the
  field doesn't help, because Tvheadend still parses the empty value. To
  recover, stop Tvheadend, remove the `xpath_*` keys from its `epggrab/config`
  file and start it again.
- Without [#2363](https://github.com/tvheadend/tvheadend/pull/2363), Tvheadend
  takes the code from a programme's first category only. On each further
  category it finds the code already listed, treats that as an unusable code
  and matches the category text instead. A later category whose text is a
  Tvheadend genre name therefore adds a second genre, and Kodi shows whichever
  code is lower. With the Schedules Direct guide this hit 7% of programmes,
  mostly "Comedy" on cartoons and "Documentary" on nature and science
  documentaries ([#2362](https://github.com/tvheadend/tvheadend/issues/2362)).

In October 2026 both fixes were still open. Check their status before you
switch to `eit`.

Codes from `0xb0` up have no Tvheadend name. In `name` format, programmes
mapped to them keep their categories, and `-report` counts them.

### Tvheadend setup

Tvheadend runs grabbers without a shell, so none of its grabber settings can
hold a pipe. Where the filter goes depends on how the guide reaches
Tvheadend. In every case the filter writes the whole guide, even when it
reports an error, so don't make the next step depend on its exit status.

#### External XMLTV socket

This is Tvheadend's documented way to feed it from scripts and cron jobs.
Enable the *External: XMLTV* module in *Configuration → Channel / EPG → EPG
Grabber Modules*. Its Path field shows the socket it listens on. Run the
grabber from cron and pipe its output through the filter into the socket:

```sh
tv_grab_something --days 7 | xmltv-dvb-genres | socat - UNIX-CONNECT:/path/to/epggrab/xmltv.sock
```

#### Internal grabber module

Tvheadend runs grabbers named `tv_grab_*` on its own schedule. The module's
Path is read-only, and Tvheadend passes Extra arguments to the grabber as
plain arguments, so a `|` there reaches the grabber as text. Install a wrapper
under a `tv_grab_*` name instead. It runs the real grabber and then the
filter; see [`examples/tv_grab_wrapper`](examples/tv_grab_wrapper). Tvheadend
finds new grabbers when it starts.

This binary is deliberately not named `tv_grab_*`. Tvheadend would detect it
as a grabber of its own.

#### Guide files

If one job writes a guide file and Tvheadend reads it later, for example with
`tv_grab_file`, filter the file in the same job and point Tvheadend at the
result:

```sh
xmltv-dvb-genres guide.xml > guide-coded.xml
```

#### Category Code XPath

`-format name` needs no setting. For `-format eit`, on a build with the fixes
from [Choosing a format](#choosing-a-format), select the module that receives
the guide and set **Category Code XPath** to `@eit`. The internal and external
XMLTV modules both have this setting.

Then run the grabber again. Clients such as Kodi show a genre for each mapped
programme, in their own language.

## Why one code per programme

Tvheadend sorts a programme's genres by code, and Kodi displays only the
lowest. If every category got its own code, a minor tag could beat the real
genre. "Series", "Sports event" and "Soccer" would display as a general sports
programme rather than football. Each table therefore ranks its rules: specific
rules come before generic ones, and the first match decides the code for the
whole programme.

## Tables

| Table | Source | Language |
| --- | --- | --- |
| `schedulesdirect-en` | [Schedules Direct](https://schedulesdirect.org) (Gracenote) via tv_grab_zz_sdjson, EpGo and similar | `en` |

On a 35,754-programme guide (Austrian, German, British and US lineups),
`schedulesdirect-en` maps 98.4% of programmes. The rest only carry generic tags
such as "Series", "Special" or "Off Air". The reasoning behind its rules is in
[docs/schedulesdirect-en.md](docs/schedulesdirect-en.md).

A table is a JSON file:

```json
{
  "name": "mysource-de",
  "description": "What produces this vocabulary.",
  "lang": "de",
  "rules": [
    {"code": "0x43", "any": ["fußball"]},
    {"code": "0x55", "all": ["kinder"], "any": ["zeichentrick", "animation"]},
    {"code": "0x40", "any": ["sport"], "none": ["magazin"]}
  ],
  "ignore": ["serie"]
}
```

- Terms compare case-insensitively with whitespace collapsed. The programme's
  whole category set is tested at once.
- `all`: every term present; `any`: at least one present; `none`: none present.
  A rule needs `all` or `any`.
- `lang` restricts the table to categories with that `lang` attribute.
  Categories without one always take part.
- `ignore` lists terms that carry no genre. They are left out of `-report`.
- Use codes from `-list-codes` between `0x10` and `0xaf`. Tvheadend has no names
  for the rest.

### Checking a table

Loading already rejects malformed tables: unknown fields, codes not defined by
DVB, rules without `all`/`any`, empty terms. `-check` also finds rules that are
valid but cannot work as written:

- a term that an earlier rule always catches first (shadowed), or a rule that
  can never be reached;
- a term in both `all`/`any` and `none` of a rule, or listed twice;
- codes Tvheadend has no name for;
- `ignore` terms that a rule also uses.

Given a guide, it also lists rule terms that never occur in it (often typos)
and guide terms that no rule or `ignore` entry covers, with up to three
programme titles for each. Warnings exit with 1; these guide findings are
informational.

```console
$ xmltv-dvb-genres -check ours.json
ours.json: warning: rule 2: term "sport" is shadowed by rule 1 and never decides this rule
ours.json: warning: rule 3: code 0xb3 has no Tvheadend name; clients will show no genre
ours.json: 3 rules, 2 warnings
```

Pass a local file with `-table ./mine.json`. List it before an embedded table
to override individual rules.

New tables and corrections are welcome. Run `-check table.json guide.xml` and
`-report` on a real guide, add rules for the most frequent uncovered terms, and
include a few cases in `genremap/genremap_test.go`. The tests lint every
embedded table and fail on warnings. A new table should come with a short page
in `docs/` that explains its non-obvious choices.

## Library

Grabbers written in Go can map terms directly, without XMLTV:

```go
tab, _ := genremap.Load("schedulesdirect-en")
res := genremap.New(tab).MapTerms("Series", "Soccer", "Sports event")
// res.OK == true, res.Code == 0x43, res.Code.Name() == "football/soccer"
```

Package `etsi` contains the content-code table from EN 300 468 (Table 29).
Package `tvh` contains the English genre names Tvheadend matches category text
against, and its matching rule. Package `xmltv` contains the streaming
rewriter. Runnable examples for
mapping, overriding a table, linting and rewriting a guide are in
`example_test.go` in each package and show up in the package documentation.

## Licence

MIT. The genre names in package `etsi` are taken from ETSI EN 300 468. Package
`tvh` lists the genre names Tvheadend matches, so that the output works with it.
