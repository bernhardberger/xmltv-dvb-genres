# Tvheadend setup

Tvheadend runs grabbers without a shell, so none of its grabber settings can
hold a pipe. Where the filter goes depends on how the guide reaches
Tvheadend. In each setup the filter writes the whole guide, even when it
reports an error, so don't make the next step depend on its exit status.

## External XMLTV socket

This is Tvheadend's documented way to take a guide from scripts and cron jobs.
Enable the *External: XMLTV* module in *Configuration → Channel / EPG → EPG
Grabber Modules*. Its Path field shows the socket it listens on, which is
`epggrab/xmltv.sock` in Tvheadend's configuration directory. Run the grabber
from cron and pipe its output through the filter into the socket:

```sh
tv_grab_zz_sdjson --days 7 --quiet | xmltv-dvb-genres | socat - UNIX-CONNECT:/config/epggrab/xmltv.sock
```

For a grabber that writes a file, filter the file instead. EpGo writes to the
file set as `XMLTV` under `Files` in its config, here `epgo.xml`:

```sh
epgo -config epgo.yaml && xmltv-dvb-genres epgo.xml | socat - UNIX-CONNECT:/config/epggrab/xmltv.sock
```

## Internal grabber module

Tvheadend looks for programs named `tv_grab_*` on its PATH when it starts and
offers each one as an *XMLTV* module that it runs on its own schedule. The
module's Path can't be edited, and Tvheadend passes Extra arguments to the
grabber as plain arguments, so a `|` there reaches the grabber as text.

Install a wrapper under a `tv_grab_*` name instead.
[`examples/tv_grab_wrapper`](../examples/tv_grab_wrapper) runs
`tv_grab_zz_sdjson` and then the filter:

1. Copy it to `/usr/local/bin/tv_grab_zz_sdjson_genres` and make it
   executable. For another grabber, change `REAL` and `DESCRIPTION` at the top.
2. Restart Tvheadend. The modules list now has *XMLTV: Schedules Direct JSON
   API with DVB genres*.
3. Give the new module the Extra arguments of the old one, enable it and
   disable the old one.
4. Tvheadend treats the wrapper as a new module with its own channel list.
   After its first grab, check that your channels use it as their EPG source.

Don't rename the filter itself to `tv_grab_*`. Tvheadend would then run it as
a grabber.

## Guide files

If one job writes a guide file and Tvheadend reads it later, for example with
`tv_grab_file`, filter the file in the same job and point Tvheadend at the
result:

```sh
xmltv-dvb-genres guide.xml > guide-coded.xml
```

## The name format

`-format name`, the default, replaces the categories of each mapped programme
with one category holding Tvheadend's name for the genre. Every Tvheadend
version since at least 4.2 matches that name without a setting, and the
output stays valid XMLTV.

The original category text is gone, though. "Crime drama", "Mystery" and
"Series" all become "Detective / Thriller". Kodi doesn't notice, because it
shows the genre and only falls back to category text when there is none.
Anything else that reads the same guide loses the source vocabulary too.

DVB codes from `0xb0` up have no Tvheadend name. Programmes mapped to them
keep their categories, and `-report` counts them. The shipped table doesn't
use these codes.

## The eit format

`-format eit` keeps the categories and adds the code to each of them:

```xml
<category eit="0x43" lang="en">Soccer</category>
```

With the module's *Scrape credits and extra information* option on,
Tvheadend stores the category text as well, and its web UI shows and filters
on it. `eit` is a Tvheadend convention, not part of the XMLTV DTD, so strict
validators such as `tv_validate_file` flag it.

Tvheadend reads the attribute only with *Category Code XPath* set to `@eit`
on the module that receives the guide. Both the internal and the external
XMLTV modules have the setting. Run the grabber again after setting it.

The setting has problems in upstream Tvheadend:

- It reached Tvheadend master on 26 October 2025
  ([#1962](https://github.com/tvheadend/tvheadend/pull/1962)). Tvheadend 4.2
  and older 4.3 builds don't have it and ignore the attribute.
- Without [#2345](https://github.com/tvheadend/tvheadend/pull/2345),
  Tvheadend can crash when one XMLTV module has a Category Code XPath and
  another XMLTV module without one imports a guide
  ([#2344](https://github.com/tvheadend/tvheadend/issues/2344)). Clearing the
  field doesn't help, because Tvheadend still parses the empty value. To
  recover, stop Tvheadend, remove the `xpath_*` keys from its
  `epggrab/config` file and start it again.
- Without [#2363](https://github.com/tvheadend/tvheadend/pull/2363),
  Tvheadend takes the code from a programme's first category only and
  matches the text of the others. A later category whose text is a Tvheadend
  genre name then adds a second genre, and Kodi shows whichever code is
  lower. With a Schedules Direct guide this hit 7% of programmes, mostly
  "Comedy" on cartoons and "Documentary" on nature and science documentaries
  ([#2362](https://github.com/tvheadend/tvheadend/issues/2362)).

In October 2026 both fixes were still open. Check their status before you
switch to `eit`, or run a Tvheadend build that includes them.

## Running the filter twice

Filtering a guide twice gives the same result as filtering it once.
Programmes that already have an `eit` attribute on any category are left
alone, and so are programmes whose only category is already a Tvheadend
genre name.
