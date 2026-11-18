# Types and channels

harp keeps two rates apart, and the checker tells you when they are mixed up by mistake.

- **Signals** are audio rate: a value for every sample.
- **Generators** are event rate: a value for every *step* of the clock (`bpm`, `steps`). They are pulled ahead of the audio on their own thread and never run in the audio loop.

## Kinds of values

| Kind | Example |
|---|---|
| number | `440`, `c4` |
| boolean | `true`, `coin(0.5)` |
| string | `"pad"` |
| signal | `saw(110)` |
| list | `[7 9 11]` |
| generator of numbers / whole numbers / booleans / chords / notes | `degrees(...)`, `mutate(...)`, `euclid(...)`, `chords(...)` |
| envelope | `adsr(0.01, 0.1, 0.5, 0.2)` |
| voice | `notes(...)` (has `.pitch` and `.gate`) |
| scale | `minor` |
| score | `phrase(...)`, `midifile(...)`: notes with a start and a length ([Scores](scores.md)) |
| function | a `fn` of yours |

## What can be used where

Where a function wants one kind, a value of another is converted if the language says so:

| Given | Where wanted | Becomes |
|---|---|---|
| number | signal | a constant signal |
| list | signal | a signal with one channel per item |
| number | generator | the same value every step |
| list | generator | the items, again and again |
| score | generator of notes | the notes of each step, again and again (for a synth) |
| generator of numbers | signal | a signal that steps: a new value every step, held |

Anything else is an error that names the kinds: `lowpass: cutoff: expected a signal, got a string`.

A generator that has a **name** is one stream: used as a signal in several places it is pulled once a step and every place has the same values. Two separate calls (`mutate(...)` twice) are two streams. See [Generators](generators.md).

## Channels

A signal has one channel (mono), two (stereo) or more. A node that takes signals is made once per channel, and the rule is strict:

- one channel goes with any number of channels;
- otherwise the counts must be equal, and each channel is paired with its own;
- two and three channels together is an error when the patch is built.

Ways to change the number of channels, on purpose:

| Function | Does |
|---|---|
| `[a, b]` (a list of signals) | puts the signals side by side: stereo |
| `dup(x, 2)` | one channel into two |
| `mix(x)` | all channels added into one |
| `pan(x, pos)` | mono into stereo, `pos` from -1 to 1, equal power |
| `spread(x, width)` | n channels spread over the stereo field |

`saw([110, 110.5])` is two oscillators, one per frequency. `lowpass(stereo, cutoff: 800)` filters each side with its own filter, one cutoff for both.

`out` can be any number of channels; `harp play` sends mono to every output and a stereo patch to the first two outputs. A live edit cannot change the number of channels ([Live editing](live-editing.md)).
