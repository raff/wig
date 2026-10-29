# vi compatibility work

This document describes the vi/vim key bindings and ex commands added on the
`vi-bindings` branch, where wig still differs from vi, and what is left to do.
The full list of default bindings is in `config/config.go` (`DefaultKeyMap`).

## What was added

### Motions

| Key | Action |
|-----|--------|
| `ctrl+f` / `ctrl+b` | page down / up (normal and visual mode) |
| `ctrl+d` / `ctrl+u` | half page down / up (normal and visual mode) |
| `H` / `M` / `L` | top / middle / bottom visible line (`H` and `L` take a count) |
| `{` / `}` | previous / next paragraph (empty-line boundaries) |
| `+` / `-` / `Enter` | first non-blank of next / previous / next line |
| `e` | end of word |
| `W` / `B` / `E` | next / previous / end of WORD (whitespace-delimited) |
| `Backspace` | one character back, wrapping to the previous line (normal mode) |
| `{count}G` | go to line N (`G` alone still goes to the last line) |
| `0` | column zero, even with counts enabled (it only extends a count after another digit) |
| `z.` / `zz` | center the cursor line |
| `ctrl+]` / `ctrl+t` | go to definition / jump back |
| `ctrl+l` | redraw the screen |

`W`, `B` and `E` skip over empty lines. The visual-mode versions of
`ctrl+d`, `ctrl+u`, `H`, `M`, `L`, `{`, `}`, `e`, `W`, `B` and `E` extend the
selection.

### Edits

| Key | Action |
|-----|--------|
| `Delete` | delete the character under the cursor (same as `x`) |
| `C` / `D` / `S` / `Y` | change to end of line / delete to end of line / change line / yank line |
| `I` | insert at the first non-blank character |
| `s` | substitute character |
| `~` | toggle case of the character under the cursor and move right |
| `p` / `P` | paste linewise text below / above the cursor line (see below) |

`p` and `P` with a linewise register now paste below and above the current
line (including on the last line), repeat the text for a count (`3p`), make the
paste a single undo step, and leave the cursor on the first pasted line.

### Operators with marks, G and gg

`y`, `d` and `c` accept these targets:

| Target | Meaning |
|--------|---------|
| `'a` | linewise, from the cursor line to mark `a` |
| `` `a `` | characterwise, from the cursor to mark `a`, exclusive like vi |
| `G` | linewise, to the end of the file, or to line N with a count (`d3G`) |
| `gg` | linewise, to the start of the file, or to line N with a count (`y3gg`) |

`c` with a linewise target leaves one empty line behind and enters insert
mode, like vi. Counts typed after an operator now work (`d2d`, `d3G`) as long
as the pending operator's keymap doesn't bind that digit itself (so
`Space 1` still switches workspace).

### Ex commands

| Command | Action |
|---------|--------|
| `:e file` | edit a file; a file that doesn't exist opens as an empty new buffer with a `[New File]` message and is created on save |
| `:r file` | insert a file below the cursor line; an unreadable file reports an error |
| `:r !cmd` | insert a command's output below the cursor line |
| `:!cmd` | run a command in `$SHELL` and show the output (status line, or a floating window for multi-line output; the window closes with `Esc`) |
| `:{range}!cmd` | filter the lines through a command and replace them with its output |
| `!!` | open the command line prefilled with `.!` (`3!!` prefills `.,.+2!`) |

Ranges understand `.`, `%`, `$`, `N`, `N,M` and offsets such as `.,+2` or
`-1,.`. Commands run with empty stdin and a 30 second timeout.
`wig <file>` also opens a missing file as a new buffer.

## Differences from vi

These are places where wig behaves differently on purpose or because of how it
is built.

**Rebound keys**
- The MRU buffer picker moved from `ctrl+b` to `alt+b` so `ctrl+b` can page up.
- `;` opens the buffer picker, so it does not repeat the last `f`/`t`/`F`/`T`.
- `'` and `` ` `` open a marks popup (press the mark's letter to jump) instead
  of jumping to a mark directly. `y'a`, `d'a`, `` c`a `` and the like do take a
  mark letter directly.
- `Backspace` moves back one character, wrapping to the previous line, as in
  vi. On a Mac keyboard the key labelled "delete" sends a backspace code, so
  that is the key that moves back; the forward-delete key (`fn+delete`) is
  `Delete`, which deletes the character under the cursor like `x`. Both are
  normal-mode bindings only.
- `.` replays the last recorded key sequence (the macro system), not a
  structured "last change".

**Counts**
- Counts typed before and after an operator are concatenated, not multiplied:
  `2d3G` is read as count 23.
- Not every command honors a count. For example `x`, `r` and `~` ignore it.

**Motions and editing**
- `W` and `B` skip empty lines; vi stops on them.
- `H` and `L` go to the first and last visible line without applying a scroll
  offset.
- `ctrl+f` and `ctrl+b` scroll by the window height minus a few lines of
  context, and the cursor moves with the view.
- `ci` is the only text-object command. It is marked TODO in the code and
  covers `w` and a few bracket and quote types. `d` and `y` have no text
  objects.
- Visual-block mode has only `I`, `d`, `x`, `y`.

**Ex commands and shell**
- Only a fixed set of ex commands exist (`:w`, `:q`, `:q!`, `:wq`, `:bd`,
  `:bn`, `:bp`, `:bl`, `:vs`, `:sp`, `:only`, `:close`, `:e`, `:r`, `:!` and a
  few wig-specific ones). Anything else prints `Unknown command`.
- A filter (`:{range}!cmd`) that fails (non-zero exit or timeout) leaves the
  buffer unchanged and shows the error on the status line. vi would replace the
  lines with the error text.
- A filter's stdout and stderr are merged into the replacement text, as in vi.
- `:!` has no `%` or `#` filename substitution and no `:!!` repeat.

**Other**
- Commands added in `vi_cmds.go` are bound in the default keymap but not
  registered in `wig.AllCommands`, so they can't be bound from `config.toml`
  or run from the command palette. Only `CmdScrollUpPage`, `CmdScrollDownPage`
  and `CmdRedrawScreen` were registered.
- Undo does not move or clamp the cursor, so undoing an insert that shortened
  the buffer can leave the cursor past the last line.

## TODO

Clear omissions from standard vi, roughly in order of how often they come up.

**Motions**
- `;` and `,` to repeat the last `f`/`t`/`F`/`T` (needs a rebind of `;`)
- `?` backward search and `#` word search backward
- `(` / `)` sentence motions, `_`, `|`, `ge`, `gE`, `g_`
- `ctrl+o` / `ctrl+i` jump list (wig uses `[[` and `]]`)
- direct `'a` and `` `a `` jumps, and the automatic marks (`''`, `` `. ``, `'<`, `'>`)
- `zt` and `zb`
- `ctrl+g`, `ga`, `gv`

**Operators and text objects**
- motions for `d`, `y` and `c`: `dw`-style coverage is partial. Missing include
  `d0`, `d^`, `dj`, `dk`, `db`, `de`, `d}`, `d{`, `yw`, `y0`, `yj`
- text objects for `d` and `y` (`iw`, `aw`, `i(`, `a"` and so on) and a
  complete set for `c`
- `>` and `<` with motions (only `>>` and `<<` exist), `=`
- `gu`, `gU`, `g~` case operators
- `!` followed by a motion (`!ip`), and `!` in visual mode (`:'<,'>!`)
- multiplying counts around operators (`2d3G`)

**Editing**
- `R` replace mode, `&`, `U`, `gJ`
- `ctrl+a` / `ctrl+x` number increment and decrement
- registers (`"ayy`, `"ap`, `"+`) and `:registers`
- `ZZ` and `ZQ`
- insert-mode keys such as `ctrl+w`, `ctrl+u`, `ctrl+d`, `ctrl+t`, `ctrl+v`, `ctrl+r`

**Visual mode**
- `o`, `O`, `J`, `~`, `u`, `U`, `r`, `aw`/`iw`, `gv`
- visual-block `A`, `c`, `r`, `o`, `$`

**Ex commands**
- `:N` (go to a line), `:w file`, `:x`, `:e!`
- `:s` substitute, `:g` and `:v`, `:d`, `:y`, `:m`, `:t`, `:normal`
- `:set` and `:noh`
- range prefixes for `:r` (`:5r file`) and `:w`
- `%` / `#` expansion and `:!!` for shell commands

**Windows**
- `ctrl+w s`, `ctrl+w h/j/k/l`, `ctrl+w =` and resizing

## Known issues

- Visual-line replace paste (`V` + motion + `p`) leaves an extra blank line and
  undoes only partway. This predates the paste rewrite.
- Undo can leave the cursor past the last line (see above).
- `TestKeyTimes` (root package) and `TestEditing` (`config` package) fail on
  `master` and on this branch.

## Where the code is

| Area | File |
|------|------|
| key bindings | `config/config.go` |
| count handling | `keys.go` |
| new motions and edits, mark and `G`/`gg` operators, `:r` | `vi_cmds.go` |
| `p` / `P` | `yank.go` |
| shell commands, line ranges, filters | `shell.go` |
| `:` command line, `!!`, `:!` | `ui/commandline.go` |
| opening new files | `editor.go` (`OpenOrCreateFile`), `cmd/main.go` |
