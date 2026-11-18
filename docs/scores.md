# Scores

A **score** is a list of notes with a start and a length, in steps: what a MIDI file is. It is the other way of making music than a [generator](generators.md), which gives a value at every step. A score is made whole when the patch is built, and then goes round.

## Playing one

A score plays on any synth, where the synth takes notes:

```harp
tune = phrase([60 62 64 -1 67], [2 1 1 2 4])
out = subtractive(tune, preset: "pluck", voices: 4)
```

`phrase(pitches, durations)` makes one from a list of pitches (below 0 is a rest) and durations in steps. `velocity` and `legato` can be lists too, a value for each pitch. Notes that start on the same step sound together. The score goes round after its last note (`pad(score, steps)` sets how long it is).

For `play`, which takes the rhythm and the numbers apart, take them from the score: `onsets(s)` is the rhythm, `chordsof(s)` the notes that start on each step as a [chord](play.md#chords) (a chord field: give `play` a voice for each note), and `pitches(s)`, `lengths(s)` and `velocities(s)` one number for each step (the highest note, when notes start together).

```harp
fn inst(gate, n) { sine(hz(n)) * gate * 0.2 }
s = phrase([60 62 64 65], [2])
out = play(inst, rhythm: onsets(s), legato: 0.5, fields: [pitches(s)]).dup()
```

With chords: `play(inst, rhythm: onsets(s), fields: [chordsof(s)], voices: 4)`.

## Where scores come from

| | |
|---|---|
| `phrase(pitches, durations)` | a line from lists |
| `midifile("song.mid", track: 0)` | a MIDI file, in the steps of the patch (the tempo is the patch's). `track: -1` takes all tracks; `harp check` says how many there are |
| `fugue(subject, durations, voices: 3, root, scale, voice: 0)` | the exposition of a fugue with a countersubject written by the rules of counterpoint. `voice` picks one voice for a synth of its own |
| `harmonize(root, degrees: [0 3 4 0])` | chords written for four voices (soprano, alto, tenor, bass) by the rules of four part writing: no parallel fifths or octaves, no crossing, a leading tone that rises, a seventh that falls. Without `degrees` it makes `n` chords by the grammar of tonic, subdominant and dominant, in phrases that end in a cadence. `voice: 0` to `3` gives a voice its own synth. `harp check` says the chords and any rule that had to be broken |
| `isorhythm(talea, color, length)` | two patterns of different lengths going round together |
| `chordtones(chords, root, scale)` | a melody over chords, a chord tone on each beat and steps of the scale between |
| `phasing(pattern, drift)` | two voices, one a little faster each round, as in Steve Reich's piano phase |
| `canon(score, delay, interval, voices)` | the score again, later and higher, in each voice |

Several of these use chance (`fugue`, `chordtones`): `seed = 3` at the top makes them the same every time.

A chorale in four parts, one synth for each voice:

```harp
seed = 3
fn part(v) { modal(harmonize(a3, mode: "minor", n: 8, cadence: "authentic", voice: v), preset: "marimba", voices: 6) }
out = (part(0) + part(1) + part(2) + part(3)).gain(0.25)
```

## Changing a score

`transpose`, `stretch` (longer or shorter), `mirror` (upside down), `retro` (backwards), `shift`, `cut`, `tile` (repeat), `pad`, `tune` (onto a scale), `thin` (leave notes out by a rhythm), and `together([a, b])` to join scores that play at once.

```harp
theme = phrase([60 62 64 67], [2 2 2 4])
piece = together([theme, shift(transpose(retro(theme), 7), 4)])
out = modal(piece, preset: "marimba", voices: 6)
```

## Not here

A score is not changed while it plays the way a generator is: a live edit that changes one starts it from its beginning. Writing MIDI files and the pitch-bend output of `microtonal` are not done.

An example: `examples/scores.harp`.
