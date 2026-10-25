# xmltv-dvb-genres

Map XMLTV category text to DVB (EIT) genre codes for Tvheadend.

Tvheadend only gives an XMLTV programme a genre when one of its categories is
exactly one of Tvheadend's genre names, such as "Movie / Drama". Guide sources
use their own words. Schedules Direct says "Crime drama", "Soccer" or "Sports
event", and none of these match, so only about a quarter of its programmes get
a genre in Tvheadend and in clients such as Kodi.

`xmltv-dvb-genres` sits between the grabber and Tvheadend. It picks one DVB
genre per programme from a rule table for the guide's vocabulary and writes it
in a form Tvheadend understands:

```xml
<!-- from the grabber -->
<category lang="en">Sports event</category>
<category lang="en">Soccer</category>

<!-- to Tvheadend -->
<category lang="en">Football / Soccer</category>
```

On a 35,439-programme Schedules Direct guide it maps 98.3% of programmes. The
rest only carry generic tags such as "Series" or "Special". Nothing else in the
guide changes.

## Requirements

- Tvheadend, any version. The default output needs no Tvheadend setting.
- A guide from [Schedules Direct](https://schedulesdirect.org), for example
  from `tv_grab_zz_sdjson` (part of [XMLTV](https://github.com/XMLTV/xmltv)) or
  [EpGo](https://github.com/Chuchodavids/EpGo). This release ships a table for
  that vocabulary only. Other sources need their own table, and new tables are
  welcome; see [docs/tables.md](docs/tables.md).

## Install

The [releases page](https://github.com/bernhardberger/xmltv-dvb-genres/releases)
has static binaries for Linux on amd64, arm64 and armv7. For example, on a
Raspberry Pi:

```sh
base=https://github.com/bernhardberger/xmltv-dvb-genres/releases/latest/download
curl -LO "$base/xmltv-dvb-genres-linux-arm64" -LO "$base/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
sudo install -m 755 xmltv-dvb-genres-linux-arm64 /usr/local/bin/xmltv-dvb-genres
```

With Go 1.22 or later you can build it instead:

```sh
go install github.com/bernhardberger/xmltv-dvb-genres/cmd/xmltv-dvb-genres@latest
```

## Quick start

Feed a grabber's output through the filter into Tvheadend's *External: XMLTV*
socket, for example from cron. The module's settings show the socket path; in
the LinuxServer Docker image it is `/config/epggrab/xmltv.sock`.

```sh
tv_grab_zz_sdjson --days 7 --quiet | xmltv-dvb-genres | socat - UNIX-CONNECT:/config/epggrab/xmltv.sock
```

EpGo writes its guide to a file, set as `XMLTV` under `Files` in its config,
so filter that file:

```sh
epgo -config epgo.yaml
xmltv-dvb-genres epgo.xml | socat - UNIX-CONNECT:/config/epggrab/xmltv.sock
```

If Tvheadend runs the grabber itself, install the
[wrapper](examples/tv_grab_wrapper) instead.
[docs/tvheadend.md](docs/tvheadend.md) covers both setups.

The guide always gets through. If the input is malformed or a table can't be
loaded, the tool passes the rest of the guide through unchanged, reports the
problem on stderr and exits with 1.

## Options

| Flag | Meaning |
| --- | --- |
| `-format name\|eit` | `name` (default) replaces the categories with one Tvheadend genre name. `eit` keeps them and adds `eit="0xNN"`; read [Formats](#formats) first. |
| `-table name\|file.json` | Rule table to use, default `schedulesdirect-en`. Repeat to chain tables; the first match wins. |
| `-report` | Print per-genre counts and the terms of unmapped programmes to stderr. |
| `-check table [guide.xml]` | Lint a rule table, optionally against a real guide, and exit. |
| `-list-tables`, `-list-codes`, `-version` | Print information and exit. |

## Formats

| | `name` (default) | `eit` |
| --- | --- | --- |
| Tvheadend | any version, no setting | Category Code XPath set to `@eit`, plus two fixes that upstream hasn't merged yet |
| Original categories | replaced by one genre name | kept; Tvheadend can store and search them |
| Genres per programme | exactly one | one, or two on Tvheadend without the fixes |
| Output | valid XMLTV | `eit` is not in the XMLTV DTD |

Use `name` unless you need the original category text in Tvheadend. On
upstream Tvheadend as of October 2026, `eit` can crash Tvheadend
([#2344](https://github.com/tvheadend/tvheadend/issues/2344)).
[docs/tvheadend.md](docs/tvheadend.md#the-eit-format) explains the risks and
which builds are safe.

## Guides

- [Tvheadend setup](docs/tvheadend.md): the socket, the grabber wrapper,
  guide files and both formats in detail.
- [Rule tables](docs/tables.md): the table format, `-check`, overriding
  rules and contributing a table.
- [The Schedules Direct table](docs/schedulesdirect-en.md): why its rules map
  the way they do.

## Library

Grabbers written in Go can map terms directly, without XMLTV:

```go
tab, _ := genremap.Load("schedulesdirect-en")
res := genremap.New(tab).MapTerms("Series", "Soccer", "Sports event")
// res.OK == true, res.Code == 0x43, res.Code.Name() == "football/soccer"
```

Package `etsi` has the genre table from ETSI EN 300 468, `tvh` the genre names
Tvheadend matches, and `xmltv` the streaming rewriter. The documentation of
`genremap` and `xmltv` has runnable examples.

## Licence

MIT. The genre names in package `etsi` are taken from ETSI EN 300 468. Package
`tvh` lists the genre names Tvheadend matches, so that the output works with it.
