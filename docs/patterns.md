# Patterns, parts and chords as text

Rhythms, melodies and chords can be written as text. The syntax is that of the patcher of psalm, so what is written there reads here.

## pattern

`pattern("x..x ..x. x...")` is a rhythm: a step to each character, `x` a hit, `.` a rest. Spaces, `|` and new lines are for reading. It goes round.

```harp
bpm = 110
steps = 4
kick = drumSynth(melody([drum("kick")], pattern("x..x ..x. x... ..x.")), kit: "909")
out = kick.gain(0.5)
```

| | |
|---|---|
| `x3` | a roll: three hits in one step (1 to 64) |
| `[x x x]` | a group that fills one step: a triplet. `[x . x x]` has rests, and groups can hold groups and rolls |
| `x*2` | it lasts 2 steps (`.*4` a rest of 4, `*0.5` half a step); the note of that hit is that long |
| `x?25` | by chance, 25% (`x?` is 50%) |
| `x%2/4` | a condition: only the second time through the pattern in every four (`%!2/4` the other three); `%first` only the first time, `%fill` only in the last bar of a section (give `song:`) |
| `x!` | it stays when the pattern mutates |
| `E(5,16)` | a euclidean rhythm (`E(5,16,2)` turned 2 steps); `A(3+3+2)` an additive one; `CA(30,16)` a row of a cellular automaton; `M(x . x x;16;3)` a Markov chain on an example |
| `E(5,16,+1)` | a function that goes on with the passes: `+1` one step more every pass, so the hits turn round; `CA(30,16,+1)` the automaton runs on, a new row every time |

`mutate: 0.5` changes the pattern from pass to pass (hits fall out and rests become hits, as many as a quarter of the steps at 1; the same `seed` makes the same changes), `freeze: 4` stops the changes after 4 passes, and `drift: 0.2` moves the timing along a smooth curve (up to 0.2 of a step late).

Plain steps and rests give a rhythm of booleans, like every other rhythm of harp. Anything more gives **hits**, which can fall anywhere in a step; `melody`, `chords` and `play` take them as their rhythm, and a synth starts a note at the moment of its hit. `maybe`, `both` and the other functions that join rhythms need plain ones.

## part

`part("C3 Eb3 G3 - Bb3 . G3 Eb3")` is a melody with its rhythm, as a [score](scores.md): a step each, a note by its name (`C3`, `Eb3`, `f#4`, C4 is 60) or its number.

| | |
|---|---|
| `.` | a rest |
| `-` | the note before held a step longer |
| `C3+Eb3+G3` | a chord |
| `[C3 D3 Eb3]` | a group in one step |
| `C3^3` | a roll of three |
| `C3*2`, `C3?25`, `C3%2/4`, `C3!` | as in a pattern. `%` conditions play the text as many times as they need (`passes:`) |
| `@1 @2 @3 @4` | the root, the next tone, the fifth, the seventh of the chord of that moment; `@-1` the tone below the root, `@b` the bass, `@c` the whole chord. `'` after one an octave up, `,` an octave down |

The chords come from `chords: "Am F C G"`, a bar each (`barSteps:`, 16). A figure written once follows them:

```harp
bpm = 100
steps = 4
prog = "Am F C G"
bass = part("@b . . @b . . @b . | @b . . @b . @b @b .", chords: prog, key: "A minor")
lead = part("@1' . @3 @1' . @2' @1' .", chords: prog, key: "A minor", register: 60)
out = (subtractive(bass, preset: "bass", voices: 2) + subtractive(lead, preset: "pluck", voices: 6).gain(0.5)).gain(0.5)
```

`register` (48) is the lowest note of `@1`, `bass` (36) of `@b`.

## changes

`changes("Dm7 G7*2 Cmaj7")` is a list of chords: symbols (`Dm7`, `Bb/D`, `F#m7b5`), or roman numerals in a `key` (`changes("i VI V7", key: "A minor")`). A chord lasts a bar, `*2` two bars, `-` holds the one before, `.` is no chord. They are in root position from `lowest` (48 is C3), or with `lead: true` each is the voicing nearest the one before.

```harp
notes = chords(changes("Am F C G", key: "A minor", lead: true), every(16), velocity: 0.6, length: 15)
out = subtractive(notes, preset: "pad", voices: 8).gain(0.4)
```

## Orders

`order([1 2 3 4], "converge")` plays the numbers of a list in an order, round and round: `in order`, `shuffled` (every value once before any again), `random`, `up and down`, `converge` (1 4 2 3), `alternate` (1 2 1 3 1 4), `backwards`. `pick(list)` is any one of them every time.
