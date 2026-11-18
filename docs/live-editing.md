# Live editing

`harp play patch.harp` watches the file. When you save it, the new patch takes over **at the next step**, and what is the same keeps its state: oscillators go on from their phase, filters from their memory, a reverb keeps its tail, a generator its place in its phrase.

```
harp play patch.harp            # watches by default
harp play patch.harp -watch=false
```

After each save it prints what happened:

```
patch.harp: changed, it takes over at the next step (55 nodes go on with their state, 2 start again: fdnreverb, fdnreverb[1])
```

A patch with a mistake is reported with its line and ignored; the old one keeps playing:

```
patch.harp:12:9: there is no function lowpas (did you mean lowPass?)
(not changed: the old patch plays on)
```

## What a save changes, and what restarts

| You change | What happens |
|---|---|
| a number: cutoff, gain, mix, frequency | it changes, nothing restarts |
| `bpm`, `steps` | the clock changes at the next step |
| the lists or generators in the `fields` of a `play` | those fields get the new generator, the notes already queued get the new numbers; the rest of the pattern goes on |
| `dur`, `rhythm`, `legato`, `sustain` of a `play` | the notes follow the new rhythm from the next step; the phrases of the fields go on, no note skipped or played twice |
| a layer added to or removed from a sum | the sum changes; the reverb and echoes after it keep their tails; the new layer's nodes start |
| the notes of a synth (`chords`, `melody`, `part`, their rhythm, a `register`) | the synth keeps its voices and what sounds goes on; the next notes are the new ones, and the new text goes on at the step the music is at (a `part` of 16 steps that is changed in its seventh step plays its eighth next), not from its start |
| a setting of a synth (`cutoff: 900`, `ampAttack`, `mods`) | it is changed in the notes that sound, and taking it out gives back the preset's own |
| a setting that is a number in an effect (`chorus(rate: 0.7)`) | that effect starts again, and its tail is cut. Make the setting a signal (`rate: lfo(0.1).range(0.5, 1)`) to move it without a restart |
| a setting inside a node: reverb `decay`, echo `max`, a synth preset | that node starts again, and its tail is cut. The nodes around it keep theirs |
| one oscillator for another (`saw` → `pulse`) | the new oscillator starts; the filter after it is the same filter, with its memory |
| `voices` or the number of `fields` of a `play` | that pattern starts again |
| the number of channels of `out` | refused (`the patch has 2 channels, the one that plays has 1`); the old patch plays on |

A save that changes nothing (a comment, spaces) gives bit-for-bit the same sound as not saving.

## Good to know

- **Random numbers.** Put `seed = N` in the patch. Edits then draw the same build-time numbers (`rand`, `choose`, ...) and a save does not re-roll them. Without a `seed`, harp keeps the one it picked at the start. Running generators (`mutate`) are not reseeded by a save.
- **Delay.** The change is heard at the next step, so at slow tempos up to one step later (0.14 s at 104 bpm).
- **Fast saves.** Changes that follow each other quickly are each applied; the later one takes over from the earlier even if that has not been heard yet.
- **Optimizer.** While watching, harp does not optimize the graph (an optimized graph changes shape after a small edit and the old and new could not be matched). This costs a little CPU.

## How it matches (for the curious)

The new graph is matched with the running one node by node: same kind of node, same inputs, constant values ignored. A matched node keeps its state if what it *is* did not change (its own settings, and for generators their fingerprint, which is the expression that made them). A kept node goes on reading the buffers it always read; if its input is another node now, the new input is copied there. See [Architecture](architecture.md).
