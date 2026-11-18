# The generator library

The generators of genny, by what they are for. Each one is a function that makes a [generator](generators.md); the details of each are in the [reference](reference.md). All of them are seeded (`seed = 7` at the top) and work with live editing: a save that does not change a generator leaves it where it was.

Some give a **list** (made when the patch is built); a list works wherever a generator is wanted, and goes round.

## Rhythms

| | |
|---|---|
| `pattern("x..x ..x. x...")` | a rhythm written as text, with rolls, triplets, chances and conditions: [Patterns](patterns.md) |
| `euclid`, `every`, `divide(n, offset)` | even spreads and clocks |
| `talea([3 2 3])` | a rhythm from lengths of notes (isorhythm: with a pitch list of another length it comes back together late) |
| `ratchets([1 2 1 4], 4)` | rolls: so many hits in each beat |
| `meter([4 4], level)` | the hits of a meter, thinned by a level that can move |
| `sieve(16, [3 5], [0 2])` | rhythm from the sieves of Xenakis |
| `cellular(90, 16)`, `life(16, 8)` | rhythms that grow from an automaton |
| `scatter(chance)` | random hits with a chance that can rise and fall |
| `maybe`, `invert`, `both`, `either` | change or join rhythms |
| `divisible(numbers, n)` | a rhythm from numbers: `divisible(fibonacci(12), 2)` |

## Combining generators

They take generators of any kind (rhythms, degrees, pitches, chords, notes) and give one of the same kind.

| | |
|---|---|
| `alternate([a, b])` | one value from each in turn |
| `anyOf([a, b])`, `chance([a, b], [3 1])` | one at random, with weights |
| `sections([a, b], [64 32])` | one after the other, for so many steps |
| `stutter(g, 2)` | each value more than once |
| `shotgun`, `noRepeat`, `distinct` | drop values, forbid repeats |
| `stack([a, b])`, `flatten(chords)` | voices into chords, chords into notes |
| `deal(list)`, `pick(list)`, `order(list, "converge")`, `uniform(lo, hi)` | a shuffled deck, any one, the orders of a list, random numbers |

## Numbers that move

`wander` (random walk), `drift` (smooth noise), `wave`, `sweep`, `orbit` (Lissajous), `chaos` (logistic, henon, lorenz, ...), `tendency` (a band that moves), and the chance curves `gauss`, `expo`, `beta`, `cauchy`, `weibull`, `pink`. `remap` moves any of them to the range you want:

```harp
cutoff = remap(chaos("lorenz"), -20, 20, 300, 2600)
out = saw(110).lowPass(cutoff: seq(cutoff), res: 0.3).gain(0.1).dup()
```

## Rules and chance

| | |
|---|---|
| `lsystem("A", ["A=AB", "B=A"], 6, ["A=0", "B=2"])` | a rewriting system, as a list of numbers |
| `grammar(["S=A B", "A=60 62 \| 67"])` | a grammar that makes a new phrase every round |
| `markov(values, weights)` | a chain: what is likely to come after what |
| `learn(example, 2)` | new music in the style of an example list |
| `stroll(values, shape: [4 4])` | a random walk through a list or a grid |
| `evolve(length: 8, smooth: 1, arch: 0.5)` | a phrase that gets better as it plays |

## Scales, chords, lines

| | |
|---|---|
| `scale([0 2 3 7 8])`, `equal(24)`, `just([...])` | scales and tunings of your own; more named scales (`phrygian`, `blues`, `wholetone`, `mode2`, ...) |
| `stepwise(60, major)`, `snap(g, root, scale)` | a walk over a scale, notes moved onto a scale |
| `chord(a3, "minor7")`, `triads(a2, minor, degrees)` | chords, and the chords on scale degrees |
| `voiceLead(chords)` | smooth voice leading |
| `arp(notes, mode: "converge")` | an arpeggio |
| `progression("pop")` | chord changes by chance (pop, classical, jazz, blues) |
| `toneRow()`, `rowForm(row, "retro")` | twelve tone rows |
| `counterpoint(cantus)` | a line against another, by the rules of strict counterpoint |

## Number sequences and durations

`fibonacci`, `thueMorse`, `primes`, `collatz`, `jump` (a way through a list), and for `dur:` of [play](play.md): `split(4, 0.25, 0.5, 2)` (a length cut into related parts) and `metric(4, 3)`.

An example that uses a good part of them: `examples/generators_more.harp`.

## Not wrapped

Genny also has what makes whole scores of timed events instead of a value for each step: those are [scores](scores.md) (`line`, `melody`, `phase`, `fugue`, `isorhythm`, `harmony`, `midi` read). Not wrapped are `microtonal` output, `template`, and the waveshapers of `float/shape`, which are for signals. They do not fit a clock that gives one value at a time. Its `groove` (swing, humanize, accent) is in [play](play.md#swing-humanize-and-accent), as `swing`, `humanize` and `accent: accents(...)`.
