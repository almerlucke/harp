# Synths

A synth plays **notes**, a generator made by `melody(pitches, rhythm)` or `chords(chords, rhythm)` (see [Generators](generators.md)), with a **preset** and a number of **voices**:

```harp
bpm = 100
notes = melody(degrees(a3, minor, mutate([0 2 4 7], swap: 0.3)), euclid(16, 7), velocity: 0.9, length: 1.5)
out = modal(notes, preset: "marimba", voices: 6).reverb(mix: 0.2)
```

The sound is stereo. A note that comes while all the voices are busy takes over the oldest voice.

| Synth | Makes |
|---|---|
| `subtractive` | oscillators, a filter with its own envelope, lfos and a modulation matrix: basses, leads, pads |
| `supersaw` | the Roland JP-8000 supersaw: seven detuned saws, an ensemble |
| `modal` | struck and bowed bars, bells, bottles, cymbals, gongs, as sets of resonances |
| `piano` | hammers, strings and a soundboard |
| `waveguide` | physical models: plucked and bowed strings, flutes, drums, pipes |
| `vocal` | a singing voice: solo, choir, humming, chants |
| `pulsar` | trains of short pulses with formants: vowel leads, buzz bass, insects |
| `shaper` | waveshaping and phase distortion: cz-like leads, fm basses, pads |

`preset:` is the name of a preset in snake case: the function of psalm without `Preset` (`GrandPreset` is `"grand"`, `ChurchBell` is `"church_bell"`). The names are in [Presets](presets.md); `harp presets modal` lists one synth's.

A wrong name says what is nearby: `there is no modal preset "marimbaa" (did you mean marimba?)`.

## Notes

`melody` and `chords` take `velocity` (0 to 1) and `length` (in steps) of the notes. The pitches are MIDI notes (`a3`, `60`, `degrees(...)`). The same synth can play a rhythm of chords or a line.

## Your own instrument

For a sound of your own on every note (an `fn` with oscillators, filters and envelopes), see [play](play.md).

## Not yet

Recordings are in [Recordings](recordings.md) (`sampler`, `streamer`). Not wrapped yet: `organ`, `dx` and `drumsynth`.

## Made from a sound

Two synths take a sound and make the instrument of it.

**`grains(notes, file, root: 60, preset: "choir_pad")`** cuts a recording into short grains; a note is a stream of them at its pitch. `root` is the note at which the recording is at its own pitch. A preset sets the character (`frozen_drone` holds one moment, `shimmer`, `glass_keys`, `pluck`, `stutter`, `texture`); `position` (0 to 1, where in the recording), `scan` (how fast that moves: 0 holds, 1 is the original speed), `spray`, `density`, `size` (a grain, in ms) and `pitchspread` change what the preset set.

**`wavetable(notes, table: "basic", preset: "pad")`** morphs through the frames of a table, with unison, a sub oscillator and a filter. The table is a sound file of single cycles (`size:` samples a cycle, 2048 is usual) or one that psalm makes: `sync`, `fm`, `fold`, `pwm`, `vowels`, `drawbars`, `filter` and more (`frames:` of them, `amount:` for the one number of the table: the sync ratio, the fm index, the pulse width; the list is in [the reference](reference.md)). `position` is where in the table.

**Moving wavetables.** `position`, `cutoff` and `resonance` can be a signal instead of a number: it moves the table, the filter and its resonance while the notes sound, a new value every 1.5 ms. `warp` (bends the cycle; `warpmode: "sync"` and others say how), `detune`, `spread` (the width of the unison) and `sub` are signals from 0 to 1 that set how much of the preset's own setting there is (up to four of these can be signals). Any signal does: a slow `lfo(0.05).range(0, 1)`, an `env`, or a generator through `seq`: `seq(remap(chaos("lorenz"), -20, 20, 0.1, 0.9))` has the table follow the Lorenz attractor.

```harp
notes = melody(degrees(a3, minor, [0 2 4 7]), every(8), length: 7)
out = wavetable(notes, table: "vowels", preset: "evolving_pad",
  position: lfo(0.05).range(0.05, 0.95), cutoff: lfo(0.03).range(900, 5000), spread: lfo(0.02).range(0.3, 1))
```

```
held = chords([[a2 e3 a3 c4] [f2 c3 f3 a3]], every(32), length: 28)
pad = grains(held, "../../../psalm/assets/sounds/chants/Johannes.Hymnus.wav", preset: "frozen_drone", root: 57)
bass = wavetable(melody([a1 f1], every(32), length: 24), table: "sync", preset: "reese", position: 0.4)
out = (pad.gain(0.5) + bass.gain(0.6)).reverb(mix: 0.3, decay: 4)
```

Both are in `examples/grains_wavetable.harp`. The presets are in [Presets](presets.md).
