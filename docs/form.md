# The form of a piece

A **song** is the sections of a piece with their bars: `song("Intro 4, Verse 8, Chorus 8, Outro 4")`. A bar is 16 steps (`barSteps:` changes that). The song goes round when it has gone through. It gives signals and rhythms that follow it, so that a layer comes and goes without a plan written by hand.

```harp
bpm = 100
steps = 4
form = song("Intro 4, Verse 8, Chorus 8, Outro 4")
pad = subtractive(chords(changes("Am F C G", key: "A minor"), every(16), length: 15), preset: "pad", voices: 8)
beat = drumSynth(melody([drum("kick")], pattern("x... x... x... x..x"), velocity: 0.9), kit: "909")
out = (pad.gain(0.3 * curve(form, "Intro: 0 -> 1 smooth, Outro: 1 -> 0")) + beat.gain(0.5 * during(form, "Verse, Chorus"))).reverb(mix: 0.2, decay: 2.5)
```

| | |
|---|---|
| `during(form, "Verse, Chorus")` | a signal that is 1 in those sections and 0 elsewhere: a layer in and out through `.gain(...)`. `"not Intro"` is all but one, `"all"` every section. `glide:` is how many steps the change takes around the edge (2) |
| `curve(form, "Intro: 0 -> 1 smooth, Verse: 0.8, Outro: 1 -> 0")` | a value that moves through the sections: a value (held), or two with an arrow; `lin` (the default), `exp` (between values above 0), `smooth` or `step` after it. A section with no curve keeps where the one before ended; sections with the same name share the curve |
| `bySection(form, ["Intro, Verse", "Chorus"], [a, b])` | other rhythms in other sections: the first in the intro and the verse, the second in the chorus (a section that is in no name plays nothing). It takes a value from the generator of a section for every step, so it is for rhythms; `restart: true` starts a generator from its start whenever its section comes |

Sections can have the same name (`Verse` twice); `during` and `curve` take all of them.

`%fill` in a [pattern](patterns.md) plays a hit only in the last bar of a section: give the pattern the song (`pattern("x . x x%fill", song: form)`).

The song is a part of the patch like any other and is edited while it plays: add a bar or rename a section, and what follows it changes at the next step.
