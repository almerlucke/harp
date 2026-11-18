# Synths

A synth plays **notes**, a generator made by `melody(pitches, rhythm)` or `chords(chords, rhythm)` (see [Generators](generators.md)), with a **preset** and a number of **voices**:

```harp
bpm = 100
notes = melody(degrees(a3, minor, mutate([0 2 4 7], swap: 0.3)), euclid(16, 7), velocity: 0.9, length: 1.5)
out = modal(notes, preset: "marimba", voices: 6).reverb(mix: 0.2)
```

The sound is stereo. A note that comes while all the voices are busy takes over the oldest voice (the organ is the exception: it is one instrument and has no voices).

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
| `organ` | a tonewheel organ: drawbars, percussion, vibrato, a Leslie speaker |
| `dx` | FM in the way of the Yamaha DX7 and TX81Z: electric pianos, bells, brass, and the patches of a DX7 SysEx file |
| `drumSynth` | drums that are made, not played from recordings: kits of 808, 909, electro, trap, industrial |
| `grains`, `wavetable` | made from a sound or a table, below |

`preset:` is the name of a preset in snake case: the function of psalm without `Preset` (`GrandPreset` is `"grand"`, `ChurchBell` is `"church_bell"`). The names are in [Presets](presets.md); `harp presets modal` lists one synth's.

A wrong name says what is nearby: `there is no modal preset "marimbaa" (did you mean marimba?)`.

## Notes

`melody` and `chords` take `velocity` (0 to 1) and `length` (in steps) of the notes. The pitches are MIDI notes (`a3`, `60`, `degrees(...)`). The same synth can play a rhythm of chords or a line.

### Velocity and length sequences

`velocity` and `length` can each be a number (the same for every note), or a **list or a generator of numbers**: one value for each note, in turn, going round. They are taken with the pitch when the rhythm plays, so a rest takes none (for `chords`, one value for each chord, the same for all its notes).

```harp
bpm = 100
steps = 4
notes = melody([a3 c4 e4 a4], euclid(16, 9),
  velocity: [1 0.4 0.7 0.4 0.9],   // a pattern of accents, five long, against the four pitches
  length: [1 1 3])                // short, short, long
out = modal(notes, preset: "marimba", voices: 6)
```

A generator moves them by itself: `velocity: wander(0.4, 1)` is a player that is a little different each time, `length: [1 1 1 4]` a phrase that holds every fourth note. **A velocity of 0 is a rest**: the note is not played, and the pitch and the length move on (`velocity: [1 1 0 1]` leaves out every third note). Lists of different lengths go round against each other, so the pattern comes back together only after their lowest common multiple.

In a [score](scores.md), `phrase` takes the same: `velocity: [1 0.5 0.8]` and `legato: [1 0.5]` are a value for each pitch (a rest has one too).

## Every setting

A preset is a starting point. Every setting of a synth is a parameter of its function, with the name psalm gives it, and what is not given stays as the preset has it. `harp settings subtractive` lists them with their ranges (all synths: [settings.md](settings.md)).

```harp
n = melody([a2 a2 c3 a2], every(2), length: 3)
out = subtractive(n, preset: "bass",
    filterMode: "band pass", cutoff: lfo(0.2).range(200, 2500), resonance: 0.6,
    ampAttack: 40, ampRelease: 600)
```

- a number sets it in every voice: `cutoff: 900`;
- a name chooses (any writing: `"band pass"`, `"bandPass"`): `filterMode: "high pass"`;
- `true` or `false` is a switch;
- a signal, for a knob, moves it while the notes sound: `cutoff: lfo(0.1).range(300, 3000)`, `env(...)`, `seq(...)` (read once in every 64 samples, and the voices glide to it);
- the envelopes have a parameter for each stage, in milliseconds (the sustain is a level from 0 to 1): `ampAttack`, `ampDecay`, `ampSustain`, `ampRelease`, and `filterAttack...`, `positionAttack...` for the synths that have those envelopes.

A wrong value says what is allowed (`cutoff is from 20 to 20000 Hz: 99999 is not`), and a name that is nearly right is suggested. In a live edit a changed setting is changed in the notes that sound, and one that is taken out goes back to what the preset has.

The modulation matrix, and the sets that are too big for parameters (the operators of `dx`, the drums of `drumSynth`), are not in the parameters yet.

## Your own instrument

For a sound of your own on every note (an `fn` with oscillators, filters and envelopes), see [play](play.md).

## Recordings

Synths that play recordings (`sampler`, `streamer`) are in [Recordings](recordings.md). With these, every synth of psalm is in harp.

## Organ, dx and drumSynth

**`organ(notes, preset: "jazz")`** is one instrument, not a set of voices: every note is a key of it (a chord is several keys, a note is held for its length), and the keys share its wheels, as on the real one. The preset is a registration; `drawbars: "888000000"` is one written as organists do, a digit from 0 to 8 for each of the nine drawbars; `leslie` (0 to 1) is how much rotary speaker there is and `speed` how fast it turns. There is no `voices`.

**`dx(notes, preset: "e_piano", voices: 8)`** is FM: operators that modulate each other. Besides its own patches, it reads the patches of a DX7: `dx(notes, file: "rom1a.syx", patch: 10)` plays patch 10 (from 0) of a SysEx bank, and `harp check` says the name of it.

**`drumSynth(notes, kit: "808")`** plays drums by their General MIDI key: a note is a drum. `drum("kick")` gives the key by name, so a pattern is written with names:

```harp
bpm = 100
steps = 4
kick = drumSynth(melody([drum("kick")], pattern("x..x..x. x...x..."), velocity: 1, length: 1), kit: "trap")
hats = drumSynth(melody([drum("hat")], every(2), velocity: 0.5, length: 1), kit: "trap")
out = (kick + hats).gain(0.5)
```

The voices of one `drumSynth` are made together, so that a closed hat stops an open one. A kit has all the sounds; the names for `drum(...)`: `kick` (35), `rim`, `snare`, `clap`, `tomLow`, `tomMid`, `tomHigh`, `hat` (42), `open` (46), `crash`, `ride`, `rideBell`, `tambourine`, `cowbell`, `shaker`, `bongoHigh`, `bongoLow`, `congaMute`, `congaHigh`, `congaLow`, `timbaleHigh`, `timbaleLow`, `agogoHigh`, `agogoLow`, `clave`, `woodblockHigh`, `woodblockLow`, `triangleMute`, `triangle`. For several drums at once, use `chords` (a kick and a hat on the same step) or one `drumSynth` for each.

## Made from a sound

Two synths take a sound and make the instrument of it.

**`grains(notes, file, root: 60, preset: "choir_pad")`** cuts a recording into short grains; a note is a stream of them at its pitch. `root` is the note at which the recording is at its own pitch. A preset sets the character (`frozen_drone` holds one moment, `shimmer`, `glass_keys`, `pluck`, `stutter`, `texture`); `position` (0 to 1, where in the recording), `scan` (how fast that moves: 0 holds, 1 is the original speed), `spray`, `density`, `size` (a grain, in ms) and `pitchSpread` change what the preset set.

**`wavetable(notes, table: "basic", preset: "pad")`** morphs through the frames of a table, with unison, a sub oscillator and a filter. The table is a sound file of single cycles (`size:` samples a cycle, 2048 is usual) or one that psalm makes: `sync`, `fm`, `fold`, `pwm`, `vowels`, `drawbars`, `filter` and more (`frames:` of them, `amount:` for the one number of the table: the sync ratio, the fm index, the pulse width; the list is in [the reference](reference.md)). `position` is where in the table.

**Moving wavetables.** `position`, `cutoff` and `resonance` can be a signal instead of a number: it moves the table, the filter and its resonance while the notes sound, a new value every 1.5 ms. `warp` (bends the cycle; `warpMode: "sync"` and others say how), `detune`, `spread` (the width of the unison) and `sub` are signals from 0 to 1 that set how much of the preset's own setting there is (up to four of these can be signals). Any signal does: a slow `lfo(0.05).range(0, 1)`, an `env`, or a generator through `seq`: `seq(remap(chaos("lorenz"), -20, 20, 0.1, 0.9))` has the table follow the Lorenz attractor.

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
