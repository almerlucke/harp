# Settings of the synths

<!-- made by `harp settings -md`: do not edit by hand (a test says when it is out of date) -->

Every setting of a synth is an optional parameter of its function, with the name psalm gives it: `subtractive(notes, preset: "pad", cutoff: 900, filterMode: "band pass", ampAttack: 400)`. A knob is a number, or a signal that moves it while the notes sound (`cutoff: lfo(0.1).range(300, 3000)`); a choice is a name (or its number); a switch is `true` or `false`. What is not given is the preset's own. The envelopes are a setting for each stage, in milliseconds (`ampAttack`, `ampDecay`, `ampSustain` from 0 to 1, `ampRelease`).


## subtractive (66)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations pitch, cutoff, resonance, pulse width, level 1, level 2, sub, noise, detune 2, fm, drive, vowel, level, lfo 1 rate, lfo 2 rate, unison detune, cutoff 2

**Oscillators**: `osc1Waveform` (sine, cosine, triangle, square, rectangle, saw, ramp, mod triangle, mod square, half sine, full sine, tri pulse, trapezoid, trapezoid var); `osc1Level` (0 to 1); `osc2Waveform` (sine, cosine, triangle, square, rectangle, saw, ramp, mod triangle, mod square, half sine, full sine, tri pulse, trapezoid, trapezoid var); `osc2Level` (0 to 1); `osc2Octave` (-3 to 3); `osc2Detune` (-24 to 24 st); `pulseWidth` (0.05 to 0.95); `subLevel` (0 to 1); `noiseLevel` (0 to 1); `noiseType` (white, pink); `ringModLevel` (0 to 1); `sync` (true or false); `fmAmount` (0 to 5); `drive` (0.5 to 8); `unison` (1 to 7); `unisonDetune` (0 to 100 ct); `drift` (0 to 50 ct)

**Filter**: `filterType` (state variable, ladder, formant); `filterMode` (low pass, band pass, high pass, notch, peak, all pass); `ladderMode` (low 24, low 12, band 24, band 12, high 24, high 12); `cutoff` (20 to 18000 Hz); `resonance` (0 to 1); `filterEnvAmount` (-6 to 6 oct); `keyTrack` (0 to 1); `filterDrive` (1 to 8); `filterOversample` (once, 2x, 4x); `vowel` (0 to 4); `formantShift` (0.5 to 2)

**Filter 2**: `filter2Mode` (off, low pass, band pass, high pass, notch); `filter2Cutoff` (20 to 18000 Hz); `filter2Resonance` (0 to 1); `filter2EnvAmount` (-6 to 6 oct); `filter2KeyTrack` (0 to 1); `filter2Parallel` (true or false)

**Pitch**: `glide` (0 to 2000 ms); `filterEnvToPitch` (-48 to 48 st); `level` (0 to 4)

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

## waveguide (45)

**Modulation matrix** (`mods`): sources filter env, amp env, lfo 1, lfo 2, velocity, key, random, wheel, ext 1, ext 2, ext 3, ext 4; destinations decay, damping, brightness, pick position, pressure, bow position, vibrato depth, body, pitch, level, cutoff, breath, embouchure, bow force, bow speed

**Pluck**: `damping` (0 to 1); `brightness` (0 to 1); `pickPosition` (0 to 0.5); `pluckWidth` (0 to 0.2); `plectrum` (0 to 1); `harmonic` (0 to 0.5); `fretNoise` (0 to 1); `fretBuzz` (0 to 1); `fretPosition` (0.05 to 0.98); `slide` (0 to 2 s)

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
