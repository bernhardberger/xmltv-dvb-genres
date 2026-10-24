# The schedulesdirect-en table

This page explains why
[`schedulesdirect-en`](../genremap/tables/schedulesdirect-en.json) maps the
way it does. Read it before changing a rule, so a fix for one programme does
not quietly break another.

## Where the vocabulary comes from

Schedules Direct resells Gracenote listings. Grabbers such as
tv_grab_zz_sdjson and EpGo write Gracenote's genre list as
`<category lang="en">` elements. A programme usually has two to five distinct
ones, mixing a genre ("Crime drama"), a format ("Series", "tvshow"), a topic
("Medical") and sometimes a scheduling tag ("Holiday").

The vocabulary is American English on every lineup, including Austrian and
German ones. A few German terms come through from German-language sources:
"Dokusoap", "Filmreihe" and "Komödie". The table maps them too.

American usage matters most for sport. "Soccer" is association football and
"Football" is American football. DVB code 0x45, "team sports (excluding
football)", uses football in the European sense, so American football belongs
there, next to rugby and basketball.

## How rules are ordered

The first matching rule decides the one code for the whole programme (the
README explains why there is only one). Order is therefore the main design
tool. The table runs in these groups:

1. Non-programmes: teleshopping, adult content.
2. Sport, specific sports before "Sports event".
3. Children's programmes.
4. News.
5. Documentaries, with a topic before plain "Documentary".
6. Leisure topics such as cooking and travel.
7. Show formats: talk, game show, variety, reality.
8. Fiction, most specific genre first.
9. Music and arts.
10. Factual topics without a documentary tag.
11. Fallbacks for generic film and entertainment tags.

Within a group, a rule that needs two terms comes before the rule for either
term alone.

## Decisions by group

### Non-programmes first

Teleshopping is the third most frequent code in the sample guide. Gracenote
tags it with whatever is being sold: "Consumer", "Fashion", "Cooking", "Talk",
"Holiday". If it did not come first, an infomercial for pans would show as a
cooking programme. "Shopping", "Paid programming", "Auction" and
"Collectibles" all map to 0xa6. Adult content ("Erotic", "Adults only") comes
next for the same reason.

### Sport

Each specific sport maps to its DVB code before the generic tags are
considered, so "Soccer, Sports event" is football (0x43), not sport in general
(0x40). DVB has no code for cycling, golf, darts, snooker, table tennis,
climbing or esports; they get 0x40. Professional wrestling is staged, but
martial sports (0x4b) is still the closest code. "Multi-sport event",
"Olympics" and "Summer Olympics" map to special events (0x41), but only when
no specific sport is tagged: "Olympics, Skiing" is winter sports (0x49).
"Sports talk" maps to sports magazines (0x42).

"Auto racing" is motor sport (0x47). "Auto" on its own marks car magazines and
maps to motoring (0xa3) in the leisure group. "Boat racing" and "Boat" are
split the same way.

### Children

"Children" with "Animated" or "Anime" is cartoons (0x55). With "Educational"
it is school programming (0x54). Anything else tagged "Children" is general
children's programming (0x50). Animation without the "Children" tag, such as
The Simpsons or Family Guy, is mapped by its fiction tags instead. Only if none
match does the "Animated" fallback at the end give it 0x55.

### News

"Weather" comes before "Newsmagazine", so a weather bulletin stays a weather
report (0x21) even if it also has a magazine tag. "Newsmagazine" comes before
"News", so magazines get 0x22 rather than plain news. "Public affairs" maps to
current affairs (0x20).

### Documentaries

A documentary with a topic gets the topic's code: nature and animals 0x91,
science and technology 0x92, medicine and health 0x93, biography 0x83
(remarkable people). Other documentaries and docudramas get 0x23. This group
comes before fiction, so "Crime, Documentary" is a documentary and not a
thriller.

### Leisure before show formats

Gracenote tags cooking competitions with both "Cooking" and "Competition
reality". The topic says more about the programme than the format, so the
leisure rules run before the show-format rules and these programmes get
cooking (0xa5). Travel, house and garden, crafts, motoring, fitness, fashion
and outdoor hobbies work the same way.

### Show formats

"Talk" is 0x33, "Game show" and "Competition reality" are 0x31, "Variety" is
0x32. "Reality" and the German "Dokusoap" (scripted reality) map to the
general show code 0x30. Court shows such as Richter Alexander Hold carry
"Law, Reality" and end up here too.

### Fiction

"Sitcom" comes first. It is a format and outranks topic tags: without this
rule, Ghosts ("Mystery") came out as a thriller, M\*A\*S\*H ("Historical
drama") as a historical drama and Futurama ("Science fiction") as science
fiction.

After that the order is religious and historical drama (0x17), soap (0x15),
crime, mystery and thriller (0x11), science fiction, fantasy and horror
(0x13), action, adventure, western and war (0x12), comedy (0x14), romance
(0x16) and finally plain drama (0x10). A programme with several fiction tags
gets the first one in this order: "Action, Adventure, Crime drama" is a
thriller.

"Medical" and "Law" are topics, not genres. Medical and legal dramas also
carry "Drama" and map to 0x10 through it. Without "Drama", "Medical" marks
health magazines such as Bewusst gesund or Die Sprechstunde and maps to
medicine (0x93) in the factual group. "Law" alone maps to social and political
issues (0x80), but in practice almost every programme with "Law" matches an
earlier rule.

"Musical" and "Musical comedy" map to musical/opera (0x65) in the music group,
so The Rocky Horror Picture Show ("Feature Film, Musical comedy") is a
musical. A musical that also carries a fiction tag such as "Comedy" gets the
fiction code.

### Fallbacks

The last three rules catch programmes that only have generic tags. "Animated"
gets 0x55. "Feature film", "Movie", "TV movie", "Miniseries", "Short film" and
"Filmreihe" get movie/drama (0x10). "Entertainment" gets the general show code
(0x30).

### Ignored terms

"Series", "tvshow", "Special", "Event", "Holiday", "Family", "Highlights",
"TBA", "Off Air", "Static message", "Podcast" and "Anthology" describe a
format, an audience or a schedule slot, not a genre. They are in `ignore`, so
`-report` and `-check` do not list them as missing. A programme with only these
tags gets no code. "Anthology" was a drama term until it turned out to send
the lifestyle magazine Euromaxx to movie/drama.

### Codes the table avoids

Tvheadend only has names for codes 0x10 to 0xaf. Codes above that range, such
as the special characteristics in 0xb0 to 0xbf, would show no genre in clients,
so the table does not use them. `-check` warns about them.

## Known imperfect results

Ugly Americans ("Animated, Comedy, Horror") maps to horror (0x13), because
horror comes before comedy and the series has no "Sitcom" tag. 9-1-1 ("Drama,
Action") maps to action (0x12). Both are defensible, and fixing them would
break more programmes than it helps.

## Coverage

Measured on 22 October 2026 on a guide with Austrian, German, British and US
lineups (35,754 programmes):

| Result | Programmes |
| --- | ---: |
| Mapped | 35,189 (98.4%) |
| Only ignored tags | 565 (1.6%) |

The most frequent codes were news (0x21, 8,386), documentary (0x23, 3,357),
shopping (0xa6, 2,339), detective/thriller (0x11, 2,214) and show (0x30,
2,001).

## Changing the table

1. Run `xmltv-dvb-genres -report guide.xml > /dev/null` on a current guide and
   look at the unmapped terms.
2. Edit the rule and run `xmltv-dvb-genres -check schedulesdirect-en.json
   guide.xml` with the edited file. It reports shadowed terms, unreachable
   rules and guide terms that nothing covers.
3. Look at real programmes that carry the terms you touched, not just at the
   counts. A rule that moves a thousand programmes to the right code can move
   fifty others to a wrong one.
4. Add the programme's tag set as a case in `TestSchedulesDirectMapping` in
   `genremap/genremap_test.go`, and record the reason here if it is not
   obvious.
