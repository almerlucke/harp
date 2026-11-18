# play: notes on an instrument you write

`play` runs a function of yours as a new instrument for every note, the way a SuperCollider `Pbind` plays a `SynthDef`.

```harp
bpm = 110

fn pluck(gate, note, bright) {
  f = hz(note)
  saw([f * 0.997, f * 1.003])
    .lowpass(cutoff: bright, res: 0.25)
    * env(gate, [0 1 0], [0.003 0.4], -4) * 0.2
}

notes = play(pluck,
  rhythm: euclid(16, 9),
  legato: 0.7,
  fields: [degrees(a3, minor, mutate([0 2 4 7], swap: 0.3)), [1800 2800 2200]],
  voices: 3)

out = notes
```

The instrument gets **the gate** first, then **one signal for each field**, and gives a signal (any number of channels, the same for every voice).

## When the notes are

Give exactly one of:

| Parameter | Meaning |
|---|---|
| `dur: [1 0.5 0.5]` | the length of each note in beats, in turn (a list, or a generator of beats). Must be a whole number of steps |
| `rhythm: euclid(16, 5)` | a generator of booleans, pulled every step: a note on every true, as long as the steps to the next |

## How long a gate is held

| Parameter | Meaning |
|---|---|
| `legato: 0.8` | the part of the note that the gate is high (default 0.8) |
| `sustain: 2` | or a fixed length in beats |

## Swing, humanize and accent

Three parameters give the notes a feel. They go by the **step** a note starts on, counted from the start of the patch, so they are the same whatever the rhythm is.

| Parameter | Meaning |
|---|---|
| `swing` | every other step starts this many steps late (0 to 0.9): 0.33 is a triplet feel, 0.5 a hard shuffle. The gate is held as long from there |
| `humanize` | every step is early or late by up to this many steps (0 to 0.5), by chance. It is the same for the same step and seed, so a patch always plays the same |
| `accent` | a list of levels, one for each step, that goes round: the note on a step is that loud (the output of the instrument is multiplied by it). `accents([3 3 2])` makes one from the groups of a meter: the first step of the bar strongest, the first of each other group medium, the rest weak |

```harp
fn tick(gate) { noise() * env(gate, [0 1 0], [0.001 0.05], -4) * 0.3 }
out = play(tick, rhythm: every(1), legato: 0.2, swing: 0.33, accent: accents([4 4 4 4], other: 0.3)).dup()
```

`groove:` is the feel of a bar by name, from psalm: `mpc 54`, `mpc 58`, `mpc 62` (every other step late, a little softer), `shuffle`, `laid back`, `push`, `dilla`, `backbeat` (loudness per step). `lanes:` gives each step a value of its own, a lane to a line (apart with `;`), each lane going round on its own length, so a lane of 3 under a pattern of 4 drifts (polymeter): `lanes: "prob: 100 50 . 80; vel: 100 60 80; shift: . 10; len: 50"`. `prob` is the chance the step plays (percent), `vel` its loudness (percent), `shift` how late (percent of a step, negative is early) and `len` how long the gate is held (percent). A `.` is the default there.

With a [pattern](patterns.md) as the `rhythm` hits can fall anywhere in a step (rolls, triplets), and a hit may last as long as it says (`x*2`).

Edit them while the patch plays: the notes that were made already get the new feel, from about two steps on. Swing, humanize and accent are for `play`; the synths (`subtractive`, `modal`, ...) play their notes on the grid.

## The numbers of a note: fields

`fields:` is a list with one item for each number the instrument takes after the gate:

- a **list or generator** (`[1800 2800]`, `degrees(...)`, `mutate(...)`) moves on **once for every note**;
- a **signal** (`seq(phrase)`, `lfo(0.1)`) has its value read at the start of every note. This is how one stream can drive the notes and an effect at the same step (see [Generators](generators.md)). The value arrives up to 63 samples after the gate.

## Chords

A field can be a chord: a generator of chords (`triads(...)`, `chords`-style lists, `voicelead(...)`) or a list of chords. Every note of the rhythm then plays **all the notes of the chord**, on the same step, each as a run of the instrument with that note in the field. The other fields are the same for all of them.

```harp
fn pad(gate, note) {
  saw([hz(note) * 0.997, hz(note) * 1.003]).lowpass(cutoff: 1200, res: 0.2) * env(gate, [0 1 0], [0.05 1.5], -2) * 0.08
}
changes = voicelead(triads(a2, minor, progression("classical")), 48, 72)
out = play(pad, rhythm: every(16), legato: 0.9, fields: [changes], voices: 4)
```

A note of a chord is played by a voice of its own, so `voices` has to be at least the notes of the biggest chord (a list of chords is checked when the patch is built). Only one field can be a chord. A chord of no notes is a rest. If a live edit changes the chords, the few chords that were made already keep their notes.

## Voices

Notes go to the voices in turn. Each voice is a run of the instrument built once, so the end of a note (its release) goes on while the next starts. `voices: 3` for overlapping plucks; `1` for a monophonic line. Random choices in the instrument (`rand`, `choose`) are made once for each voice.

## Every note is a new synth

The nodes of the instrument that carry state and are made as harp's custom nodes start again from their start on the rise of the gate, as a new synth does in SuperCollider: the filters (`lowpass`, `highpass`, `bandpass`, `bpf`), `echo`, `comb`, `reverb`, `freeverb2`, `env`, and the `sc*` functions. Plain oscillators and noise (`sine`, `saw`, `noise`, `lfo`, ...) keep running, so a note starts at whatever phase they are at; use `scsawsync` or `scsin` where the start must be exact.

Build what should *carry over* (a reverb tail, say) outside the instrument and put it on the output of `play`.

Notes start in the 64-sample block their time falls in, as SuperCollider's messages do.

## Limits

- Notes are only as exact as their nodes: for a restart that is the same every time, use `scsawsync`, `env` and the `sc*` functions ([SuperCollider](supercollider.md)).
- A signal field must have one channel.
