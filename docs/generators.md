# Generators

A generator makes a value for every step of the clock: a rhythm (true or false), a note, a chord. They run on their own thread, ahead of the audio, and can be changed or shared without touching the sample loop.

## The clock

`bpm` is beats a minute, `steps` is steps in a beat (default 4: a step is a sixteenth). At 120 bpm a step is 125 ms.

## Making them

| Function | Gives |
|---|---|
| `euclid(steps, pulses, rotation)` | a euclidean rhythm: `pulses` hits spread over `steps`, turned by `rotation` |
| `every(n)` | true every n-th step |
| `[true false true false]` | a list of booleans is a rhythm that repeats |
| `mutate([7 9 11], swap: 0.4)` | a phrase that swaps notes around now and then (whole numbers: scale degrees) |
| `degrees(a3, minor, phrase)` | scale degrees as MIDI notes |
| `melody(pitches, rhythm)` / `chords(chords, rhythm)` | notes on a rhythm, for the synths |
| `notes(melody, rhythm, length: 0.6)` | a voice: `.pitch` (Hz) and `.gate`, the pitch only moves when a note is played |

## Using them

**As a signal.** Anywhere a signal is wanted a generator steps, a new value every step:

```
phrase = degrees(a3, minor, mutate([0 2 4 7], swap: 0.4))
cutoff = glide(hz(phrase + 12), ms: 60)
```

- `seq(values, glide: ms)` makes a stepping signal of a list or a generator; `glide: 60` moves to each new value in about 60 ms.
- `glide(x, ms)` smooths any jumping signal (psalm's smoother). Use it on a stepping control so it does not click.
- `pulses(rhythm, length: 0.5)` makes a gate: high for part of the step on every hit. Use it with `adsr(...).on(gate)`.

**As notes.** `notes` gives a voice for oscillators; `subtractive` and `supersaw` take `chords(...)` or `melody(...)`; `play` plays an instrument you write ([play](play.md)).

## One stream or two

A generator with a **name** is one stream:

```
p = mutate([0 2 4 7], swap: 0.5)
out = [lowPass(x, cutoff: hz(p + 60)), pan(y, p / 10)]   // the same values in both places
```

Written twice, `mutate(...)` is two independent streams. This holds for generators used as signals and for `pulses` with the same rhythm and length.

## Random numbers

`seed = 21` makes the patch sound the same every time it plays. Two kinds of random numbers:

- **at build time**, once, when the patch is read: `rand(lo, hi)`, `expRand(lo, hi)`, `coin(p)`, `choose(list)`. Used in `fn` and `if` to build a patch that is different for every seed.
- **while it plays**: `mutate` draws as it goes.

With a `seed`, a patch that is read again (a live edit) draws the same build numbers again, so saving a file changes nothing unless you changed something. Without one, harp picks a seed when the patch first loads and keeps it for the edits after.
