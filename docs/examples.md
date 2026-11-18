# Examples

In `lang/examples/`. Play one with `harp play lang/examples/NAME.harp` (copy it first if you want to edit it live).

| File | What it shows |
|---|---|
| `play.harp` | the whole language: a melody from generators, a filter, an echo, a synth pad, a reverb |
| `drone.harp`, `supersaw.harp` | a drone; psalm's supersaw synth |
| `generators_play.harp` | `play`: a euclidean rhythm and a mutating phrase play an instrument of your own |
| `generators_fx.harp` | a small piece in A minor: bass, pad, lead, drums, all driven by generators; effect parameters from generators |
| `shared_stream.harp` | one stream drives the notes of a `play` and the filter and pitch of a drone |
| `psalm_pieces.harp` | the synths and effects of psalm together: a vibraphone, a choir, a plucked bass, a chorus moved by an lfo, a tremolo that pans, tape, shimmer |
| `organ_dx_drums.harp` | a drum synth written with names and patterns, an FM bass and bells (dx), and a tonewheel organ with chords of six keys |
| `scores.harp` | a fugue exposition in three voices, each on its own synth, and a piece of phasing: music made whole as a score |
| `generators_more.harp` | chords from a progression with voice leading, a bass from a markov chain, an L-system lead on a cellular rhythm, a drone filtered by the Lorenz attractor |
| `grains_wavetable.harp` | a pad of grains cut from a chant, and a synced wavetable bass |
| `sampler.harp` | the sampler four ways, from psalm's assets: a drum kit of single hits, an 808 played as a bass (one sound on many keys), and a sung line cut at its pauses into pieces that a phrase picks from. `harp check` says which pieces are on which keys |
| `recordings.harp` | a celesta (SFZ), percussion cut from a loop, and a long recording from disk, with the files of psalm's assets |
| `live_demo.harp` | made for [live editing](live-editing.md): a slow sweep and a long reverb, so a restart is easy to hear. The comment at the top lists edits to try |

`lang/examples/supercollider/` has SuperCollider examples translated to harp, each with the original in its top comment, and comparable with [`harp compare`](supercollider.md): `1_vibrato` to `7_dust_pings` (small ones), `8_random_cascade` (random modulation, 16 stages), `9_gabber_kick`, `10_hoover` and `11_gabber_rave` (a pattern of two instruments).
