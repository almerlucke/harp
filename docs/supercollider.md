# Comparing with SuperCollider

harp can render a patch and the SuperCollider code it translates, and say how alike they are. It was used to check harp against SuperCollider 3.14.1, and to find what is different.

## `harp compare`

```
harp compare patch.harp [-sc code|file] [-seconds 10] [-rate 44100] [-stats] [-seeds N] [-ref file.wav] [-save folder] [-sclang folder]
```

The SuperCollider code is taken from the top of the patch: a comment line that starts with `// SuperCollider`, then the code, indented three spaces, as a function:

```
// SuperCollider: a sine with vibrato, the same on both channels
//
//   { SinOsc.ar(SinOsc.kr(5, 0, 10, 440), 0, 0.2) ! 2 }.play

out = sine(lfo(5) * 10 + 440).gain(0.2).dup()
```

| Option | Does |
|---|---|
| `-sc code-or-file` | use other SuperCollider code |
| `-stats` | compare only the level and spectrum, not the waves (for noise and dust) |
| `-seeds N` | for a patch made by chance when it is built: render seeds 1..N in both and compare the *distributions* (use 50 or more) |
| `-ref file.wav` | compare with a WAV that SuperCollider made (a `Pbind` cannot run as code; see `lang/examples/supercollider/gabber_compare.scd`) |
| `-save folder` | write both renderings as WAV files, to listen to |
| `-sclang folder` | where `sclang` and `scsynth` are (default: `$HARP_SC`, the `PATH`, `~/.local/supercollider/bin`) |

The report has, per channel, the level, the best alignment and correlation, the level over time, the energy in octave bands, and for stereo the position over time; then a verdict in words, with its limits (level within 0.5 dB is the same; spectrum shape within 1 dB). SuperCollider is run offline (`Score.recordNRT`), so nothing needs a sound card.

## SuperCollider's own algorithms

Some SuperCollider sounds depend on how its units work, not only on what they are. These functions are ports of SuperCollider's code, for translations that must match:

`scSaw` `scSawSync` `scSin` `scLpf` `scHpf` `scRlpf` `scPeq` `scDelayN` `scDelayC` `freeverb2` and `env` (SuperCollider's `EnvGen`). See the [reference](reference.md#supercolliders-own-algorithms).

What was found:

- SuperCollider reads most parameters once per **block of 64 samples**: control signals (`.kr`), filter frequencies, the frequency of `Saw`, gates, and the time messages apply. `lfNoise(..., control: true)` and `ramp(..., control: true)` do the same.
- `Saw` is a leaky integrated pulse train: half the size, falling, and it **rings** (peaks many times its size) when its frequency jumps. harp's `saw` does not. A patch that sweeps a saw fast sounds different with each.
- `Pulse` is half the size, has no DC and is turned over; harp's `pulse` is the conventional one (`rectangle` has no DC).
- A new synth starts with its filters holding the first input sample, `DelayC` has a minimum delay of 1 sample, `SinOsc` has a whole-number phase (8 Hz is 9 ppm off).
- With these ported, the kick of "60Hz Gabber Rave 1995" matches SuperCollider's wave to -55 dB over 20 s.

A patch that uses `rand` or `choose` cannot be the same wave as SuperCollider's (the random numbers differ); compare it with `-seeds`.
