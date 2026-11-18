# Effects of psalm

psalm has a library of effects; harp hosts them as functions. They are made from what each effect says of itself (its settings, their ranges and defaults), so a new effect in psalm is a few lines here.

| Function | Is |
|---|---|
| `chorus` `flanger` `phaser` `tremolo` | modulation: copies that move, a sweeping delay or filter, a level or place that moves |
| `rotary` | a rotary speaker, slow or fast |
| `widener` `ringMod` | stereo width; ring modulation or frequency shift |
| `crusher` `saturate` `tape` | bit crushing; clipping, folding, tube; tape delay with wow and flutter |
| `pingPong` `shimmer` `freeverb` | a delay that bounces; a reverb that goes up an octave; the Freeverb |
| `granulator` | the sound cut in grains, scattered, shifted, frozen |
| `compressor` `gate` `transient` `eq` | dynamics and tone |
| `vocoder` | the spectrum of one sound on another |

Each function's settings are in the [reference](reference.md#effects-of-psalm), with range, unit and default.

```harp
out = saw([110.0, 110.7])
  .lowPass(cutoff: 900)
  .chorus(rate: 0.5, depth: 4, mix: 0.4)
  .tape(time: 320, feedback: 0.4, mix: 0.3)
  .gain(0.2)
```

## Settings

A setting is a named argument, and it can be:

- a **number**, set when the patch is built (`mix: 0.4`);
- a **name**, for a choice (`shape: "triangle"`, `mode: "pan"`);
- `true` or `false`, for a switch (`fast: true`);
- a **signal**, which moves the setting while it plays: `rate: lfo(0.1).range(0.2, 4)`. A signal can move a knob (a number setting), not a choice or a switch. It is read once in every call of at most 64 samples, and psalm glides to each new value (30–50 ms), so it does not click.

Settings you leave out keep psalm's defaults. A setting that is out of range is clamped by the effect.

## In and out

- The sound comes **out in stereo**. A mono signal goes to both sides; a stereo one is used as it is. A signal with more channels has to be mixed down or `spread` first.
- A few effects work on blocks of 64 samples (`shimmer`, `granulator`, `freeverb`, `vocoder`): their sound is **1.5 ms late**. The reference says which.
- The effects start from silence with their settings already in place.

## Sidechain and vocoder

`compressor` and `gate` listen to their input, or to another signal if you give `sidechain:`:

```harp
bass = saw(55).lowPass(cutoff: 400).gain(0.3)
kick = sine(60).gain(0.5) * pulses(every(4), length: 0.2)

out = bass.compressor(threshold: -30, ratio: 12, attack: 2, release: 120, sidechain: kick) + kick.dup()
```

The vocoder takes two signals: `vocoder(carrier, modulator)`.

## In a live edit

An effect is kept (its tail rings on) when its settings do not change; a changed number setting makes it a new effect with a new tail. To change a setting without a restart, make it a signal: `mix: seq([0.2 0.5])` or `lfo(...)`. See [Live editing](live-editing.md).
