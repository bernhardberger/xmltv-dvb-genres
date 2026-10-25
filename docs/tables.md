# Rule tables

A rule table maps one guide source's category vocabulary to DVB genre codes.
This release has one:

| Table | Source | Language |
| --- | --- | --- |
| `schedulesdirect-en` | [Schedules Direct](https://schedulesdirect.org), through `tv_grab_zz_sdjson`, EpGo and similar grabbers | `en` |

[schedulesdirect-en.md](schedulesdirect-en.md) explains its rules. Guides
from other sources use other words and need their own table.

## One code per programme

Tvheadend sorts a programme's genres by code, and Kodi displays only the
lowest. If every category got its own code, a minor tag could beat the real
genre: "Series", "Sports event" and "Soccer" would display as a general
sports programme rather than football. So the filter gives each programme one
code. A table ranks its rules, specific ones before generic ones, and the
first rule that matches decides.

## Format

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

A rule tests the programme's whole set of categories at once. It matches if
every `all` term is present, at least one `any` term is present and no `none`
term is present. Each rule needs `all` or `any`. Terms compare
case-insensitively, with runs of whitespace treated as one space.

`lang` limits the table to categories with that `lang` attribute. Categories
without one always take part.

`ignore` lists terms that carry no genre, such as "Series". `-report` and
`-check` leave them out of their lists of unmapped terms.

Use codes from `-list-codes` between `0x10` and `0xaf`. Tvheadend has names
only for those.

## Checking a table

Loading a table rejects unknown fields, codes that DVB doesn't define, rules
without `all` or `any`, and empty terms. `-check` also finds rules that load
fine but can't work as written:

- a term that an earlier rule always catches first, or a rule that can never
  match for that reason
- a term in both `none` and `all` or `any` of the same rule, or a term listed
  twice
- a code that Tvheadend has no name for
- an `ignore` term that a rule also uses

```console
$ xmltv-dvb-genres -check ours.json
ours.json: warning: rule 2: term "sport" is shadowed by rule 1 and never decides this rule
ours.json: warning: rule 3: code 0xb3 has no Tvheadend name; clients will show no genre
ours.json: 3 rules, 2 warnings
```

Given a guide as well, `-check` reports how many of its programmes the table
maps, rule terms that never occur in it, which are often typos, and guide
terms that no rule or `ignore` entry covers, with up to three programme titles
each:

```console
$ xmltv-dvb-genres -check mine.json guide.xml
mine.json: 34843 of 35439 programmes in guide.xml mapped by this table alone
mine.json: info: guide term "track/field" (3 categories) is in no rule and not ignored, e.g. "College Track and Field"
mine.json: 69 rules, 0 warnings
```

Warnings make `-check` exit with 1. The guide findings don't.

## Using your own table

Pass a file with `-table ./mine.json`. Repeat `-table` to chain tables; the
first table with a matching rule decides. To change a few rules of a shipped
table, put a small file of your own in front of it:

```sh
xmltv-dvb-genres -table ./fixes.json -table schedulesdirect-en guide.xml
```

## Contributing a table

New tables and corrections are welcome:

1. Run `-check` and `-report` against a real guide from the source.
2. Add rules for the most frequent uncovered terms, specific rules first.
3. Add a few cases to `genremap/genremap_test.go`. The tests lint every
   shipped table and fail on warnings.
4. Add a short page here in `docs/` that explains the table's non-obvious
   choices, like [schedulesdirect-en.md](schedulesdirect-en.md).
