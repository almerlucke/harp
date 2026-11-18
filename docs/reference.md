# Reference: the functions

<!-- made by `harp functions -md`: do not edit by hand (a test says when it is out of date) -->

Every function can be called as `f(x, ...)` or as `x.f(...)`: the thing before the dot is the first argument. A parameter with `= value` can be left out. Arguments can be named: `lowpass(x, cutoff: 800)`. The kinds (signal, generator, ...) are explained in [types](types.md).

## Oscillators and noise

### `sine`

`sine(freq: signal) -> signal`

A sine: freq in Hz.

### `saw`

`saw(freq: signal) -> signal`

A rising sawtooth between -1 and 1, band limited (SuperCollider's Saw is half the size and falls): freq in Hz (a list of frequencies is a signal of several channels, one oscillator each).

### `square`

`square(freq: signal) -> signal`

A square wave from -1 to 1: freq in Hz (psalm's band limited oscillator).

### `triangle`

`triangle(freq: signal) -> signal`

A triangle wave from -1 to 1: freq in Hz (psalm's band limited oscillator).

### `pulse`

`pulse(freq: signal, width: signal = 0.5) -> signal`

A band limited pulse wave between -1 and 1: +1 for the first width of the period and -1 for the rest, freq in Hz, width from 0 to 1 (0.5 is a square). With a width other than 0.5 it has a mean of 2*width-1 (a DC offset; rectangle has none). SuperCollider's Pulse is not this: it is half the size, has a mean of 0 and is turned over (see examples/supercollider/6_pan_pulse.harp).

### `rectangle`

`rectangle(freq: signal, width: signal = 0.5) -> signal`

Psalm's band limited pulse wave, with no DC offset: its mean is 0 for every width (so it swings between about +1.6 and -0.8 at a width of 0.3), freq in Hz, width from 0 to 1.

### `ramp`

`ramp(freq: signal, control: boolean = false) -> signal`

A rising ramp from -1 to 1, freq times a second, that starts at 0 and jumps back to -1 (LFSaw). Not band limited, for modulation. With control: true as SuperCollider's .kr: one value for every block of 64 samples.

### `lfo`

`lfo(rate: signal) -> signal`

A sine for modulation, from -1 to 1: rate in Hz. Use range to map it.

### `noise`

`noise() -> signal`

White noise from -1 to 1 (WhiteNoise); every noise is a noise of its own.

### `lfnoise`

`lfnoise(freq: signal, interp: boolean = true, control: boolean = false) -> signal`

A new random value from -1 to 1, freq times a second: in straight lines between them (LFNoise1), or in steps with interp: false (LFNoise0). With control: true as SuperCollider's .kr: one value for every block of 64 samples, so that it moves in steps of 1.5 ms.

### `dust`

`dust(density: signal) -> signal`

Random impulses of a value from 0 to 1, density of them a second on average.

### `impulse`

`impulse(freq: signal) -> signal`

A 1 at the start and then freq times a second, and 0 between.

## Envelopes and ramps

### `adsr`

`adsr(attack: number, decay: number, sustain: number, release: number) -> envelope`

An envelope: attack, decay and release in seconds, sustain a level from 0 to 1. Give it a gate with on.

### `on`

`on(env: envelope, gate: signal) -> signal`

Runs an envelope on a gate: a rising edge starts it, a falling edge lets it go.

### `env`

`env(gate: signal, levels: anything, times: anything, curve: anything = 0, release: number = -1, loop: number = -1, control: boolean = false, shape: string = curve) -> signal`

An envelope as SuperCollider's EnvGen: it starts when the gate rises, goes through levels in times (seconds, one fewer than the levels), each segment bent by its curve (0 is straight, negative is fast first, a list for a curve for each segment). shape: how a segment goes (curve uses the curve number; exp is XLine). With release: n it waits after segment n for the gate to fall. control: true is .kr (a value for every 64 samples).

### `decay`

`decay(x: signal, time: signal) -> signal`

Makes impulses an exponential decay that falls 60 dB in time seconds.

### `decay2`

`decay2(x: signal, attack: signal, decay: signal) -> signal`

Makes impulses a swell that rises in about attack seconds and falls 60 dB in decay seconds (Decay2).

### `line`

`line(start: number, end: number, seconds: number) -> signal`

Goes from start to end in seconds, in a straight line, and stays at end.

### `xline`

`xline(start: number, end: number, seconds: number) -> signal`

Goes from start to end in seconds by the same ratio every second, and stays at end. Neither can be 0.

## Filters

### `lowpass`

`lowpass(x: signal, cutoff: signal, res: signal = 0.5) -> signal`

Psalm's state variable lowpass: cutoff in Hz (a signal moves it), res from 0 to 1. A filter for every channel.

### `highpass`

`highpass(x: signal, cutoff: signal, res: signal = 0.5) -> signal`

Psalm's state variable highpass: cutoff in Hz, res from 0 to 1.

### `bandpass`

`bandpass(x: signal, cutoff: signal, res: signal = 0.5) -> signal`

Psalm's state variable bandpass: cutoff in Hz, res from 0 to 1 (near 1 it is narrow). Its gain at the cutoff is the Q, which is higher than a unity bandpass (Resonz).

### `bpf`

`bpf(x: signal, freq: signal, rq: number = 1) -> signal`

A bandpass that lets the cutoff through at its own level, as SuperCollider's BPF: freq in Hz, rq is the bandwidth as a part of freq (0.2 is a Q of 5).

## Delays and reverb

### `echo`

`echo(x: signal, max: number, time: signal, feedback: signal = 0.4, mix: signal = 0.3) -> signal`

A delay with feedback: max is the longest time in seconds, time can be a signal (a list gives every channel its own), mix is how much of the echoes is heard. A time that moves bends the pitch of what is in the line (the echoes squeak): keep it fixed, or move it slowly.

### `comb`

`comb(x: signal, delay: number, decay: number) -> signal`

A comb filter with a feedback delay of delay seconds that rings for decay seconds (60 dB), as CombN: only the delayed sound is heard.

### `reverb`

`reverb(x: signal, mix: number = 0.25, decay: number = 2.5) -> signal`

Psalm's feedback delay network reverb: mix from 0 (dry) to 1, decay is the reverb time in seconds. Stereo out.

## Effects of psalm

### `chorus`

`chorus(x: signal, rate: anything = 0.6, depth: anything = 3, delay: anything = 15, spread: anything = 1, feedback: anything = 0, tone: anything = 9000, mix: anything = 0.4, voices: anything = 3) -> signal`

A chorus: copies of the sound whose delay moves, so that it thickens and widens. The sound comes out in stereo (a mono signal goes to both sides). Settings: rate (0.05 to 10 Hz, 0.6); depth (0 to 10 ms, 3); delay (1 to 35 ms, 15); spread (0 to 1, 1); feedback (0 to 0.9, 0); tone (500 to 20000 Hz, 9000); mix (0 to 1, 0.4); voices (1 to 4, 3). A setting can be a number or a signal that moves it.

### `flanger`

`flanger(x: signal, rate: anything = 0.2, beats: anything = 0, manual: anything = 0.5, depth: anything = 4, feedback: anything = 0.5, stereo: anything = 0.5, mix: anything = 0.5, shape: anything = 1) -> signal`

A flanger: a short delay that sweeps, with feedback. The sound comes out in stereo (a mono signal goes to both sides). Settings: rate (0.02 to 10 Hz, 0.2); beats (0 to 32, 0); manual (0.1 to 10 ms, 0.5); depth (0 to 20 ms, 4); feedback (-0.95 to 0.95, 0.5); stereo (0 to 1, 0.5); mix (0 to 1, 0.5); shape (sine, triangle; triangle). A setting can be a number or a signal that moves it.

### `phaser`

`phaser(x: signal, rate: anything = 0.4, sync: anything = 0, min: anything = 300, max: anything = 2500, feedback: anything = 0.4, stereo: anything = 0.25, mix: anything = 0.5, stages: anything = 4) -> signal`

A phaser: allpass stages that sweep, with feedback. The sound comes out in stereo (a mono signal goes to both sides). Settings: rate (0.01 to 10 Hz, 0.4); sync (0 to 32 beats, 0); min (20 to 10000 Hz, 300); max (100 to 20000 Hz, 2500); feedback (-0.95 to 0.95, 0.4); stereo (0 to 1, 0.25); mix (0 to 1, 0.5); stages (1 to 12, 4). A setting can be a number or a signal that moves it.

### `tremolo`

`tremolo(x: signal, mode: anything = 0, shape: anything = 0, rate: anything = 0.05, sync: anything = 0.5, depth: anything = 0.5, stereo: anything = 0) -> signal`

A tremolo (the level moves) or an auto pan (the place moves). The sound comes out in stereo (a mono signal goes to both sides). Settings: mode (tremolo, pan; tremolo); shape (sine, triangle, square; sine); rate (0.05 to 20 Hz, 0.05); sync (0 to 4 beats, 0.5); depth (0 to 1, 0.5); stereo (0 to 1, 0). A setting can be a number or a signal that moves it.

### `rotary`

`rotary(x: signal, fast: anything = 0, drive: anything = 3, horn: anything = 0.5, spread: anything = 1, mix: anything = 1, crossover: anything = 800) -> signal`

A rotary speaker (Leslie): a horn and a drum that turn, slow or fast. The sound comes out in stereo (a mono signal goes to both sides). Settings: fast (true or false, false); drive (0 to 24 dB, 3); horn (0.01 to 1, 0.5); spread (0.01 to 1, 1); mix (0 to 1, 1); crossover (150 to 4000 Hz, 800). A setting can be a number or a signal that moves it.

### `widener`

`widener(x: signal, width: anything = 1.4, monoBelow: anything = 120) -> signal`

A stereo widener, that keeps the lows in the middle. The sound comes out in stereo (a mono signal goes to both sides). Settings: width (0 to 3, 1.4); monoBelow (20 to 1000 Hz, 120). A setting can be a number or a signal that moves it.

### `ringmod`

`ringmod(x: signal, mode: anything = 0, freq: anything = 300, stereo: anything = 0, mix: anything = 0.5) -> signal`

A ring modulator, or a frequency shifter (mode: shift). The sound comes out in stereo (a mono signal goes to both sides). Settings: mode (ring, shift; ring); freq (-4000 to 4000 Hz, 300); stereo (-50 to 50 Hz, 0); mix (0 to 1, 0.5). A setting can be a number or a signal that moves it.

### `crusher`

`crusher(x: signal, bits: anything = 8, rate: anything = 11025, output: anything = 0, mix: anything = 1) -> signal`

A bit crusher and sample rate reducer. The sound comes out in stereo (a mono signal goes to both sides). Settings: bits (1 to 24, 8); rate (100 to 44100 Hz, 11025); output (-24 to 24 dB, 0); mix (0 to 1, 1). A setting can be a number or a signal that moves it.

### `saturate`

`saturate(x: signal, shape: anything = 0, drive: anything = 9, bias: anything = 0, tone: anything = 9000, output: anything = -4, mix: anything = 1) -> signal`

Saturation: soft, tube, hard clipping or folding, with drive and tone. The sound comes out in stereo (a mono signal goes to both sides). Settings: shape (soft, tube, hard, fold; soft); drive (0 to 36 dB, 9); bias (-1 to 1, 0); tone (500 to 20000 Hz, 9000); output (-24 to 12 dB, -4); mix (0 to 1, 1). A setting can be a number or a signal that moves it.

### `tape`

`tape(x: signal, time: anything = 0.75, feedback: anything = 0.45, wow: anything = 0.3, flutter: anything = 0.2, drive: anything = 3, highCut: anything = 4500, lowCut: anything = 150, mix: anything = 0.25) -> signal`

A tape delay with wow, flutter and saturation. The sound comes out in stereo (a mono signal goes to both sides). Settings: time (0.0625 to 4 beats, 0.75); feedback (0 to 1.2, 0.45); wow (0 to 1, 0.3); flutter (0 to 1, 0.2); drive (1 to 24 dB, 3); highCut (500 to 20000 Hz, 4500); lowCut (20 to 2000 Hz, 150); mix (0 to 1, 0.25). A setting can be a number or a signal that moves it.

### `pingpong`

`pingpong(x: signal, time: anything = 0.75, feedback: anything = 0.35, stereo: anything = 0, highCut: anything = 6000, lowCut: anything = 120, mix: anything = 0.2) -> signal`

A delay that bounces from one side to the other. The sound comes out in stereo (a mono signal goes to both sides). Settings: time (0.0625 to 4 beats, 0.75); feedback (0 to 0.98, 0.35); stereo (0 to 1, 0); highCut (500 to 20000 Hz, 6000); lowCut (20 to 2000 Hz, 120); mix (0 to 1, 0.2). A setting can be a number or a signal that moves it.

### `shimmer`

`shimmer(x: signal, shimmer: anything = 0.5, semitones: anything = 12, lowCut: anything = 300, mix: anything = 0.37, size: anything = 0.85, decay: anything = 6, damping: anything = 0.4, preDelay: anything = 20, diffusion: anything = 0.8, modDepth: anything = 0.6, modRate: anything = 0.6, width: anything = 1) -> signal`

A reverb whose tail goes up an octave (or another interval) again and again. The sound comes out in stereo (a mono signal goes to both sides). Settings: shimmer (0 to 1, 0.5); semitones (-24 to 24 st, 12); lowCut (20 to 2000 Hz, 300); mix (0 to 1, 0.37); size (0 to 1, 0.85); decay (0.1 to 30 s, 6); damping (0 to 1, 0.4); preDelay (0 to 490 ms, 20); diffusion (0 to 1, 0.8); modDepth (0 to 4 ms, 0.6); modRate (0.05 to 5 Hz, 0.6); width (0 to 1, 1). A setting can be a number or a signal that moves it. It works in blocks of 64 samples, so the sound is 1.5 ms late.

### `freeverb`

`freeverb(x: signal, roomSize: anything = 0.5, damp: anything = 0.5, width: anything = 1, wet: anything = 0.33, dry: anything = 0.7, mode: anything = 0) -> signal`

The Freeverb reverb (comb and allpass filters), stereo. The sound comes out in stereo (a mono signal goes to both sides). Settings: roomSize (0 to 1, 0.5); damp (0 to 1, 0.5); width (0 to 1, 1); wet (0 to 1, 0.33); dry (0 to 1, 0.7); mode (true or false, false). A setting can be a number or a signal that moves it. It works in blocks of 64 samples, so the sound is 1.5 ms late.

### `granulator`

`granulator(x: signal, density: anything = 25, grainSize: anything = 120, shape: anything = 0, jitter: anything = 0.4, delay: anything = 150, spray: anything = 80, feedback: anything = 0, freeze: anything = 0, pitch: anything = 0, pitchSpread: anything = 0, octave: anything = 0, reverse: anything = 0, stereoSpread: anything = 0.7, mix: anything = 0.5) -> signal`

A granular effect: the sound cut into small grains that are played again, scattered, shifted or frozen. The sound comes out in stereo (a mono signal goes to both sides). Settings: density (0.1 to 200 /s, 25); grainSize (5 to 1000 ms, 120); shape (0 to 1, 0); jitter (0 to 1, 0.4); delay (0 to 4000 ms, 150); spray (0 to 1000 ms, 80); feedback (0 to 0.98, 0); freeze (true or false, false); pitch (-24 to 24 st, 0); pitchSpread (0 to 12 st, 0); octave (0 to 1, 0); reverse (0 to 1, 0); stereoSpread (0 to 1, 0.7); mix (0 to 1, 0.5). A setting can be a number or a signal that moves it. It works in blocks of 64 samples, so the sound is 1.5 ms late.

### `compressor`

`compressor(x: signal, sidechain: anything = 0, threshold: anything = -18, ratio: anything = 3, knee: anything = 6, attack: anything = 10, release: anything = 150, makeup: anything = 0, mix: anything = 1, autoMakeup: anything = 0, sideCut: anything = 20) -> signal`

A compressor with a soft knee and a sidechain: give a signal as sidechain to duck by it (a kick that makes the bass go down). The sound comes out in stereo (a mono signal goes to both sides). Settings: threshold (-60 to 0 dB, -18); ratio (1 to 20, 3); knee (0 to 24 dB, 6); attack (0.1 to 200 ms, 10); release (5 to 2000 ms, 150); makeup (-12 to 24 dB, 0); mix (0 to 1, 1); autoMakeup (true or false, false); sideCut (20 to 500 Hz, 20). A setting can be a number or a signal that moves it.

### `gate`

`gate(x: signal, sidechain: anything = 0, threshold: anything = -45, ratio: anything = 100, range: anything = 80, attack: anything = 1, hold: anything = 30, release: anything = 120, sideCut: anything = 20) -> signal`

A gate or expander with a sidechain: it turns the sound down when it is quiet. The sound comes out in stereo (a mono signal goes to both sides). Settings: threshold (-80 to 0 dB, -45); ratio (1 to 100, 100); range (1 to 100 dB, 80); attack (0.1 to 100 ms, 1); hold (0 to 500 ms, 30); release (5 to 2000 ms, 120); sideCut (20 to 500 Hz, 20). A setting can be a number or a signal that moves it.

### `transient`

`transient(x: signal, attack: anything = 6, sustain: anything = -3, output: anything = 0, mix: anything = 1) -> signal`

A transient shaper: more or less attack and sustain. The sound comes out in stereo (a mono signal goes to both sides). Settings: attack (-24 to 24 dB, 6); sustain (-24 to 24 dB, -3); output (-24 to 12 dB, 0); mix (0 to 1, 1). A setting can be a number or a signal that moves it.

### `eq`

`eq(x: signal, lowCut: anything = 20, lowFreq: anything = 100, lowGain: anything = 0, mid1Freq: anything = 500, mid1Gain: anything = 0, mid1Q: anything = 1, mid2Freq: anything = 2500, mid2Gain: anything = 0, mid2Q: anything = 1, highFreq: anything = 8000, highGain: anything = 0, highCut: anything = 20000, output: anything = 0) -> signal`

An equalizer: low and high cuts, a low and a high shelf, two bells. The sound comes out in stereo (a mono signal goes to both sides). Settings: lowCut (20 to 1000 Hz, 20); lowFreq (20 to 1000 Hz, 100); lowGain (-18 to 18 dB, 0); mid1Freq (40 to 18000 Hz, 500); mid1Gain (-18 to 18 dB, 0); mid1Q (0.1 to 10, 1); mid2Freq (40 to 18000 Hz, 2500); mid2Gain (-18 to 18 dB, 0); mid2Q (0.1 to 10, 1); highFreq (1000 to 18000 Hz, 8000); highGain (-18 to 18 dB, 0); highCut (1000 to 20000 Hz, 20000); output (-24 to 24 dB, 0). A setting can be a number or a signal that moves it.

### `vocoder`

`vocoder(carrier: signal, modulator: signal, attack: anything = 5, release: anything = 50, flatten: anything = 1, formantShift: anything = 1, sibilance: anything = 0.25, sibilanceFreq: anything = 5000, gain: anything = 1) -> signal`

A vocoder: the spectrum of the modulator (a voice) on the carrier (a synth). The sound comes out in stereo (a mono signal goes to both sides). Settings: attack (1 to 500 ms, 5); release (5 to 2000 ms, 50); flatten (0 to 1, 1); formantShift (0.5 to 2, 1); sibilance (0 to 1, 0.25); sibilanceFreq (1000 to 16000 Hz, 5000); gain (0 to 4, 1). A setting can be a number or a signal that moves it. It works in blocks of 64 samples, so the sound is 1.5 ms late.

## Shaping and mapping

### `gain`

`gain(x: signal, amount: signal) -> signal`

Scales a signal by a number or another signal (x * amount).

### `tanh`

`tanh(x: signal) -> signal`

The hyperbolic tangent: a soft clip between -1 and 1.

### `atan`

`atan(x: signal) -> signal`

The arc tangent: a soft clip between -pi/2 and pi/2.

### `sign`

`sign(x: signal) -> signal`

-1, 0 or 1: the sign.

### `clip2`

`clip2(x: signal, level: signal = 1) -> signal`

Limits a signal to -level..level (clip2).

### `midiratio`

`midiratio(x: signal) -> signal`

The ratio of frequencies that many semitones apart: 12 is 2.

### `hz`

`hz(note: signal) -> signal`

MIDI notes as frequencies in Hz (midicps): 69 is 440. A signal of notes, a sweep, is a signal of frequencies.

### `range`

`range(x: signal, lo: number, hi: number) -> signal`

Maps a signal from -1..1 (an lfo) to lo..hi.

### `exprange`

`exprange(x: signal, lo: number, hi: number) -> signal`

Maps a signal from -1..1 to lo..hi along an exponential curve, the way pitch is heard (exprange). Neither can be 0.

### `linexp`

`linexp(x: signal, inlo: number, inhi: number, lo: signal, hi: signal) -> signal`

Maps a signal from the range inlo..inhi to lo..hi along an exponential curve (LinExp); lo and hi can be signals; the input is not clipped.

## Channels

### `dup`

`dup(x: signal, n: number = 2) -> signal`

Gives a signal of one channel n channels, all the same (x ! n).

### `mix`

`mix(x: signal) -> signal`

Adds all the channels of a signal into one (Mix).

### `spread`

`spread(x: signal, width: number = 1) -> signal`

Puts the channels of a signal in the stereo field, evenly from left to right (width 0 to 1); one channel is in the middle.

### `pan`

`pan(x: signal, pos: signal = 0) -> signal`

Puts a signal in the stereo field: pos from -1 (left) to 1 (right), equal power as Pan2. A signal of several channels gives left and right for every channel.

### `product`

`product(x: signal) -> signal`

Multiplies all the channels of a signal into one.

### `reverse`

`reverse(x: signal) -> signal`

The channels of a signal in the other order.

## Generators (rhythm, notes, phrases)

### `euclid`

`euclid(steps: number, pulses: number, rotation: number = 0) -> generator of booleans`

A euclidean rhythm: pulses spread evenly over steps, turned by rotation.

### `every`

`every(n: number) -> generator of booleans`

Plays on the first of every n steps.

### `mutate`

`mutate(phrase: list of number, swap: number = 0.5) -> generator of whole numbers`

Plays a phrase of scale degrees, and swaps notes around now and then (swap is the chance, 0 to 1).

### `degrees`

`degrees(root: number, scale: scale, degrees: generator of whole numbers) -> generator of numbers`

Scale degrees (0 is the root, 7 an octave up in a scale of 7 notes) as pitches, in a scale on a root note.

### `melody`

`melody(pitches: generator of numbers, rhythm: anything, velocity: anything = 0.8, length: anything = 1) -> generator of notes`

A melody on a rhythm, as notes for a synth: the next pitch is taken when the rhythm plays. velocity (0 to 1) and length (in steps) can each be a number, or a list or a generator of numbers: a sequence, one value for each note, in turn, that goes round (velocity: [1 0.5 0.8 0.5], length: [1 1 3], velocity: wander(0.4, 1)); a velocity of 0 is a rest.

### `chords`

`chords(chords: generator of chords, rhythm: anything, velocity: anything = 0.7, length: anything = 1) -> generator of notes`

Chords on a rhythm, as notes for a synth: when the rhythm plays, all the notes of the next chord start, held for length steps. velocity (0 to 1) and length can each be a number, or a list or a generator of numbers: a sequence, one value for each chord, in turn (velocity: [1 0.5 0.8 0.5], length: [2 6 4]); a velocity of 0 is a rest.

### `notes`

`notes(melody: generator of numbers, rhythm: generator of booleans, length: number = 0.6) -> voice`

A melody on a rhythm: the next pitch is taken when the rhythm plays, together with the gate that starts an envelope. Gives a voice with the signals pitch (in Hz) and gate.

## Rhythms

### `pattern`

`pattern(steps: string, mutate: number = 0, seed: number = 0, freeze: number = 0, drift: number = 0, song: anything = 0) -> anything`

A rhythm written as text, as the clock of psalm reads it: x is a hit and . a rest, a step each ("x..x ..x. x...", spaces and | are for reading). x3 is a roll, three hits in a step; [x x x] a group that fills one step (a triplet; [x . x x] with rests); *2 after a step or a group: it lasts 2 steps; ?25 after a hit: 25% chance (? is 50%); %2/4 only the second time through in every four (%!2/4 the other three), %first only the first time, %fill only in the last bar of a section of song; ! after a hit: it stays when the pattern mutates. The functions E(5,16) (euclid), A(3+3+2) (additive), CA(30,16) (a cellular automaton) and M(x . x x;16;3) (a Markov chain on an example) give their steps, and +1 in them (E(5,16,+1)) goes on every pass: the rhythm turns round. mutate (0 to 1) changes the pattern from pass to pass (hits fall out, rests become hits: as many as a quarter of the steps at 1) with seed, freeze is the pass after which it changes no more, drift (0 to 0.3) moves the timing along a smooth curve. Plain steps and rests give a rhythm of booleans; with more it gives hits, that melody, chords and play take as their rhythm.

### `maybe`

`maybe(rhythm: generator of booleans, p: number) -> generator of booleans`

A rhythm with each hit kept with the chance p (0 to 1): maybe(euclid(16, 5), 0.8).

### `invert`

`invert(rhythm: generator of booleans) -> generator of booleans`

A rhythm turned inside out: a hit where there was a rest and a rest where there was a hit.

### `both`

`both(rhythms: list of generator of booleans) -> generator of booleans`

The hits that the rhythms of the list have together (and): both([euclid(16, 9), every(2)]).

### `either`

`either(rhythms: list of generator of booleans) -> generator of booleans`

The hits of all the rhythms of the list (or): either([euclid(16, 5), euclid(16, 3, 4)]).

### `meter`

`meter(divisions: list of number, level: generator of numbers) -> generator of booleans`

The hits of a meter, thinned by a level: the pulses of the divisions (meter([4, 4], level) is 16 pulses in 4 groups of 4) have an importance, the downbeat the most; the pulses less important than the level (a number, or a generator of numbers: a rising level thins the rhythm out) are rests. Level 0 is every pulse.

### `sieve`

`sieve(length: number, mods: list of number, starts: list of number) -> generator of booleans`

A rhythm from a sieve of Xenakis: a hit on every step that is a member of a residue class, the classes of mods and starts joined (sieve(16, [3, 5], [0, 2]): the steps 0 3 6 .. and 2 7 12 ..); length is how many steps go round.

## Combining generators

### `alternate`

`alternate(gens: list) -> anything`

Generators in turn: a value from the first, then one from the second, and so on, round again (a list of generators of one kind: alternate([melody(...), melody(...)])).

### `anyof`

`anyof(gens: list) -> anything`

Generators at random: every value is from one of the generators of the list, picked at random (a list of generators of one kind).

### `chance`

`chance(gens: list, weights: list of number) -> anything`

Generators at random with weights: every value is from one of the generators, picked with a chance in proportion to its weight (chance([a, b], [3, 1]): a three times as often as b).

### `stutter`

`stutter(gen: anything, n: number = 2, max: number = 0) -> anything`

Every value of a generator n times (or between n and max times, at random): stutter(degrees(...), 2).

### `shotgun`

`shotgun(gen: anything, p: number = 0.3) -> anything`

A generator with values dropped at random: each value is dropped with the chance p and the next one taken, so that it gets thinner in what it plays, not in length.

### `norepeat`

`norepeat(gen: anything) -> anything`

A generator of numbers that never plays the same value twice in a row.

### `distinct`

`distinct(gen: anything, n: number) -> anything`

A generator of numbers in groups of n values that are all different (a value that is in the group already is drawn again).

### `flatten`

`flatten(chords: generator of chords) -> generator of numbers`

The notes of a generator of chords, one after the other (an arpeggio of the chords).

### `stack`

`stack(voices: list of generator of numbers) -> generator of chords`

A chord of a value from each of the generators of numbers of the list: stack([degrees(...), degrees(...)]) is two voices that move on their own.

### `deal`

`deal(list: list of number) -> generator of numbers`

The numbers of a list in a random order, every one once, then again in another order (a shuffled deck).

### `pick`

`pick(values: list of number) -> generator of numbers`

Any one number of a list, a new one every time (the same as order(values, "random")).

### `order`

`order(values: list of number, how: string = in order) -> generator of numbers`

The numbers of a list in an order, round and round: alternate, backwards, converge, in order, random, shuffled, up and down. in order is 1 2 3 4; shuffled every value once before any again, in a new order each time; random any one every time; up and down up and back down without repeating the ends; converge from the outside in (1 4 2 3); alternate the first with each of the others in turn (1 2 1 3 1 4); backwards the list reversed.

### `uniform`

`uniform(lo: number, hi: number) -> generator of numbers`

Random numbers from lo up to hi, at every step.

## Numbers that move

### `wander`

`wander(lo: number, hi: number, step: number = 0.1, start: number = the preset's own, mode: string = bounce) -> generator of numbers`

A random walk: the number takes a step at random from where it is, of at most step (a part of the range, 0 to 1), and stays between lo and hi: at the edge it clips, bounces or jumps back (mode: bounce, clip, jump).

### `drift`

`drift(speed: number = 0.1, octaves: number = 1) -> generator of numbers`

Smooth noise from -1 to 1: it wanders without jumps (speed is how many wiggles a step, 0.1 is slow; octaves add finer detail).

### `wave`

`wave(period: number, lo: number = -1, hi: number = 1, shape: string = sine) -> generator of numbers`

A slow wave in steps: one cycle takes period steps, between lo and hi (shape: hold, saw, sine, smooth, square, triangle; hold is a new random value each cycle, smooth a smooth random one).

### `sweep`

`sweep(from: number, to: number, steps: number, curve: number = 1) -> generator of numbers`

A ramp in steps: from from to to in steps steps, then it stays at to (curve 1 is straight, above 1 slow at first, below 1 fast at first).

### `orbit`

`orbit(a: number = 3, b: number = 2, steps: number = 32, axis: number = 0, delta: number = 1.5707963267948966) -> generator of numbers`

A Lissajous curve as a number: two sines at the ratio a to b with a phase difference of delta (radians), one round in steps steps; axis 0 is x, 1 is y (use both for two sources that belong together). From -1 to 1.

### `chaos`

`chaos(system: string, p: number = the preset's own, q: number = the preset's own, axis: number = 0) -> generator of numbers`

A chaotic system as a number (system: logistic, henon, lorenz, rossler, duffing, pendulum, standardmap): it never repeats, and is not random. p and q set the system (logistic: p is the rate, 3.7 is chaotic; henon: p and q, 1.4 and 0.3; standardmap: p the kick, 1; lorenz and rossler: p is the speed of time, 0.01); axis picks which of its values (0 is x, 1 is y, 2 is z). The ranges differ per system: use remap.

### `remap`

`remap(gen: generator of numbers, inlo: number, inhi: number, lo: number, hi: number) -> generator of numbers`

A generator of numbers moved from one range to another: remap(chaos("lorenz"), -20, 20, 48, 72).

### `gauss`

`gauss(mean: number, sd: number = 1) -> generator of numbers`

Random numbers in a bell curve around mean, most of them within sd of it.

### `expo`

`expo(mean: number) -> generator of numbers`

Random numbers from an exponential curve with this mean: many small ones, now and then a big one.

### `beta`

`beta(a: number, b: number, lo: number = 0, hi: number = 1) -> generator of numbers`

Random numbers between lo and hi shaped by a and b: both below 1 gives mostly the edges, both above 1 the middle, a above b gives the high end.

### `cauchy`

`cauchy(location: number, scale: number = 1) -> generator of numbers`

Random numbers around location, half of them within scale of it, with now and then one very far away.

### `weibull`

`weibull(threshold: number, spread: number = 1, shape: number = 1) -> generator of numbers`

Random numbers from threshold up: shape below 1 gives many near the threshold and a long tail, 1 is exponential, above 1 a hump.

### `pink`

`pink(rows: number = 6) -> generator of numbers`

Random numbers from 0 to 1 with slow drifts and fast detail (1/f noise): between white noise and a random walk; rows is the number of octaves of detail.

## Rules, chance and growth

### `lsystem`

`lsystem(axiom: string, rules: list of string, depth: number, values: list of string) -> list of number`

An L-system as a list of numbers: the axiom is rewritten depth times with the rules, and the symbols that have a value are the numbers: lsystem("A", ["A=AB", "B=A"], 6, ["A=0", "B=2"]) is a phrase that is like itself at every size. A list is a phrase wherever a generator of numbers is wanted.

### `grammar`

`grammar(rules: list of string, start: string = S, depth: number = 4) -> generator of numbers`

A grammar that makes a new phrase every time round: rules are like "S=A B" (a name on the left, on the right the names of other rules and numbers), with "|" between the choices, each with a weight in front if you like ("A=3: 60 62 | 1: 67"). start is the rule to begin with, depth how far it is rewritten (a rule that refers to itself needs one).

### `cellular`

`cellular(rule: number, width: number = 16, start: string = one) -> generator of booleans`

A rhythm from an elementary cellular automaton (rule 0 to 255, 30 is chaotic, 90 a triangle of triangles, 110 complex): the cells of a row are the steps, and the next row comes after it. start: "one" is a single live cell in the middle, "random" starts from chance.

### `life`

`life(w: number = 16, h: number = 8, density: number = 0.3, row: number = 0, rule: string = B3/S23) -> generator of booleans`

A rhythm from Conway's game of life: a grid of w by h cells, grown from chance (density, 0 to 1); the cells of one row are the steps, and the grid goes on a generation after a row. rule is in B/S notation ("B3/S23" is Life).

### `scatter`

`scatter(chance: generator of numbers) -> generator of booleans`

Random hits: each step is a hit with the chance given by the generator of numbers (0 to 1): scatter(sweep(0, 0.8, 64)) gets denser. A number is the same chance every step.

### `markov`

`markov(values: list of number, weights: list of list of number, start: number = 0) -> generator of numbers`

A chain of chance: the values of the list are the states, and weights is a list for each of them of how likely each state is to come next (markov([60 62 67], [[0 3 1] [2 0 2] [1 1 0]]): after 60 comes 62 three times as often as 67). It begins at start (an item of the list, counted from 0).

### `learn`

`learn(example: list of number, order: number = 2) -> generator of numbers`

New music in the style of an example: the numbers of the list are the example (a melody, a bass line), and what comes out is made by what followed the last order values in it (2 or 3 stays close to it, 1 wanders). It goes on from the beginning of the example.

### `stroll`

`stroll(values: list of number, shape: list of number = []) -> generator of numbers`

A random walk through the numbers of a list: it starts at a place at random and each step goes to a neighbour. shape makes the list a grid (stroll(values, shape: [4 4]) walks a 4 by 4 grid, the values row by row), and it goes one step along one of the directions.

## Scales, chords and lines

### `scale`

`scale(steps: list of number) -> scale`

A scale of your own: the steps, in semitones above the root, from 0 and rising, within an octave (scale([0 2 3 7 8]) is a scale of 5 notes; fractions are microtones: scale([0 1.5 3 4.5 6 7.5 9 10.5])).

### `equal`

`equal(n: number) -> scale`

A scale that divides the octave in n equal steps: equal(12) is every semitone, equal(24) quarter tones, equal(19) and equal(31) are popular tunings. The pitches are in fractions of a semitone.

### `just`

`just(ratios: list of number) -> scale`

A scale from frequency ratios (just intonation): just([1, 9/8, 5/4, 4/3, 3/2, 5/3, 15/8]) written with the ratios as numbers (1.125, 1.25, ...); the first is 1, all are below 2.

### `stepwise`

`stepwise(root: number, scale: scale, start: number = 0, lo: number = -7, hi: number = 7, step: number = 1, leap: number = 0.1, maxleap: number = 4) -> generator of numbers`

A melody that walks over the degrees of a scale, mostly a step up or down, now and then a leap: it stays between the degrees lo and hi (bouncing off the edges), takes steps of at most step degrees, and with the chance leap a jump of up to maxleap degrees.

### `snap`

`snap(gen: generator of numbers, root: number, scale: scale) -> generator of numbers`

The numbers of a generator moved to the nearest note of a scale on a root: snap(remap(drift(), -1, 1, 48, 72), a2, minor) is a melody that wanders in A minor.

### `chord`

`chord(root: number, kind: string = major, inversion: number = 0, open: boolean = false) -> list of number`

The notes of a chord, as a list: chord(a3, "minor7"); kind is add9, aug, dim, dim7, dom7, dom9, halfdim, major, major6, major7, major9, minmaj7, minor, minor6, minor7, minor9, sus2, sus4; inversion turns it that many times (the lowest note goes an octave up), open spreads it over two octaves.

### `triads`

`triads(root: number, scale: scale, degrees: generator of whole numbers, size: number = 3) -> generator of chords`

The chords on the scale degrees that a generator gives, built of thirds of the scale: triads(a2, minor, [0 5 3 4]) is the chords i, vi, iv, v (size: 4 for seventh chords). The generator can be a progression.

### `voicelead`

`voicelead(chords: generator of chords, low: number = 48, high: number = 72) -> generator of chords`

Chords with smooth voice leading: each chord is turned and moved by octaves to be nearest the one before it, and kept between low and high.

### `arp`

`arp(notes: list of number, mode: string = up, mirror: string = none, reverse: boolean = false) -> generator of numbers`

An arpeggio of a list of notes (a chord): mode is alternate, converge, random, up; mirror (exclusive, inclusive, none) plays it back reversed after the first time; reverse starts from the top.

### `progression`

`progression(style: string = pop, start: number = 0) -> generator of whole numbers`

Chord changes by chance, as scale degrees (0 is I): a table of what can follow what, in the style of blues, classical, jazz, pop. It starts on degree start. Use it with triads: triads(a2, minor, progression("pop")).

### `tonerow`

`tonerow() -> list of number`

A twelve tone row: the pitch classes 0 to 11 in a random order (made when the patch is built, so seed it). Use it as notes, or through rowform.

### `rowform`

`rowform(row: list of number, form: string = prime, transpose: number = 0) -> list of number`

A form of a tone row (or any list of notes): form is invert, prime, retro, retroinvert, moved up transpose semitones.

### `counterpoint`

`counterpoint(cantus: list of number, below: boolean = false, maxleap: number = 9) -> list of number`

A line to go with another, written by the rules of strict counterpoint (note against note): the cantus is a list of notes (whole numbers), the line above it (or below with below: true) is consonant, moves by step and small leaps, and begins and ends on a perfect consonance. It is made when the patch is built, and by chance: seed it. If no line keeps all the rules the best one is given.

## Number sequences, time and structure

### `fibonacci`

`fibonacci(mod: number = 12) -> generator of whole numbers`

The Fibonacci numbers 0 1 1 2 3 5 8 ... each taken modulo mod (so they go round in a pattern): fibonacci(8) is 0 1 1 2 3 5 0 5 5 2 7 1 0 ...

### `thuemorse`

`thuemorse() -> generator of whole numbers`

The Thue-Morse sequence 0 1 1 0 1 0 0 1 ...: it never repeats and has no long runs. As degrees it is a fair melody of two notes, with divisible a rhythm.

### `primes`

`primes() -> generator of whole numbers`

The prime numbers 2 3 5 7 11 13 ...

### `collatz`

`collatz(start: number) -> generator of whole numbers`

The hailstone numbers from start: halve it when even, else three times it plus one, until it reaches 1, then again (27 climbs to 9232 before it falls).

### `divisible`

`divisible(numbers: generator of whole numbers, n: number = 2) -> generator of booleans`

A rhythm from a generator of whole numbers: a hit where the number can be divided by n (divisible(fibonacci(12), 2): a hit on the even numbers).

### `divide`

`divide(n: number, offset: number = 0) -> generator of booleans`

A hit on every nth step, from the step offset: divide(3) is x..x..x.., divide(4, 2) is ..x...x.

### `talea`

`talea(lengths: list of number) -> generator of booleans`

A rhythm from the lengths of its notes, in steps: talea([3 2 3]) is x..x.x.. (a note of 3 steps, one of 2, one of 3). With a list of notes of another length (the color) that goes round alongside it, the two come back together only after both have gone round whole times: isorhythm.

### `ratchets`

`ratchets(counts: list of number, size: number = 4) -> generator of booleans`

A rhythm of rolls: every beat of size steps has as many hits, equally spaced, as the next number of counts says (ratchets([1 2 1 4], 4): one hit, two, one, then four, in beats of 4 steps; a count above size is cut to size).

### `jump`

`jump(values: list of number, chunk: number = 3, step: number = 1, leap: number = 1) -> generator of numbers`

A way through a list: chunk values in a row, each step further along, then the start jumps leap places and another chunk begins; jump([0 1 2 3 4 5 6 7], 3, 1, 4) is 0 1 2, 4 5 6, 0 1 2 ... (the places go round the list).

### `split`

`split(value: number, lower: number = 0.25, prob: number = 0.5, branches: number = 2) -> list of number`

A length cut into related parts, as a list (for the beats of play): the value is divided into branches equal parts, and each part may be divided again with the chance prob, until a part would be shorter than lower. split(4, 0.5, 0.5, 2) is a half, then two quarters, one of which is two eighths, ... It is made when the patch is built, by chance: seed it.

### `metric`

`metric(value: number, n: number) -> list of number`

A length in n equal parts, as a list: metric(4, 3) is three parts of 1.333.

### `sections`

`sections(gens: list, steps: list of number, restart: boolean = false) -> anything`

Generators one after the other, each for a number of steps (a step is a value taken from it), then round again: sections([a, b], [64, 32]) is a for 64, b for 32. With restart: true each starts from the beginning when its turn comes, else it goes on where it was.

### `evolve`

`evolve(length: number = 8, lo: number = -7, hi: number = 7, smooth: number = 1, variety: number = 0, arch: number = 0, generations: number = 1, population: number = 40) -> generator of whole numbers`

A phrase of scale degrees that gets better as it plays: a population of random phrases of length notes between lo and hi is bred, and the best is played; after every round of the phrase it evolves generations more. What is better is what you weigh: smooth (small steps), variety (many different intervals), arch (a rise to a peak and a fall).

### `tendency`

`tendency(segments: list of list of number, loop: boolean = true) -> generator of numbers`

Random numbers inside a band that moves, a tendency mask of Koenig: segments is a list of [steps, lo1, hi1, lo2, hi2] (over so many steps the lower edge goes from lo1 to lo2 and the upper from hi1 to hi2): tendency([[16, 60, 62, 48, 72]]) opens from nearly one note to two octaves. loop: true goes round, else the last band stays.

## Scores (notes with a start and a length)

### `phrase`

`phrase(pitches: list of number, durations: list of number = [{number 1}], velocity: anything = 0.8, legato: anything = 1) -> score`

A score from a list of pitches that follow each other, each as long as the next of durations (in steps, going round): phrase([60 62 64 -1 67], [2 1 1 2 4]). A pitch below 0 is a rest. velocity and legato (the part of its duration a note is held) are a number, or a list with a value for each pitch (a rest has one too), going round: velocity: [1 0.5 0.8], legato: [1 0.5]. A score plays on any synth: a synth takes it where it takes notes.

### `part`

`part(text: string, chords: string = , key: string = C major, barsteps: number = 16, register: number = 48, bass: number = 36, velocity: number = 0.8, legato: number = 0.9, passes: number = 0) -> score`

A melody and its rhythm in one text, as a score: part("C3 Eb3 G3 - Bb3 . G3 Eb3"), a step each. A note is a name (C3, Eb3, f#4; C4 is 60) or a number, . a rest, - the note before held a step longer, C3+Eb3+G3 a chord, [C3 D3 Eb3] a group that fills a step, C3^3 a roll of three, *2 how many steps, ?25 a chance, %2/4 only some passes through it (the text is played passes times: 0 gives as many as the conditions need), ! as in a pattern. chords ("Am F C G": symbols, or roman numerals in key, a bar each of barsteps) gives chord tones to write: @1 the root, @2 the next tone, @3 the fifth, @4 the seventh, @-1 the tone below the root, @b the bass, @c the whole chord, ' after one an octave up and , down: "@1 @3 @5 @3" follows the chords. register is the lowest note of @1 (48), bass of @b (36). legato is how much of its step a note is held (0.9).

### `changes`

`changes(text: string, key: string = C major, lowest: number = 48, lead: boolean = false) -> list of list of number`

Chords written as text, as a list of chords: changes("Dm7 G7*2 Cmaj7") or roman numerals in a key: changes("i VI V7", key: "A minor"). A chord lasts a bar, or *2 bars (*3 and so on: whole bars), - holds the chord before for another bar, . is no chord (a rest). The chords are in root position from lowest up (48 is C3); lead: true gives each the voicing nearest the one before. Give them to chords: chords(changes("Am F C G"), every(16)).

### `midifile`

`midifile(file: string, track: number = -1) -> score`

A score from a MIDI file: its notes, in the steps of this patch (steps a beat; the tempo is that of the patch). track picks one track (counted from 0), -1 takes all of them together. harp check says how many tracks and notes there are. Look at the tracks one at a time to give each its own synth.

### `harmonize`

`harmonize(root: number, mode: string = major, degrees: list of number = [], n: number = 8, chordsteps: number = 16, voice: number = -1, seventh: boolean = false, sevenths: number = 0, secondary: number = 0, borrowed: number = 0, phrase: number = 4, cadence: string = , variety: number = 0, legato: number = 1, velocity: number = 0.8) -> score`

Chords written for four voices (soprano, alto, tenor, bass) by the rules of four part writing: no parallel fifths or octaves, no crossing voices, a leading tone that rises and a seventh that falls, small leaps. The chords are the scale degrees of degrees (0 is I, 3 is IV, 4 is V, counted from 0: [0 3 4 0]), or, when there are none, n chords that are made by the grammar of tonic, subdominant and dominant, in phrases that end in a cadence. root is the tonic (a note, any octave) and mode is major or minor. With explicit degrees seventh: true makes them seventh chords; for the chords that are made, sevenths, secondary (dominants of the next chord) and borrowed (iv in a major key) are chances from 0 to 1, phrase the chords to a phrase, cadence the way phrases end (authentic, half, plagal, deceptive; else half and authentic in turn). A chord lasts chordsteps steps. voice picks one voice (0 is the soprano, 3 the bass) to give a synth of its own, -1 takes all together. harp check says the chords, and the rules that had to be broken, if any.

### `isorhythm`

`isorhythm(talea: list of number, color: list of number, length: number) -> score`

An isorhythmic piece: the talea is a list of durations in steps (a negative one is a rest) and the color a list of pitches; the two go round at their own lengths, and come back together only after their lowest common multiple. length is how many steps it lasts.

### `chordtones`

`chordtones(chords: list of list of number, root: number, scale: scale, bars: number = 0, barsteps: number = 16, beatsteps: number = 4, maxleap: number = 3, low: number = 60, high: number = 84) -> score`

A melody over chords, one chord to a bar: on the first step of each beat a note of the chord (the first of the bar strongest), between them steps along the scale, and the last step of a bar a chord tone again. chords is a list of chords (lists of notes), repeated if there are more bars. By chance: seed it.

### `fugue`

`fugue(subject: list of number, durations: list of number, voices: number = 3, root: number, scale: scale, voice: number = -1, barsteps: number = 16) -> score`

The exposition of a fugue: a subject (pitches that are all notes of the key, and a duration in steps for each) enters in each of the voices in turn, as the subject and as the answer in the fifth, against a countersubject that is written for it by the rules of counterpoint. voice picks one voice (0 is the highest) to give its own synth, -1 takes all of them together. root and scale are the key; barsteps is the length of a bar. By chance: seed it.

### `phasing`

`phasing(pattern: score, drift: number, cycles: number = 0, voice: number = -1) -> score`

Phasing music as Steve Reich made it: a pattern (a score) is played by two voices, one of which is a little faster by drift steps every round, so that it slowly moves away from the other and after some rounds is in unison again. voice 0 is the steady one, 1 the one that moves, -1 both. cycles is how many rounds (0: until they are together again).

### `canon`

`canon(score: score, delay: number, interval: number = 0, voices: number = 2, voice: number = -1) -> score`

A canon: voices that all play the score, the next one delay steps later and interval semitones higher (a canon at the fifth after two bars of 16 steps is delay 32, interval 7). voice picks one (0 is the score itself), -1 takes all of them together.

### `transpose`

`transpose(score: score, semitones: number) -> score`

A score moved up by semitones.

### `stretch`

`stretch(score: score, factor: number) -> score`

A score longer (factor above 1, augmentation) or shorter (below 1, diminution): starts and lengths are multiplied.

### `mirror`

`mirror(score: score, axis: number, root: number = the preset's own, scale: scale = [0 1 2 3 4 5 6 7 8 9 10 11]) -> score`

A score turned upside down around the pitch axis: a note as far under it as it was over it. With a scale (and root) the mirror is in the scale, else every semitone counts.

### `retro`

`retro(score: score) -> score`

A score played backwards (the crab canon): the note that ended last starts first.

### `shift`

`shift(score: score, steps: number) -> score`

A score that starts later by steps (or earlier, with a negative number: what starts before the start is cut).

### `together`

`together(scores: list of score) -> score`

Scores that play at once, as one: together([bass, melody]).

### `cut`

`cut(score: score, from: number = 0, to: number) -> score`

The part of a score from step from up to step to, moved to start at 0.

### `tile`

`tile(score: score, n: number, length: number = 0) -> score`

A score repeated n times, each pass length steps after the one before (the length of the score if left out).

### `pad`

`pad(score: score, steps: number) -> score`

A score that goes round after steps steps: to leave silence at the end (or to cut the end short: the notes that start after it do not play).

### `tune`

`tune(score: score, root: number, scale: scale) -> score`

A score with every pitch moved to the nearest note of a scale on a root.

### `thin`

`thin(score: score, keep: generator of booleans) -> score`

A score with notes left out: a value of the rhythm for each note in turn, a note stays where it is a hit (thin(score, euclid(7, 5))).

### `pitches`

`pitches(score: score) -> generator of numbers`

The pitches of a score, one for each note in turn, going round: a field for play. When notes start together, the highest is taken (play is one voice a note).

### `lengths`

`lengths(score: score) -> generator of numbers`

How long each note of a score is, in steps, one for each note in turn: a field for play (sustain).

### `velocities`

`velocities(score: score) -> generator of numbers`

How hard each note of a score is, 0 to 1, one for each note in turn: a field for play.

### `onsets`

`onsets(score: score) -> generator of booleans`

The rhythm of a score: a hit on each step that a note starts on, going round: the rhythm of play, with the pitches of the score as a field: play(inst, rhythm: onsets(s), fields: [pitches(s)]).

### `chordsof`

`chordsof(score: score) -> generator of chords`

The notes of a score that start together, as chords, one for each step that a note starts on, going round: the chord field of play, with onsets as its rhythm: play(inst, rhythm: onsets(s), fields: [chordsof(s)], voices: 4). A step with one note is a chord of one.

## The form of a piece

### `song`

`song(sections: string, barsteps: number = 16) -> song`

The form of a piece: sections with their bars, song("Intro 8, Verse 8, Drop 16, Break 8, Drop 16, Outro 8"). barsteps is the steps of a bar (16). It goes round when it has gone through. Use it with during (a layer in and out), curve (a value over the sections) and bysection (other rhythms in other sections).

### `during`

`during(song: song, sections: string, glide: number = 2) -> signal`

1 in the sections of a song that are named, 0 elsewhere, as a signal: .gain(during(form, "Drop, Chorus")) puts a layer in and out. "not Intro" is all but that one, "all" every section. glide is how many steps the change takes, around the edge of a section (2).

### `curve`

`curve(song: song, curves: string) -> signal`

A value that moves through the sections of a song, as a signal: curve(form, "Intro: 0 -> 1 smooth, Drop: 1, Break: 1 -> 0.3"). A section is a value (held), or two with an arrow, and lin (the default), exp (between values above 0), smooth or step. A section with no curve keeps the value where the one before it ended; the sections that have a name twice share the curve.

### `bysection`

`bysection(song: song, sections: list of string, gens: list, restart: boolean = false) -> anything`

Other generators in other sections of a song: bysection(form, ["Intro, Verse", "Drop"], [every(4), euclid(16, 9)]) plays the first in the intro and the verse and the second in the drop (a section that is in no name plays nothing). It takes a value from the generator of the section for every step, so it is for rhythms; restart: true starts a generator from its start whenever its section comes.

## Generators as signals

### `seq`

`seq(values: anything, glide: number = 0) -> signal`

A signal that takes the next value of a list or a generator at every step, and holds it: seq([400 1200 800]). glide: so many milliseconds to move to the new value (0 steps at once). Where a signal is wanted, a generator can be given as it is (it steps). A generator with a name is one stream: used as a signal in several places it is pulled once a step and every place has the same values.

### `glide`

`glide(x: signal, ms: number = 30) -> signal`

Makes a signal that jumps (a generator, a seq) move to its new values in about ms milliseconds, with a one pole lowpass (psalm's smoother).

### `pulses`

`pulses(rhythm: generator of booleans, length: number = 0.5) -> signal`

A gate from a rhythm: 1 for length (0 to 1) of the step on every step where the rhythm gives true, 0 otherwise. Use it with on(env, gate) or as a multiplier.

## Notes on an instrument of your own

### `play`

`play(instrument: function, dur: anything = 0, rhythm: anything = 0, legato: number = 0.8, sustain: number = 0, fields: anything = [], voices: number = 1, swing: number = 0, humanize: number = 0, accent: anything = [], groove: string = , lanes: string = ) -> signal`

Plays notes on an instrument that is a function of the program, as a Pbind plays a SynthDef: the function is given the gate and then one number for each of the fields, and every note is a new run of it. when the notes are: dur: the beats of each note in turn (a list, or a generator of beats; a whole number of steps), or rhythm: a generator of booleans that is pulled every step (euclid(16, 5)): a note on every true, as long as the steps up to the next. legato: the part of it that the gate is high (or sustain: so many beats), fields: a list with, for each number the instrument takes, a list or a generator (degrees(...), mutate(...)) that is pulled once for every note, or a signal (seq(phrase), lfo(0.1)) that has its value read at the start of every note: so one stream can drive both the notes and an effect voices: notes go to the voices in turn, each of which is a run of the function of its own (random choices in it are made again for each), so that the end of a note goes on while the next one starts. The notes start at the block of 64 samples they fall in, as SuperCollider's messages do.

### `accents`

`accents(groups: list of number, downbeat: number = 1, group: number = 0.8, other: number = 0.55) -> list of number`

How strong each step of a bar is, by the groups of a meter, as a list for the accent of play: the first step of the bar is the strongest (downbeat), the first of every other group medium (group), the rest weak (other). accents([3 3 2]) is 8 steps in groups of 3, 3 and 2; accents([4 4 4 4]) a bar of 4 beats of 4 steps.

## Synths

### `subtractive`

`subtractive(notes: generator of notes, preset: string = pad, voices: number = 8) -> signal`

Psalm's subtractive synth playing notes, with one of its presets (see presets) and voices notes at once. Stereo out.

### `supersaw`

`supersaw(notes: generator of notes, preset: string = trance_pad, voices: number = 8) -> signal`

Psalm's supersaw, the sound of the JP-8000, playing notes, with one of its presets and voices notes at once. Stereo out.

### `modal`

`modal(notes: generator of notes, preset: string = marimba, voices: number = 8) -> signal`

Psalm's modal synth: struck and bowed bars, bells, bottles, cymbals and gongs as sets of resonances (marimba, vibraphone, church bell, ...), playing notes with voices notes at once. Stereo out. The presets are in docs/presets.md (harp presets modal).

### `piano`

`piano(notes: generator of notes, preset: string = grand, voices: number = 8) -> signal`

Psalm's piano: hammers, strings and a soundboard, playing notes with voices notes at once. Stereo out. The presets are in docs/presets.md (harp presets piano).

### `waveguide`

`waveguide(notes: generator of notes, preset: string = guitar, voices: number = 8) -> signal`

Psalm's physical models: plucked and bowed strings, flutes, drums and more as waves going up and down a line, playing notes with voices notes at once. Stereo out. The presets are in docs/presets.md (harp presets waveguide).

### `vocal`

`vocal(notes: generator of notes, preset: string = solo, voices: number = 8) -> signal`

Psalm's singing voice: a glottis and a vocal tract (solo, choir, humming, chants, ...), playing notes with voices notes at once. Stereo out. The presets are in docs/presets.md (harp presets vocal).

### `pulsar`

`pulsar(notes: generator of notes, preset: string = vowel_lead, voices: number = 8) -> signal`

Psalm's pulsar synthesis: trains of short pulses with formants (vowel leads, buzz bass, insects, ...), playing notes with voices notes at once. Stereo out. The presets are in docs/presets.md (harp presets pulsar).

### `shaper`

`shaper(notes: generator of notes, preset: string = acid, voices: number = 8) -> signal`

Psalm's waveshaping and phase distortion synth (cz leads, fm basses, pads, ...), playing notes with voices notes at once. Stereo out. The presets are in docs/presets.md (harp presets shaper).

### `organ`

`organ(notes: generator of notes, preset: string = jazz, drawbars: string = , leslie: number = the preset's own, speed: number = the preset's own) -> signal`

Psalm's tonewheel organ, one instrument: every note is a key of it (a chord is several keys, a note is held for its length), with its drawbars, percussion, vibrato and Leslie. The preset (harp presets organ) sets the registration; drawbars is the registration written as organists do, a digit from 0 to 8 for each of the nine drawbars ("888000000" is a jazz sound), leslie is how much of the rotary speaker there is (0 to 1) and speed how fast it turns (0 stopped, 0.5 slow, 1 fast). Stereo out.

### `dx`

`dx(notes: generator of notes, preset: string = e_piano, voices: number = 8, file: string = , patch: number = 0) -> signal`

Psalm's FM synth in the way of the Yamaha DX7 and TX81Z: operators that modulate each other (electric pianos, bells, brass, basses, pads, ...), playing notes with voices notes at once. The preset (harp presets dx) is one of its own patches; file is a DX7 SysEx bank (a .syx file of 32 patches) and patch the number of the patch in it, counted from 0, which takes the place of the preset. Stereo out.

### `drumsynth`

`drumsynth(notes: generator of notes, kit: string = 808, voices: number = 8) -> signal`

Psalm's drum synthesizer: drums that are made, not played from recordings. kit is 808, 909, electro, industrial, trap. A note is a drum, by its General MIDI key: drum("kick") is the number for a kick, drum("snare"), drum("hat"), drum("open"), drum("clap"), drum("tomLow") and so on (the names are in docs/synths.md); melody([drum("kick")], every(4)) plays it. The voices are made together, so that a closed hat stops an open one. Stereo out.

### `drum`

`drum(name: string) -> number`

The General MIDI key of a drum, by its name, for drumsynth: agogoHigh, agogoLow, bongoHigh, bongoLow, clap, clave, congaHigh, congaLow, congaMute, cowbell, crash, hat, kick, open, ride, rideBell, rim, shaker, snare, tambourine, timbaleHigh, timbaleLow, tomHigh, tomLow, tomMid, triangle, triangleMute, woodblockHigh, woodblockLow.

### `grains`

`grains(notes: generator of notes, file: string, root: number = 60, preset: string = choir_pad, voices: number = 6, position: number = the preset's own, scan: number = the preset's own, spray: number = the preset's own, density: number = the preset's own, size: number = the preset's own, pitchspread: number = the preset's own) -> signal`

Psalm's granular synth: every note is a stream of short grains of a recording (a sound file), played at the pitch of the note; root is the note at which the recording is at its own pitch. The preset (harp presets grains) sets the character: position (0-1, where in the recording), scan (how fast that moves: 0 holds, 1 is the original speed), spray (seconds of spread), density (grains a second), size (a grain, in ms), pitchspread (semitones) are the preset's own unless given. Stereo out.

### `wavetable`

`wavetable(notes: generator of notes, table: string = basic, preset: string = pad, voices: number = 6, position: anything = the preset's own, cutoff: anything = the preset's own, resonance: anything = the preset's own, warp: anything = the preset's own, warpmode: string = , detune: anything = the preset's own, spread: anything = the preset's own, sub: anything = the preset's own, size: number = 2048, frames: number = 32, amount: number = 0) -> signal`

Psalm's wavetable synth: an oscillator morphing through the frames of a table, with unison, a sub oscillator and a filter. The table is a sound file of single cycles of size samples each (2048 is usual), or one that psalm makes: basic, brightness, comb, drawbars, filter, fm, fold, odd_even, pwm, quantize, random, resonance, saturate, steps, sync, vowels (frames of them, and amount: the number of their own, like the sync ratio, the fm index or the pulse width; 0 is the usual one). The preset (harp presets wavetable) sets the rest. These can be a number, or a signal that moves them while the notes sound (lfo(0.1).range(0, 1), env(...), seq(...)): position (0 to 1, where in the table: the table moves), cutoff (Hz) and resonance of the filter, and, as a signal from 0 to 1 that is the amount of the preset's own setting: warp (bends the cycle: warpmode is benddown, bendup, fmb, fmsine, mirror, off, sync), detune (up to 100 cents), spread (the stereo width of the unison), sub (the level of the sub oscillator); up to 4 of those four can be signals. A signal is one channel. Stereo out.

## Recordings

### `sampler`

`sampler(notes: generator of notes, file: string, mode: string = , args: anything = [], first: number = 36, voices: number = 8, vol: number = 0, pan: number = 0, loop: number = -1, mipmaps: number = 0) -> signal`

Psalm's sampler playing notes with a recording: an SFZ instrument (file: "harp.sfz", as it is), or a sound file (wav, aiff, flac, ogg) in the way of mode: whole (the sound on one key, or the keys first to args[0] in tune at args[1]), hits (cut at the onsets: args: rise in dB, least gap in seconds, then from and to in seconds), change, pauses, verses, even (args: the number of pieces), chops, notes. The pieces are on the keys from first up (a note of that pitch plays the first piece). loop: seconds of crossfade makes it go round while the key is down. mipmaps: the number of lower resolutions made of each sound (4 is good), for cleaner high notes; the load takes much longer. Stereo out. See docs/recordings.md.

### `streamer`

`streamer(notes: generator of notes, file: string, key: number = 60, loop: boolean = false, hold: boolean = false, speed: number = 1, start: number = 0, end: number = 0, loopstart: number = 0, loopend: number = 0, release: number = 0.1, vol: number = 0, pan: number = 0, voices: number = 4) -> signal`

Psalm's streamer playing notes with a long recording that is read from disk while it plays (the memory it takes is its first half second): file is a sound file; key is the note at which it plays at its own pitch (a note above plays it faster); loop makes it go round (between loopstart and loopend, in seconds) while the key is down; hold lets the end of a note release it; start and end take a part of the file (seconds). Stereo out.

## While the patch is built (random numbers, lists, functions)

### `rand`

`rand(lo: number, hi: number) -> number`

A random number from lo up to hi, decided when the patch is built.

### `exprand`

`exprand(lo: number, hi: number) -> number`

A random number from lo up to hi, with the same chance for every ratio (ExpRand), decided when the patch is built. Neither can be 0.

### `coin`

`coin(p: number = 0.5) -> boolean`

True with the chance p (0 to 1), false otherwise: decided when the patch is built, for if.

### `choose`

`choose(items: list) -> anything`

One item of a list, picked at random when the patch is built.

### `geom`

`geom(n: number, lo: number, hi: number) -> list of number`

N numbers from lo to hi, each the same ratio from the one before: geom(40, 100, 4000).

### `fill`

`fill(f: function, n: number, with: anything = 0) -> list`

A list of what the function f gives for 0, 1, ... n-1 (and, if f takes a second argument, with): a function with random choices in it gives something else every time.

### `repeat`

`repeat(f: function, n: number, x: anything) -> anything`

Gives x to the function f, gives what comes out to f again, and so on, n times: repeat(f, 16, x). A function with random choices in it makes new ones every time.

### `dbamp`

`dbamp(db: number) -> number`

Decibels as an amplitude: -6 is 0.5 (about), 0 is 1.

## SuperCollider's own algorithms

### `scsaw`

`scsaw(freq: signal) -> signal`

SuperCollider's Saw as SuperCollider works it out: half the size, falling, the frequency read once in 64 samples, and it rings when the frequency jumps (saw does not). To mimic a patch of SuperCollider.

### `scsawsync`

`scsawsync(freq: signal, sync: signal) -> signal`

Scsaw that starts again from phase 0 when sync rises above 0, as a new synth of SuperCollider does for every note.

### `scsin`

`scsin(freq: signal, phase: number = 0) -> signal`

SuperCollider's SinOsc as it works it out: a table of 8192 values, a phase that is a whole number (so the frequency is a little off), the start phase in radians.

### `sclpf`

`sclpf(x: signal, freq: signal) -> signal`

SuperCollider's LPF: a second order lowpass, cutoff in Hz.

### `schpf`

`schpf(x: signal, freq: signal) -> signal`

SuperCollider's HPF: a second order highpass, cutoff in Hz.

### `scrlpf`

`scrlpf(x: signal, freq: signal, rq: signal = 1) -> signal`

SuperCollider's RLPF: a resonant lowpass, rq the reciprocal of Q (1 is flat, 0.5 resonates).

### `scpeq`

`scpeq(x: signal, freq: signal, rq: signal = 1, db: signal = 0) -> signal`

SuperCollider's BPeakEQ: boosts or cuts db decibel around freq, rq the bandwidth as a part of freq.

### `scdelayn`

`scdelayn(x: signal, max: number, time: signal) -> signal`

SuperCollider's DelayN: a delay of time seconds (up to max), cut to whole samples.

### `scdelayc`

`scdelayc(x: signal, max: number, time: signal) -> signal`

SuperCollider's DelayC: a delay of time seconds (up to max), with cubic interpolation, for a time that moves.

### `freeverb2`

`freeverb2(x: signal, mix: number = 0.33, room: number = 0.5, damp: number = 0.5) -> signal`

SuperCollider's FreeVerb2: the reverb of Freeverb for a signal of two channels; mix, room and damp from 0 to 1.
