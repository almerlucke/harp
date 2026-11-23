# Settings of the synths

<!-- made by `harp settings -md`: do not edit by hand (a test says when it is out of date) -->

Every setting of a synth is an optional parameter of its function, with the name psalm gives it: `subtractive(notes, preset: "pad", cutoff: 900, filterMode: "band pass", ampAttack: 400)`. A knob is a number, or a signal that moves it while the notes sound (`cutoff: lfo(0.1).range(300, 3000)`); a choice is a name (or its number); a switch is `true` or `false`. What is not given is the preset's own. The envelopes are a setting for each stage, in milliseconds (`ampAttack`, `ampDecay`, `ampSustain` from 0 to 1, `ampRelease`).


## subtractive (67)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations pitch, cutoff, resonance, pulse width, level 1, level 2, sub, noise, detune 2, fm, drive, vowel, level, lfo 1 rate, lfo 2 rate, unison detune, cutoff 2

**Oscillators**: `osc1Waveform` (sine, cosine, triangle, square, rectangle, saw, ramp, mod triangle, mod square, half sine, full sine, tri pulse, trapezoid, trapezoid var); `osc1Level` (0 to 1); `osc2Waveform` (sine, cosine, triangle, square, rectangle, saw, ramp, mod triangle, mod square, half sine, full sine, tri pulse, trapezoid, trapezoid var); `osc2Level` (0 to 1); `osc2Octave` (-3 to 3); `osc2Detune` (-24 to 24 st); `pulseWidth` (0.05 to 0.95); `subLevel` (0 to 1); `noiseLevel` (0 to 1); `noiseType` (white, pink); `ringModLevel` (0 to 1); `sync` (true or false); `fmAmount` (0 to 5); `drive` (0.5 to 8); `unison` (1 to 7); `unisonDetune` (0 to 100 ct); `drift` (0 to 50 ct)

**Filter**: `filterType` (state variable, ladder, formant); `filterMode` (low pass, band pass, high pass, notch, peak, all pass); `ladderMode` (low 24, low 12, band 24, band 12, high 24, high 12); `cutoff` (20 to 18000 Hz); `resonance` (0 to 1); `filterEnvAmount` (-6 to 6 oct); `keyTrack` (0 to 1); `filterDrive` (1 to 8); `filterOversample` (once, 2x, 4x); `vowel` (0 to 4); `formantShift` (0.5 to 2)

**Filter 2**: `filter2Mode` (off, low pass, band pass, high pass, notch); `filter2Cutoff` (20 to 18000 Hz); `filter2Resonance` (0 to 1); `filter2EnvAmount` (-6 to 6 oct); `filter2KeyTrack` (0 to 1); `filter2Parallel` (true or false)

**Pitch**: `glide` (0 to 2000 ms); `glideMode` (exponential, constant time, constant rate); `filterEnvToPitch` (-48 to 48 st); `level` (0 to 4)

**LFO 1**: `lfo1Waveform` (sine, triangle, saw, ramp, square, sample & hold); `lfo1Rate` (0.01 to 40 Hz); `lfo1Delay` (0 to 3000 ms); `lfo1KeySync` (true or false); `lfo1Pitch` (-12 to 12 st); `lfo1Cutoff` (-4 to 4 oct); `lfo1PulseWidth` (0 to 0.5); `lfo1Amp` (0 to 1)

**LFO 2**: `lfo2Waveform` (sine, triangle, saw, ramp, square, sample & hold); `lfo2Rate` (0.01 to 40 Hz); `lfo2Delay` (0 to 3000 ms); `lfo2KeySync` (true or false); `lfo2Pitch` (-12 to 12 st); `lfo2Cutoff` (-4 to 4 oct); `lfo2PulseWidth` (0 to 0.5); `lfo2Amp` (0 to 1)

**Modulation**: `ext1` (0 to 1); `ext2` (0 to 1); `ext3` (0 to 1); `ext4` (0 to 1)

****: `mods` (0 to 0)

**Envelopes**: `ampAttack` (0.1 to 60000 ms); `ampDecay` (0.1 to 60000 ms); `ampSustain` (0 to 1); `ampRelease` (0.1 to 60000 ms); `filterAttack` (0.1 to 60000 ms); `filterDecay` (0.1 to 60000 ms); `filterSustain` (0 to 1); `filterRelease` (0.1 to 60000 ms)

## supersaw (30)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations detune, mix, stereo, sub, cutoff, resonance, pitch, level, lfo rate, ensemble

**Saws**: `saws` (1 to 16); `detune` (0 to 1); `mix` (0 to 1); `stereo` (0 to 1); `sub` (0 to 1)

**Filter**: `cutoff` (20 to 18000 Hz); `resonance` (0 to 1); `hpCutoff` (20 to 2000 Hz); `filterEnvAmount` (-6 to 6 oct)

**LFO**: `lfoRate` (0 to 20 Hz); `lfoCutoff` (0 to 4 oct); `lfoDetune` (0 to 1); `lfo2Rate` (0 to 20 Hz); `lfo2Shape` (sine, triangle)

**Ensemble**: `ensemble` (0 to 1); `ensembleRate` (0.05 to 8 Hz)

**Voice**: `level` (0 to 4)

**Modulation**: `ext1` (0 to 1); `ext2` (0 to 1); `ext3` (0 to 1); `ext4` (0 to 1)

****: `mods` (0 to 0)

**Envelopes**: `ampAttack` (0.1 to 60000 ms); `ampDecay` (0.1 to 60000 ms); `ampSustain` (0 to 1); `ampRelease` (0.1 to 60000 ms); `filterAttack` (0.1 to 60000 ms); `filterDecay` (0.1 to 60000 ms); `filterSustain` (0 to 1); `filterRelease` (0.1 to 60000 ms)

## wavetable (36)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations position, cutoff, resonance, pitch, detune, spread, sub, level, warp, level b, position b, detune b, lfo 1 rate, lfo 2 rate

**Position**: `positionEnvAmount` (-1 to 1); `positionLFORate` (0.01 to 20 Hz); `positionLFODepth` (0 to 1); `normalize` (true or false)

**Oscillator**: `unison` (1 to 7); `subLevel` (0 to 1); `oversample` (once, 2x, 4x)

**Oscillator B**: `oscBLevel` (0 to 1); `oscBPosition` (0 to 1); `oscBOctave` (-3 to 3); `oscBSemi` (-12 to 12 st); `oscBDetune` (-50 to 50 ct); `oscBUnison` (1 to 7); `oscBSpread` (0 to 100 ct)

**Modulation**: `lfo1Rate` (0.01 to 20 Hz); `lfo2Rate` (0.01 to 20 Hz)

**Filter**: `filter` (off, low pass, band pass, high pass); `filterEnvAmount` (-6 to 6 oct); `keyTrack` (0 to 1)

**Pitch**: `vibratoRate` (0.1 to 20 Hz); `vibratoDepth` (0 to 2 st); `glide` (0 to 2000 ms); `level` (0 to 4)

****: `mods` (0 to 0)

**Envelopes**: `ampAttack` (0.1 to 60000 ms); `ampDecay` (0.1 to 60000 ms); `ampSustain` (0 to 1); `ampRelease` (0.1 to 60000 ms); `filterAttack` (0.1 to 60000 ms); `filterDecay` (0.1 to 60000 ms); `filterSustain` (0 to 1); `filterRelease` (0.1 to 60000 ms); `positionAttack` (0.1 to 60000 ms); `positionDecay` (0.1 to 60000 ms); `positionSustain` (0 to 1); `positionRelease` (0.1 to 60000 ms)

## modal (36)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations pitch, decay, hardness, position, tremolo depth, beat, noise, level, cutoff, resonance, inharmonicity, high damping

**Instrument**: `instrument` (custom, bar, marimba, vibraphone, xylophone, glockenspiel, tine, tubular bell, church bell, handbell, plate, gong, cymbal, singing bowl, steel pan, wood block, temple block, bottle); `resonator` (0 to 1); `spread` (0 to 4); `drift` (0 to 1); `undercut` (0.01 to 1); `nonlinearity` (0 to 1)

**Mallet**: `excitation` (strike, bow, noise burst); `hardness` (0 to 1); `position` (0.01 to 1); `noise` (0 to 1)

**Contact**: `contact` (0 to 1); `mass` (0 to 1)

**Body**: `highDamping` (0 to 2); `inharmonicity` (0 to 1)

**Tremolo**: `tremolo` (0 to 15 Hz); `tremoloDepth` (0 to 1)

**Modulation**: `lfoRate` (0.1 to 20 Hz); `lfo2Rate` (0.05 to 20 Hz); `ext1` (0 to 1); `ext2` (0 to 1); `ext3` (0 to 1); `ext4` (0 to 1)

**Output**: `level` (0 to 4)

**Filter**: `filterType` (off, low pass, band pass, high pass, notch, ladder 24, ladder 12); `cutoff` (20 to 18000 Hz); `resonance` (0 to 1); `filterEnvAmount` (-6 to 6 oct); `keyTrack` (0 to 1); `filterDrive` (1 to 8); `filterAttack` (0.1 to 60000 ms); `filterDecay` (0.1 to 60000 ms); `filterSustain` (0 to 1); `filterRelease` (0.1 to 60000 ms)

****: `mods` (0 to 0)

**Envelopes**: `ampDecay` (0.1 to 60000 ms); `ampRelease` (0.1 to 60000 ms)

## piano (25)

**Hammer**: `hardness` (0 to 1); `strikePosition` (0.02 to 0.5); `velocitySens` (0 to 1); `variation` (0 to 1); `strikeRegister` (0 to 1)

**Strings**: `stiffness` (0 to 3); `decay` (0.05 to 4); `damping` (0 to 4); `unison` (0 to 20 ct); `strings` (1 to 3); `stretch` (0 to 2); `bark` (0 to 1); `wound` (0 to 1); `precursor` (0 to 1); `resonance` (0 to 1); `damperTime` (0.02 to 2 s)

**Body**: `soundboard` (0 to 1); `body` (0 to 1); `bridge` (0 to 1); `mechanics` (0 to 1); `keyBed` (0 to 1)

**Output**: `position` (player, studio, hall); `width` (0 to 1); `treble` (-24 to 12 dB); `level` (0 to 32)

## waveguide (46)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations decay, damping, brightness, pick position, pressure, bow position, vibrato depth, body, pitch, level, cutoff, breath, embouchure, bow force, bow speed

**Pluck**: `damping` (0 to 1); `brightness` (0 to 1); `pickPosition` (0 to 0.5); `pluckWidth` (0 to 0.2); `edge` (0 to 1); `plectrum` (0 to 1); `harmonic` (0 to 0.5); `fretNoise` (0 to 1); `fretBuzz` (0 to 1); `fretPosition` (0.05 to 0.98); `slide` (0 to 2 s)

**String**: `inharmonicity` (0 to 0.005); `polarization` (0 to 1); `tensionMod` (0 to 1); `bodyCoupling` (0 to 1)

**Bow**: `bowModel` (table, friction); `bowPressure` (0 to 1); `bowPosition` (0.02 to 0.5); `bowForce` (0 to 1); `bowSpeed` (0 to 1); `bowNoise` (0 to 1); `bowReverse` (0 to 4 s); `vibratoRate` (0 to 12 Hz); `vibratoDepth` (0 to 2 st)

**Output**: `bodyMix` (0 to 1); `sympathetic` (0 to 1); `level` (0 to 4)

**Filter**: `filterType` (off, low pass, band pass, high pass, notch, ladder 24, ladder 12); `cutoff` (20 to 18000 Hz); `resonance` (0 to 1); `filterEnvAmount` (-6 to 6 oct); `keyTrack` (0 to 1); `filterDrive` (1 to 8); `filterAttack` (0.1 to 60000 ms); `filterDecay` (0.1 to 60000 ms); `filterSustain` (0 to 1); `filterRelease` (0.1 to 60000 ms)

**Modulation**: `ext1` (0 to 1); `ext2` (0 to 1); `ext3` (0 to 1); `ext4` (0 to 1); `lfo2Rate` (0.05 to 20 Hz)

****: `mods` (0 to 0)

**Envelopes**: `ampAttack` (0.1 to 60000 ms); `ampDecay` (0.1 to 60000 ms); `ampRelease` (0.1 to 60000 ms)

## vocal (47)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations vowel, brightness, breath, vibrato depth, detune, spread, pitch, level, cutoff, whisper

**Voice**: `range` (auto, bass, tenor, alto, soprano, child); `vowel` (0 to 4); `vowelGlide` (0 to 2 s); `brightness` (0 to 1); `breath` (0 to 1); `formantShift` (0.25 to 4); `phoneme` (scale, i, ɪ, e, ɛ, æ, a, ɔ, o, ʊ, u, ʌ, ɜ); `formantTuning` (0 to 1); `training` (0 to 1); `nasal` (0 to 1)

**Diphthong**: `phonemeTo` (none, i, ɪ, e, ɛ, æ, a, ɔ, o, ʊ, u, ʌ, ɜ); `diphthongTime` (0 to 2 s); `diphthongHold` (0 to 2 s)

**Source**: `source` (rosenberg-klatt, lf); `rd` (0 to 2.7); `whisper` (0 to 1); `subglottal` (0 to 1); `openQuotient` (0 to 0.95); `speedQuotient` (0 to 4); `returnPhase` (0 to 1); `jitter` (0 to 0.05); `shimmer` (0 to 0.3)

**Vibrato**: `vibratoRate` (0 to 12 Hz); `vibratoDepth` (0 to 2 st); `naturalVibrato` (0 to 1)

**Consonant**: `consonant` (none, t, k, s, sh, f, m, n, ng, l, r, w, y, p, b, d, g, z, zh, v); `consonantLevel` (0 to 1); `coarticulation` (0.005 to 0.2 s)

**Word**: `wordSync` (0 to 4 beats)

**Output**: `level` (0 to 4)

**Filter**: `filterType` (off, low pass, band pass, high pass, notch, ladder 24, ladder 12); `cutoff` (20 to 18000 Hz); `resonance` (0 to 1); `filterEnvAmount` (-6 to 6 oct); `keyTrack` (0 to 1); `filterDrive` (1 to 8); `filterAttack` (0.1 to 60000 ms); `filterDecay` (0.1 to 60000 ms); `filterSustain` (0 to 1); `filterRelease` (0.1 to 60000 ms)

**Modulation**: `ext1` (0 to 1); `ext2` (0 to 1); `ext3` (0 to 1); `ext4` (0 to 1)

****: `mods` (0 to 0)

**Envelopes**: `ampAttack` (0.1 to 60000 ms); `ampRelease` (0.1 to 60000 ms)

## pulsar (45)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations formant, formant 2, duty, masking, jitter, pulse spread, pitch, level, cutoff, resonance, shape mix, lfo rate

**Pulse**: `duty` (0.01 to 1); `window` (hann, gaussian, percussive, rectangle, tapered); `pulsaret2` (saw, square, triangle); `shapeMix` (0 to 1); `burst` (0 to 16); `burstRest` (0 to 16); `masking` (0 to 1); `jitter` (0 to 1); `stereoSpread` (0 to 1); `noteSpread` (0 to 1)

**Formant**: `formant` (50 to 12000 Hz); `formant2` (0 to 8000 Hz); `formant2Level` (0 to 1); `formantTracking` (0 to 1); `formantEnvAmount` (-6 to 6 oct); `formantLFORate` (0.01 to 20 Hz); `formantLFODepth` (0 to 2 oct)

**Pitch**: `vibratoRate` (0.1 to 20 Hz); `vibratoDepth` (0 to 12 st); `glide` (0 to 2000 ms); `level` (0 to 4)

**Modulation**: `lfo2Rate` (0.01 to 20 Hz); `ext1` (0 to 1); `ext2` (0 to 1); `ext3` (0 to 1); `ext4` (0 to 1)

**Filter**: `filterType` (off, low pass, band pass, high pass, notch, ladder 24, ladder 12); `cutoff` (20 to 18000 Hz); `resonance` (0 to 1); `filterEnvAmount` (-6 to 6 oct); `keyTrack` (0 to 1); `filterDrive` (1 to 8); `filterAttack` (0.1 to 60000 ms); `filterDecay` (0.1 to 60000 ms); `filterSustain` (0 to 1); `filterRelease` (0.1 to 60000 ms)

****: `mods` (0 to 0)

**Envelopes**: `ampAttack` (0.1 to 60000 ms); `ampDecay` (0.1 to 60000 ms); `ampSustain` (0 to 1); `ampRelease` (0.1 to 60000 ms); `formantAttack` (0.1 to 60000 ms); `formantDecay` (0.1 to 60000 ms); `formantSustain` (0 to 1); `formantRelease` (0.1 to 60000 ms)

## shaper (72)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations morph, macro 1, macro 2, macro 3, macro 4, pitch, vibrato, width, level, detune 2, cutoff, resonance, cross mod

**Morph**: `morph` (0 to 1); `macro1` (0 to 1); `macro2` (0 to 1); `macro3` (0 to 1); `macro4` (0 to 1); `morphAmount` (-1 to 1); `shapeAmount` (-1 to 1)

**Stereo**: `stereoMorph` (-1 to 1); `stereoMacro1` (-1 to 1); `stereoMacro2` (-1 to 1); `stereoMacro3` (-1 to 1); `stereoMacro4` (-1 to 1)

**Line 1**: `octave1` (-4 to 4); `detune1` (-1200 to 1200 ct); `level1` (0 to 2); `morph1` (-1 to 1)

**Line 2**: `octave2` (-4 to 4); `detune2` (-1200 to 1200 ct); `level2` (0 to 2); `morph2` (-1 to 1)

**Fullness**: `unison` (1 to 7); `spread` (0 to 100 ct); `uniStereo` (0 to 1); `drift` (0 to 1); `sub` (0 to 1); `noise` (0 to 1); `noiseColor` (-1 to 1)

**Cross**: `xmod` (0 to 1); `xmodType` (phase, ring); `xmodSilent` (true or false)

**Voice**: `oversample` (auto (tables), off, 2x, 4x, 8x); `pitchAmount` (-48 to 48 st); `glide` (0 to 1000 ms); `width` (0 to 1); `vibratoDepth` (0 to 2 st); `vibratoRate` (0.1 to 12 Hz); `lfo1Rate` (0 to 20 Hz); `lfo2Rate` (0 to 20 Hz); `lfo2Shape` (sine, triangle); `level` (0 to 4); `trim` (-24 to 12 dB)

**Filter**: `filterType` (off, low pass, band pass, high pass, notch, ladder 24, ladder 12); `cutoff` (20 to 18000 Hz); `resonance` (0 to 1); `filterEnvAmount` (-6 to 6 oct); `keyTrack` (0 to 1); `filterDrive` (1 to 8); `filterAttack` (0.1 to 60000 ms); `filterDecay` (0.1 to 60000 ms); `filterSustain` (0 to 1); `filterRelease` (0.1 to 60000 ms)

**Modulation**: `ext1` (0 to 1); `ext2` (0 to 1); `ext3` (0 to 1); `ext4` (0 to 1)

****: `mods` (0 to 0)

**Envelopes**: `ampAttack` (0.1 to 60000 ms); `ampDecay` (0.1 to 60000 ms); `ampSustain` (0 to 1); `ampRelease` (0.1 to 60000 ms); `morphEnvAttack` (0.1 to 60000 ms); `morphEnvDecay` (0.1 to 60000 ms); `morphEnvSustain` (0 to 1); `morphEnvRelease` (0.1 to 60000 ms); `shapeEnvAttack` (0.1 to 60000 ms); `shapeEnvDecay` (0.1 to 60000 ms); `shapeEnvSustain` (0 to 1); `shapeEnvRelease` (0.1 to 60000 ms); `pitchEnvAttack` (0.1 to 60000 ms); `pitchEnvDecay` (0.1 to 60000 ms); `pitchEnvSustain` (0 to 1); `pitchEnvRelease` (0.1 to 60000 ms)

## organ (41)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations drive, click, vibrato depth, level, cutoff, resonance, percussion level, leakage, pitch, tone, leslie, leslie speed

**Drawbars**: `drawbar1` (0 to 8); `drawbar2` (0 to 8); `drawbar3` (0 to 8); `drawbar4` (0 to 8); `drawbar5` (0 to 8); `drawbar6` (0 to 8); `drawbar7` (0 to 8); `drawbar8` (0 to 8); `drawbar9` (0 to 8)

**Percussion**: `percussion` (off, second, third); `percFast` (true or false); `percSoft` (true or false)

**Vibrato**: `vibrato` (off, V1, V2, V3, C1, C2, C3); `vibratoHz` (0.5 to 20 Hz); `vibratoScale` (0.25 to 6)

**Modulation**: `lfo2Rate` (0.05 to 20 Hz); `ext1` (0 to 1); `ext2` (0 to 1); `ext3` (0 to 1); `ext4` (0 to 1)

**Generator**: `tonewheel` (0 to 1); `contactSpread` (0 to 10 ms); `leakage` (0 to 1)

**Leslie**: `leslieSpeed` (0 to 1); `lesliePosition` (0 to 1)

**Output**: `click` (0 to 1); `clickRand` (0 to 1); `drive` (0 to 1); `tone` (0 to 1); `level` (0 to 4)

**Filter**: `filterType` (off, low pass, band pass, high pass, notch, ladder 24, ladder 12); `cutoff` (20 to 18000 Hz); `resonance` (0 to 1); `filterEnvAmount` (-6 to 6 oct); `keyTrack` (0 to 1); `filterDrive` (1 to 8); `filterAttack` (0.1 to 60000 ms); `filterDecay` (0.1 to 60000 ms); `filterSustain` (0 to 1); `filterRelease` (0.1 to 60000 ms)

****: `mods` (0 to 0)

## grains (12)

**Sample**: `rootNote` (0 to 127)

**Grains**: `grainSize` (5 to 1000 ms); `shape` (0 to 1); `jitter` (0 to 1); `reverse` (0 to 1)

**Spread**: `octave` (0 to 1); `stereoSpread` (0 to 1)

**Output**: `level` (0 to 16)

**Envelopes**: `ampAttack` (0.1 to 60000 ms); `ampDecay` (0.1 to 60000 ms); `ampSustain` (0 to 1); `ampRelease` (0.1 to 60000 ms)

## dx (197)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations mod depth, feedback, op 1 gain, op 2 gain, op 3 gain, op 4 gain, op 5 gain, op 6 gain, pitch, level, cutoff (filter), resonance (filter), lfo 2 rate

**Voice**: `algorithm` (1 to 32); `feedback` (0 to 7); `transpose` (0 to 48); `oversample` (0 to 2); `oscSync` (true or false)

**LFO**: `lfoWaveform` (0 to 5); `lfoSpeed` (0 to 99); `lfoDelay` (0 to 99); `lfoPitchDepth` (0 to 99); `lfoAmpDepth` (0 to 99); `lfoSync` (true or false); `pitchModSens` (0 to 7)

**Pitch envelope**: `pitchRate1` (0 to 99); `pitchRate2` (0 to 99); `pitchRate3` (0 to 99); `pitchRate4` (0 to 99); `pitchLevel1` (0 to 99); `pitchLevel2` (0 to 99); `pitchLevel3` (0 to 99); `pitchLevel4` (0 to 99)

**Filter**: `filterType` (off, low pass, band pass, high pass, notch, ladder 24, ladder 12); `filterCutoff` (0 to 99); `filterResonance` (0 to 99); `filterEnvAmount` (0 to 99); `filterKeyTrack` (0 to 99); `filterAttack` (0 to 99); `filterDecay` (0 to 99); `filterSustain` (0 to 99); `filterRelease` (0 to 99)

**Operator 1**: `op1Coarse` (0 to 31); `op1Fine` (0 to 99); `op1Detune` (0 to 14); `op1Wave` (0 to 7); `op1Exact` (true or false); `op1Level` (0 to 99); `op1Fixed` (true or false); `op1VelocitySens` (0 to 7); `op1RateScaling` (0 to 7); `op1AmpModSens` (0 to 3); `op1Rate1` (0 to 99); `op1Rate2` (0 to 99); `op1Rate3` (0 to 99); `op1Rate4` (0 to 99); `op1Level1` (0 to 99); `op1Level2` (0 to 99); `op1Level3` (0 to 99); `op1Level4` (0 to 99); `op1BreakPoint` (0 to 99); `op1LeftDepth` (0 to 99); `op1RightDepth` (0 to 99); `op1LeftCurve` (0 to 3); `op1RightCurve` (0 to 3); `op1Ratio` (0.125 to 32); `op1Cents` (-100 to 100 ct); `op1Gain` (0 to 2)

**Operator 2**: `op2Coarse` (0 to 31); `op2Fine` (0 to 99); `op2Detune` (0 to 14); `op2Wave` (0 to 7); `op2Exact` (true or false); `op2Level` (0 to 99); `op2Fixed` (true or false); `op2VelocitySens` (0 to 7); `op2RateScaling` (0 to 7); `op2AmpModSens` (0 to 3); `op2Rate1` (0 to 99); `op2Rate2` (0 to 99); `op2Rate3` (0 to 99); `op2Rate4` (0 to 99); `op2Level1` (0 to 99); `op2Level2` (0 to 99); `op2Level3` (0 to 99); `op2Level4` (0 to 99); `op2BreakPoint` (0 to 99); `op2LeftDepth` (0 to 99); `op2RightDepth` (0 to 99); `op2LeftCurve` (0 to 3); `op2RightCurve` (0 to 3); `op2Ratio` (0.125 to 32); `op2Cents` (-100 to 100 ct); `op2Gain` (0 to 2)

**Operator 3**: `op3Coarse` (0 to 31); `op3Fine` (0 to 99); `op3Detune` (0 to 14); `op3Wave` (0 to 7); `op3Exact` (true or false); `op3Level` (0 to 99); `op3Fixed` (true or false); `op3VelocitySens` (0 to 7); `op3RateScaling` (0 to 7); `op3AmpModSens` (0 to 3); `op3Rate1` (0 to 99); `op3Rate2` (0 to 99); `op3Rate3` (0 to 99); `op3Rate4` (0 to 99); `op3Level1` (0 to 99); `op3Level2` (0 to 99); `op3Level3` (0 to 99); `op3Level4` (0 to 99); `op3BreakPoint` (0 to 99); `op3LeftDepth` (0 to 99); `op3RightDepth` (0 to 99); `op3LeftCurve` (0 to 3); `op3RightCurve` (0 to 3); `op3Ratio` (0.125 to 32); `op3Cents` (-100 to 100 ct); `op3Gain` (0 to 2)

**Operator 4**: `op4Coarse` (0 to 31); `op4Fine` (0 to 99); `op4Detune` (0 to 14); `op4Wave` (0 to 7); `op4Exact` (true or false); `op4Level` (0 to 99); `op4Fixed` (true or false); `op4VelocitySens` (0 to 7); `op4RateScaling` (0 to 7); `op4AmpModSens` (0 to 3); `op4Rate1` (0 to 99); `op4Rate2` (0 to 99); `op4Rate3` (0 to 99); `op4Rate4` (0 to 99); `op4Level1` (0 to 99); `op4Level2` (0 to 99); `op4Level3` (0 to 99); `op4Level4` (0 to 99); `op4BreakPoint` (0 to 99); `op4LeftDepth` (0 to 99); `op4RightDepth` (0 to 99); `op4LeftCurve` (0 to 3); `op4RightCurve` (0 to 3); `op4Ratio` (0.125 to 32); `op4Cents` (-100 to 100 ct); `op4Gain` (0 to 2)

**Operator 5**: `op5Coarse` (0 to 31); `op5Fine` (0 to 99); `op5Detune` (0 to 14); `op5Wave` (0 to 7); `op5Exact` (true or false); `op5Level` (0 to 99); `op5Fixed` (true or false); `op5VelocitySens` (0 to 7); `op5RateScaling` (0 to 7); `op5AmpModSens` (0 to 3); `op5Rate1` (0 to 99); `op5Rate2` (0 to 99); `op5Rate3` (0 to 99); `op5Rate4` (0 to 99); `op5Level1` (0 to 99); `op5Level2` (0 to 99); `op5Level3` (0 to 99); `op5Level4` (0 to 99); `op5BreakPoint` (0 to 99); `op5LeftDepth` (0 to 99); `op5RightDepth` (0 to 99); `op5LeftCurve` (0 to 3); `op5RightCurve` (0 to 3); `op5Ratio` (0.125 to 32); `op5Cents` (-100 to 100 ct); `op5Gain` (0 to 2)

**Operator 6**: `op6Coarse` (0 to 31); `op6Fine` (0 to 99); `op6Detune` (0 to 14); `op6Wave` (0 to 7); `op6Exact` (true or false); `op6Level` (0 to 99); `op6Fixed` (true or false); `op6VelocitySens` (0 to 7); `op6RateScaling` (0 to 7); `op6AmpModSens` (0 to 3); `op6Rate1` (0 to 99); `op6Rate2` (0 to 99); `op6Rate3` (0 to 99); `op6Rate4` (0 to 99); `op6Level1` (0 to 99); `op6Level2` (0 to 99); `op6Level3` (0 to 99); `op6Level4` (0 to 99); `op6BreakPoint` (0 to 99); `op6LeftDepth` (0 to 99); `op6RightDepth` (0 to 99); `op6LeftCurve` (0 to 3); `op6RightCurve` (0 to 3); `op6Ratio` (0.125 to 32); `op6Cents` (-100 to 100 ct); `op6Gain` (0 to 2)

**Exact**: `modDepth` (0.05 to 4); `freeFeedback` (0 to 1); `glide` (0 to 2000 ms); `vibratoRate` (0.1 to 20 Hz); `vibratoDepth` (0 to 2 st); `vibratoDelay` (0 to 3000 ms)

**Modulation**: `lfo2Rate` (0 to 40 Hz); `ext1` (0 to 1); `ext2` (0 to 1); `ext3` (0 to 1); `ext4` (0 to 1)

****: `mods` (0 to 0)

## drumSynth (271)

**Output**: `level` (0 to 4)

**Kick**: `kickPitch` (20 to 8000 Hz); `kickDecay` (0.005 to 10 s); `kickSweep` (0 to 48 st); `kickSweepTime` (0.001 to 0.5 s); `kickTone` (0 to 1); `kickSnap` (0 to 1); `kickDrive` (0 to 1); `kickLevel` (0 to 1.5); `kickPan` (-1 to 1)

**Snare**: `snarePitch` (20 to 8000 Hz); `snareDecay` (0.005 to 10 s); `snareSweep` (0 to 48 st); `snareSweepTime` (0.001 to 0.5 s); `snareTone` (0 to 1); `snareSnap` (0 to 1); `snareDrive` (0 to 1); `snareLevel` (0 to 1.5); `snarePan` (-1 to 1)

**Clap**: `clapPitch` (20 to 8000 Hz); `clapDecay` (0.005 to 10 s); `clapSweep` (0 to 48 st); `clapSweepTime` (0.001 to 0.5 s); `clapTone` (0 to 1); `clapSnap` (0 to 1); `clapDrive` (0 to 1); `clapLevel` (0 to 1.5); `clapPan` (-1 to 1)

**Rim**: `rimPitch` (20 to 8000 Hz); `rimDecay` (0.005 to 10 s); `rimSweep` (0 to 48 st); `rimSweepTime` (0.001 to 0.5 s); `rimTone` (0 to 1); `rimSnap` (0 to 1); `rimDrive` (0 to 1); `rimLevel` (0 to 1.5); `rimPan` (-1 to 1)

**Hat**: `hatPitch` (0.25 to 4); `hatDecay` (0.005 to 10 s); `hatSweep` (0 to 48 st); `hatSweepTime` (0.001 to 0.5 s); `hatTone` (0 to 1); `hatSnap` (0 to 1); `hatDrive` (0 to 1); `hatLevel` (0 to 1.5); `hatPan` (-1 to 1)

**Open**: `openPitch` (0.25 to 4); `openDecay` (0.005 to 10 s); `openSweep` (0 to 48 st); `openSweepTime` (0.001 to 0.5 s); `openTone` (0 to 1); `openSnap` (0 to 1); `openDrive` (0 to 1); `openLevel` (0 to 1.5); `openPan` (-1 to 1)

**TomLow**: `tomLowPitch` (20 to 8000 Hz); `tomLowDecay` (0.005 to 10 s); `tomLowSweep` (0 to 48 st); `tomLowSweepTime` (0.001 to 0.5 s); `tomLowTone` (0 to 1); `tomLowSnap` (0 to 1); `tomLowDrive` (0 to 1); `tomLowLevel` (0 to 1.5); `tomLowPan` (-1 to 1)

**TomMid**: `tomMidPitch` (20 to 8000 Hz); `tomMidDecay` (0.005 to 10 s); `tomMidSweep` (0 to 48 st); `tomMidSweepTime` (0.001 to 0.5 s); `tomMidTone` (0 to 1); `tomMidSnap` (0 to 1); `tomMidDrive` (0 to 1); `tomMidLevel` (0 to 1.5); `tomMidPan` (-1 to 1)

**TomHigh**: `tomHighPitch` (20 to 8000 Hz); `tomHighDecay` (0.005 to 10 s); `tomHighSweep` (0 to 48 st); `tomHighSweepTime` (0.001 to 0.5 s); `tomHighTone` (0 to 1); `tomHighSnap` (0 to 1); `tomHighDrive` (0 to 1); `tomHighLevel` (0 to 1.5); `tomHighPan` (-1 to 1)

**Cymbal**: `cymbalPitch` (0.25 to 4); `cymbalDecay` (0.005 to 10 s); `cymbalSweep` (0 to 48 st); `cymbalSweepTime` (0.001 to 0.5 s); `cymbalTone` (0 to 1); `cymbalSnap` (0 to 1); `cymbalDrive` (0 to 1); `cymbalLevel` (0 to 1.5); `cymbalPan` (-1 to 1)

**Crash**: `crashPitch` (0.25 to 4); `crashDecay` (0.005 to 10 s); `crashSweep` (0 to 48 st); `crashSweepTime` (0.001 to 0.5 s); `crashTone` (0 to 1); `crashSnap` (0 to 1); `crashDrive` (0 to 1); `crashLevel` (0 to 1.5); `crashPan` (-1 to 1)

**Ride**: `ridePitch` (0.25 to 4); `rideDecay` (0.005 to 10 s); `rideSweep` (0 to 48 st); `rideSweepTime` (0.001 to 0.5 s); `rideTone` (0 to 1); `rideSnap` (0 to 1); `rideDrive` (0 to 1); `rideLevel` (0 to 1.5); `ridePan` (-1 to 1)

**RideBell**: `rideBellPitch` (20 to 8000 Hz); `rideBellDecay` (0.005 to 10 s); `rideBellSweep` (0 to 48 st); `rideBellSweepTime` (0.001 to 0.5 s); `rideBellTone` (0 to 1); `rideBellSnap` (0 to 1); `rideBellDrive` (0 to 1); `rideBellLevel` (0 to 1.5); `rideBellPan` (-1 to 1)

**Cowbell**: `cowbellPitch` (20 to 8000 Hz); `cowbellDecay` (0.005 to 10 s); `cowbellSweep` (0 to 48 st); `cowbellSweepTime` (0.001 to 0.5 s); `cowbellTone` (0 to 1); `cowbellSnap` (0 to 1); `cowbellDrive` (0 to 1); `cowbellLevel` (0 to 1.5); `cowbellPan` (-1 to 1)

**Tambourine**: `tambourinePitch` (0.25 to 4); `tambourineDecay` (0.005 to 10 s); `tambourineSweep` (0 to 48 st); `tambourineSweepTime` (0.001 to 0.5 s); `tambourineTone` (0 to 1); `tambourineSnap` (0 to 1); `tambourineDrive` (0 to 1); `tambourineLevel` (0 to 1.5); `tambourinePan` (-1 to 1)

**Shaker**: `shakerPitch` (0.25 to 4); `shakerDecay` (0.005 to 10 s); `shakerSweep` (0 to 48 st); `shakerSweepTime` (0.001 to 0.5 s); `shakerTone` (0 to 1); `shakerSnap` (0 to 1); `shakerDrive` (0 to 1); `shakerLevel` (0 to 1.5); `shakerPan` (-1 to 1)

**BongoHigh**: `bongoHighPitch` (20 to 8000 Hz); `bongoHighDecay` (0.005 to 10 s); `bongoHighSweep` (0 to 48 st); `bongoHighSweepTime` (0.001 to 0.5 s); `bongoHighTone` (0 to 1); `bongoHighSnap` (0 to 1); `bongoHighDrive` (0 to 1); `bongoHighLevel` (0 to 1.5); `bongoHighPan` (-1 to 1)

**BongoLow**: `bongoLowPitch` (20 to 8000 Hz); `bongoLowDecay` (0.005 to 10 s); `bongoLowSweep` (0 to 48 st); `bongoLowSweepTime` (0.001 to 0.5 s); `bongoLowTone` (0 to 1); `bongoLowSnap` (0 to 1); `bongoLowDrive` (0 to 1); `bongoLowLevel` (0 to 1.5); `bongoLowPan` (-1 to 1)

**CongaMute**: `congaMutePitch` (20 to 8000 Hz); `congaMuteDecay` (0.005 to 10 s); `congaMuteSweep` (0 to 48 st); `congaMuteSweepTime` (0.001 to 0.5 s); `congaMuteTone` (0 to 1); `congaMuteSnap` (0 to 1); `congaMuteDrive` (0 to 1); `congaMuteLevel` (0 to 1.5); `congaMutePan` (-1 to 1)

**CongaHigh**: `congaHighPitch` (20 to 8000 Hz); `congaHighDecay` (0.005 to 10 s); `congaHighSweep` (0 to 48 st); `congaHighSweepTime` (0.001 to 0.5 s); `congaHighTone` (0 to 1); `congaHighSnap` (0 to 1); `congaHighDrive` (0 to 1); `congaHighLevel` (0 to 1.5); `congaHighPan` (-1 to 1)

**CongaLow**: `congaLowPitch` (20 to 8000 Hz); `congaLowDecay` (0.005 to 10 s); `congaLowSweep` (0 to 48 st); `congaLowSweepTime` (0.001 to 0.5 s); `congaLowTone` (0 to 1); `congaLowSnap` (0 to 1); `congaLowDrive` (0 to 1); `congaLowLevel` (0 to 1.5); `congaLowPan` (-1 to 1)

**TimbaleHigh**: `timbaleHighPitch` (20 to 8000 Hz); `timbaleHighDecay` (0.005 to 10 s); `timbaleHighSweep` (0 to 48 st); `timbaleHighSweepTime` (0.001 to 0.5 s); `timbaleHighTone` (0 to 1); `timbaleHighSnap` (0 to 1); `timbaleHighDrive` (0 to 1); `timbaleHighLevel` (0 to 1.5); `timbaleHighPan` (-1 to 1)

**TimbaleLow**: `timbaleLowPitch` (20 to 8000 Hz); `timbaleLowDecay` (0.005 to 10 s); `timbaleLowSweep` (0 to 48 st); `timbaleLowSweepTime` (0.001 to 0.5 s); `timbaleLowTone` (0 to 1); `timbaleLowSnap` (0 to 1); `timbaleLowDrive` (0 to 1); `timbaleLowLevel` (0 to 1.5); `timbaleLowPan` (-1 to 1)

**AgogoHigh**: `agogoHighPitch` (20 to 8000 Hz); `agogoHighDecay` (0.005 to 10 s); `agogoHighSweep` (0 to 48 st); `agogoHighSweepTime` (0.001 to 0.5 s); `agogoHighTone` (0 to 1); `agogoHighSnap` (0 to 1); `agogoHighDrive` (0 to 1); `agogoHighLevel` (0 to 1.5); `agogoHighPan` (-1 to 1)

**AgogoLow**: `agogoLowPitch` (20 to 8000 Hz); `agogoLowDecay` (0.005 to 10 s); `agogoLowSweep` (0 to 48 st); `agogoLowSweepTime` (0.001 to 0.5 s); `agogoLowTone` (0 to 1); `agogoLowSnap` (0 to 1); `agogoLowDrive` (0 to 1); `agogoLowLevel` (0 to 1.5); `agogoLowPan` (-1 to 1)

**Clave**: `clavePitch` (20 to 8000 Hz); `claveDecay` (0.005 to 10 s); `claveSweep` (0 to 48 st); `claveSweepTime` (0.001 to 0.5 s); `claveTone` (0 to 1); `claveSnap` (0 to 1); `claveDrive` (0 to 1); `claveLevel` (0 to 1.5); `clavePan` (-1 to 1)

**WoodblockHigh**: `woodblockHighPitch` (20 to 8000 Hz); `woodblockHighDecay` (0.005 to 10 s); `woodblockHighSweep` (0 to 48 st); `woodblockHighSweepTime` (0.001 to 0.5 s); `woodblockHighTone` (0 to 1); `woodblockHighSnap` (0 to 1); `woodblockHighDrive` (0 to 1); `woodblockHighLevel` (0 to 1.5); `woodblockHighPan` (-1 to 1)

**WoodblockLow**: `woodblockLowPitch` (20 to 8000 Hz); `woodblockLowDecay` (0.005 to 10 s); `woodblockLowSweep` (0 to 48 st); `woodblockLowSweepTime` (0.001 to 0.5 s); `woodblockLowTone` (0 to 1); `woodblockLowSnap` (0 to 1); `woodblockLowDrive` (0 to 1); `woodblockLowLevel` (0 to 1.5); `woodblockLowPan` (-1 to 1)

**Triangle**: `trianglePitch` (20 to 8000 Hz); `triangleDecay` (0.005 to 10 s); `triangleSweep` (0 to 48 st); `triangleSweepTime` (0.001 to 0.5 s); `triangleTone` (0 to 1); `triangleSnap` (0 to 1); `triangleDrive` (0 to 1); `triangleLevel` (0 to 1.5); `trianglePan` (-1 to 1)

**TriangleMute**: `triangleMutePitch` (20 to 8000 Hz); `triangleMuteDecay` (0.005 to 10 s); `triangleMuteSweep` (0 to 48 st); `triangleMuteSweepTime` (0.001 to 0.5 s); `triangleMuteTone` (0 to 1); `triangleMuteSnap` (0 to 1); `triangleMuteDrive` (0 to 1); `triangleMuteLevel` (0 to 1.5); `triangleMutePan` (-1 to 1)
