# The language

A patch is a list of statements. Each is a name and an expression:

```harp
cutoff = lfo(0.3).range(300, 4000)
out = saw(110).lowPass(cutoff: cutoff).gain(0.2)
```

White space does not matter (a line that starts with a dot goes on with the line before). `//` starts a comment.

## Names with a meaning

| Name | Meaning |
|---|---|
| `out` | what is heard (required, once) |
| `bpm` | tempo, default 120 |
| `steps` | steps in a beat, default 4: the clock that generators run on |
| `seed` | seeds the random numbers: the same patch sounds the same |

`true`, `false`, `major`, `minor`, `dorian`, `pentatonic` and `pi` are always there. A name is defined once per scope.

## How names are written

Names are **camelCase**: a name of two words has the second one with a capital, `voiceLead`, `expRange`, `barSteps`, `filterEnvAmount`. That goes for functions, for parameters and for the names in your own patches. A function of one word is all lowercase (`euclid`, `reverb`), and so are the names of presets and kits, which are values and are written as psalm writes them: `preset: "church_bell"`, `kit: "909"`.

Parameters of the effects and the synths are psalm's own names for the settings (`autoMakeup`, `highCut`), so what you read in psalm is what you write here.

The names that were written in one word before (`lowpass`, `bysection`, `drumsynth`, `barsteps`, `warpmode`...) still work, and `harp check` says what each is now called.

## Values

| Written | Is |
|---|---|
| `440`, `0.5`, `1e3`, `-8` | a number |
| `c4`, `a3`, `f#2` | a note, as its MIDI number (`c4` is 60, `a3` is 57) |
| `"pad"` | a string |
| `[7 9 11]` | a list, items separated by spaces (plain items) |
| `[a * 2, b]` | a list with a comma: the items can be expressions |

Operators: `+ - * /`, with the usual precedence. A signal can be divided by a number, not by a signal.

## Calls and chains

`f(x, a, b)` and `x.f(a, b)` are the same: the thing before the dot is the first argument. So a chain reads in the order the sound flows:

```
saw(110).lowPass(cutoff: 800, res: 0.3).gain(0.5)
```

Arguments can be named (`res: 0.3`); positional ones come first. A name after a dot without parentheses is a field: `voice.pitch`, `voice.gate`.

## Your own functions

```
fn pluck(gate, note) {
  tone = saw(hz(note))
  tone.lowPass(cutoff: 2000) * env(gate, [0 1 0], [0.003 0.3], -4)
}
```

The body is statements, then the value. A function sees its parameters and the names of the patch that are defined by the time it is called, not the variables of the place it is called from. It can call functions defined before it (and itself, up to 200 deep).

## A patch in several files

`use "file.harp"` puts the statements of another file at that place, as if they were written there. The names and the functions of the file are the patch's from there on, so a long piece can be cut in parts: the form and the chords in one file, the drums in another, the instruments in a third, and the patch itself with `out` and the mix.

```text
use "parts/song.harp"
use "parts/drums.harp"
use "parts/instruments.harp"

out = (drums + instruments).reverb(mix: 0.15).gain(0.6)
```

- the name of a file is from the folder of the file that has the `use` (`parts/drums.harp` can say `use "song.harp"` for `parts/song.harp`);
- a file that is used twice (the drums and the instruments both use `song.harp`) is read once;
- **only the patch itself has `out`**; a file that is used with an `out` is refused;
- a name or a function that is defined twice, in any of the files, is a mistake that says where both are (`prog is already defined, at song.harp:4:1`);
- a file that uses itself, through others, is a mistake that shows the chain;
- a mistake in a used file is shown with that file's name and line.

`use`, like `fn`, `if` and `else`, is a word of the language: it can't be the name of something (`use is a word of the language, it can't be used as a name`).

`bpm`, `steps` and `seed` can be in a used file too; the last one that is done is the one that counts. With `harp play`, a save in any of the files takes over like a save in the patch ([live editing](live-editing.md)). `examples/in_parts.harp` is a piece in four files.

## `if`

```
filter = if coin(0.5) { lowPass(x, 800) } else { highPass(x, 800) }
```

`if` is an expression. Only the branch that is taken is worked out, so nothing in the other is built or drawn. The condition must be true or false (`coin(p)` is true with chance p, decided when the patch is built).

## Errors

Mistakes are reported with the line and column, the line itself, and, where a name is nearly right, a suggestion:

```
first.harp:3:9: there is no function lowpas (did you mean lowPass?)
```

The checker also keeps signals and generators apart: see [Types and channels](types.md).
