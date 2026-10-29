package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/firstrow/wig"
	"github.com/gdamore/tcell/v2"
)

var cmdHistory []string

type uiCommandLine struct {
	e          *wig.Editor
	keymap     *wig.KeyHandler
	chBuf      []rune
	historyIdx int
	candidates []string
	candIdx    int
}

func (u *uiCommandLine) Plane() wig.RenderPlane {
	return wig.PlaneEditor
}

func CmdLineInit(ctx wig.Context) {
	cmdLineInit(ctx, "")
}

// CmdFilterLineInit opens the command line prefilled for filtering the current
// line (!!), or {count} lines, through a shell command.
func CmdFilterLineInit(ctx wig.Context) {
	prefix := ".!"
	if ctx.Count > 1 {
		prefix = fmt.Sprintf(".,.+%d!", ctx.Count-1)
	}
	cmdLineInit(ctx, prefix)
}

func cmdLineInit(ctx wig.Context, prefix string) {
	u := &uiCommandLine{
		e:          ctx.Editor,
		chBuf:      append(make([]rune, 0, 32), []rune(prefix)...),
		historyIdx: len(cmdHistory), // Start at the end of history
		candidates: []string{},
		candIdx:    -1,
	}
	u.keymap = wig.NewKeyHandler(wig.ModeKeyMap{
		wig.MODE_INSERT: wig.KeyMap{
			"Esc": func(ctx wig.Context) {
				ctx.Editor.PopUi()
			},
			"Enter": func(ctx wig.Context) {
				cmd := string(u.chBuf)
				if strings.TrimSpace(cmd) != "" {
					cmdHistory = append(cmdHistory, cmd)
				}
				u.execute(cmd)
			},
			"Tab": func(ctx wig.Context) {
				u.autocomplete()
			},
			"Up": func(ctx wig.Context) {
				if u.historyIdx > 0 {
					u.historyIdx--
					u.chBuf = []rune(cmdHistory[u.historyIdx])
					u.candidates = []string{}
					u.candIdx = -1
				}
			},
			"Down": func(ctx wig.Context) {
				if u.historyIdx < len(cmdHistory)-1 {
					u.historyIdx++
					u.chBuf = []rune(cmdHistory[u.historyIdx])
					u.candidates = []string{}
					u.candIdx = -1
				} else {
					u.historyIdx = len(cmdHistory)
					u.chBuf = []rune{}
					u.candidates = []string{}
					u.candIdx = -1
				}
			},
		},
	})
	u.keymap.Fallback(u.insertCh)
	ctx.Editor.PushUi(u)
}

func (u *uiCommandLine) insertCh(ctx wig.Context, ev *tcell.EventKey) {
	if ev.Modifiers()&tcell.ModCtrl != 0 {
		return
	}
	if ev.Modifiers()&tcell.ModAlt != 0 {
		return
	}
	if ev.Modifiers()&tcell.ModMeta != 0 {
		return
	}

	// Removed tcell.KeyEnter handling here!
	// If it was handled here, it bypassed the keymap's "Enter" handler,
	// meaning the command was never executed and history was never saved.

	if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
		if len(u.chBuf) > 0 {
			u.chBuf = u.chBuf[:len(u.chBuf)-1]
		}
		u.candidates = []string{}
		u.candIdx = -1
		return
	}

	if ev.Key() != tcell.KeyRune {
		return
	}
	u.chBuf = append(u.chBuf, ev.Rune())
	u.candidates = []string{}
	u.candIdx = -1
}

func (u *uiCommandLine) autocomplete() {
	input := string(u.chBuf)
	parts := strings.SplitN(input, " ", 2)

	// File path completion for :e, :w and :r
	if len(parts) == 2 && (parts[0] == "e" || parts[0] == "e!" || parts[0] == "edit" ||
		parts[0] == "w" || parts[0] == "w!" || parts[0] == "write" ||
		parts[0] == "r" || parts[0] == "read") {
		cmdPart := parts[0]
		prefix := parts[1]

		if len(u.candidates) > 0 {
			u.candIdx++
			if u.candIdx >= len(u.candidates) {
				u.candIdx = 0
			}
			u.chBuf = []rune(fmt.Sprintf("%s %s", cmdPart, u.candidates[u.candIdx]))
			return
		}

		dir := "."
		filePrefix := prefix
		if strings.Contains(prefix, "/") {
			lastSlash := strings.LastIndex(prefix, "/")
			dir = prefix[:lastSlash]
			if dir == "" {
				dir = "/"
			}
			filePrefix = prefix[lastSlash+1:]
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}

		var matches []string
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, filePrefix) {
				if entry.IsDir() {
					name += "/"
				}
				if dir != "." {
					if dir == "/" {
						name = "/" + name
					} else {
						name = dir + "/" + name
					}
				}
				matches = append(matches, name)
			}
		}

		if len(matches) == 0 {
			return
		}

		sort.Strings(matches)

		common := matches[0]
		for _, m := range matches[1:] {
			i := 0
			for i < len(common) && i < len(m) && common[i] == m[i] {
				i++
			}
			common = common[:i]
		}

		u.chBuf = []rune(fmt.Sprintf("%s %s", cmdPart, common))
		u.candidates = matches
		u.candIdx = -1 // Next Tab will cycle to 0
		return
	}

	// Command name completion
	prefix := input
	if len(u.candidates) > 0 {
		u.candIdx++
		if u.candIdx >= len(u.candidates) {
			u.candIdx = 0
		}
		u.chBuf = []rune(u.candidates[u.candIdx])
		return
	}

	var matches []string
	for name := range wig.AllCommands {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}

	if len(matches) == 0 {
		return
	}

	sort.Strings(matches)

	if len(matches) == 1 {
		u.chBuf = []rune(matches[0])
		u.candidates = matches
		u.candIdx = 0
		return
	}

	common := matches[0]
	for _, m := range matches[1:] {
		i := 0
		for i < len(common) && i < len(m) && common[i] == m[i] {
			i++
		}
		common = common[:i]
	}
	u.chBuf = []rune(common)
	u.candidates = matches
	u.candIdx = -1
}

func (u *uiCommandLine) execute(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		u.e.PopUi()
		return
	}

	// :N, :$, :+3 ... go to a line
	if isLineRange(cmd) {
		u.e.PopUi()
		u.gotoLine(cmd)
		return
	}

	// :!cmd and :{range}!cmd
	if i := strings.Index(cmd, "!"); i >= 0 && isLineRange(cmd[:i]) {
		u.e.PopUi()
		u.runShell(strings.TrimSpace(cmd[:i]), strings.TrimSpace(cmd[i+1:]))
		return
	}

	parts := strings.SplitN(cmd, " ", 2)
	// :e! reloads the buffer from disk. :e! file is the same as :e file.
	if len(parts) == 1 && parts[0] == "e!" {
		u.e.PopUi()
		if err := wig.EditForce(u.e.NewContext()); err != nil {
			u.e.EchoMessage(err.Error())
		}
		return
	}

	// :w file writes a copy. A bare :w is a regular command, see below.
	if len(parts) == 2 && (parts[0] == "w" || parts[0] == "w!" || parts[0] == "write") {
		u.e.PopUi()
		filePath := strings.TrimSpace(parts[1])
		if err := wig.WriteTo(u.e.NewContext(), filePath); err != nil {
			u.e.EchoMessage(fmt.Sprintf("Can't write %s: %v", filePath, err))
		}
		return
	}

	if len(parts) > 0 && (parts[0] == "e" || parts[0] == "e!" || parts[0] == "edit") {
		u.e.PopUi()
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			u.e.EchoMessage("No file name")
			return
		}
		filePath := strings.TrimSpace(parts[1])
		buf, err := u.e.OpenOrCreateFile(filePath)
		if err != nil {
			u.e.EchoMessage(fmt.Sprintf("Error opening %s: %v", filePath, err))
			return
		}
		ctx := u.e.NewContext()
		ctx.Buf = buf
		u.e.ActiveWindow().VisitBuffer(ctx)
		return
	}

	if len(parts) > 0 && (parts[0] == "r" || parts[0] == "read") {
		u.e.PopUi()
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			u.e.EchoMessage("No file name")
			return
		}
		filePath := strings.TrimSpace(parts[1])
		if err := wig.ReadFileBelowCursor(u.e.NewContext(), filePath); err != nil {
			u.e.EchoMessage(fmt.Sprintf("Can't read %s: %v", filePath, err))
		}
		return
	}

	// Pop the command line UI *before* executing the command.
	// This allows commands (like :q) to push their own UI (like the exit prompt)
	// without it being immediately popped by this deferred cleanup.
	u.e.PopUi()

	if def, ok := wig.AllCommands[cmd]; ok {
		ctx := u.e.NewContext()
		if fn, ok := def.Fn.(func(wig.Context)); ok {
			fn(ctx)
		} else {
			u.e.EchoMessage(fmt.Sprintf("Command %s is not executable", cmd))
		}
	} else {
		u.e.EchoMessage(fmt.Sprintf("Unknown command: %s", cmd))
	}
}

func (u *uiCommandLine) Keymap() *wig.KeyHandler {
	return u.keymap
}

func (u *uiCommandLine) Render(view wig.View) {
	vw, vh := view.Size()
	prompt := fmt.Sprintf(":%s", string(u.chBuf))

	bgStyle := wig.Color("default")
	view.SetContent(0, vh-1, strings.Repeat(" ", vw), bgStyle)
	view.SetContent(0, vh-1, prompt, bgStyle)

	cursorStyle := bgStyle.Reverse(true)
	if len(prompt) < vw {
		view.SetContent(len(prompt), vh-1, " ", cursorStyle)
	}

	// Show candidates visually on the line above the prompt (like Vim)
	if len(u.candidates) > 1 {
		candStr := strings.Join(u.candidates, "  ")
		if len(candStr) > vw {
			candStr = candStr[:vw]
		}
		if vh-2 >= 0 {
			view.SetContent(0, vh-2, candStr, bgStyle)
		}
	}
}

func (u *uiCommandLine) Mode() wig.Mode {
	return wig.MODE_INSERT
}

// gotoLine moves the cursor to the last line of a range such as 5, $ or +3.
func (u *uiCommandLine) gotoLine(spec string) {
	ctx := u.e.NewContext()
	last := ctx.Buf.Lines.Len - 1

	var line int
	if n, err := strconv.Atoi(spec); err == nil {
		// like vi, a number past the end goes to the last line and 0 to the first
		line = min(max(n-1, 0), last)
	} else {
		_, to, err := wig.ParseLineRange(spec, wig.ContextCursorGet(ctx).Line, last)
		if err != nil {
			u.e.EchoMessage(err.Error())
			return
		}
		line = to
	}

	ctx.Count = uint32(line + 1)
	wig.CmdGotoLineEndOfFile(ctx)
}

// isLineRange reports whether s is empty or made only of range characters.
func isLineRange(s string) bool {
	return strings.Trim(s, " .$%,+-0123456789") == ""
}

// runShell handles :!cmd (show the output) and :{range}!cmd (filter lines).
func (u *uiCommandLine) runShell(rng, cmdline string) {
	if cmdline == "" {
		u.e.EchoMessage("No shell command")
		return
	}
	ctx := u.e.NewContext()

	if rng != "" {
		if ctx.Buf == nil {
			u.e.EchoMessage("No buffer to filter")
			return
		}
		cur := wig.ContextCursorGet(ctx)
		from, to, err := wig.ParseLineRange(rng, cur.Line, ctx.Buf.Lines.Len-1)
		if err == nil {
			err = wig.FilterLines(ctx, from, to, cmdline)
		}
		if err != nil {
			u.e.EchoMessage("Filter failed: " + err.Error())
		}
		return
	}

	out, err := wig.RunShell(cmdline, "")
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		out += fmt.Sprintf("\n[exit status %d]", exitErr.ExitCode())
	} else if err != nil {
		u.e.EchoMessage(err.Error())
		return
	}

	out = strings.TrimRight(out, "\n")
	switch {
	case out == "":
		u.e.EchoMessage("[done]")
	case !strings.Contains(out, "\n"):
		u.e.EchoMessage(out)
	default:
		buf := wig.NewBuffer()
		buf.ResetLines()
		buf.Append(out)
		fw := NewFloatingWindow(u.e, buf, FloatingWindowOpts{
			Width:  u.e.Config.FloatingWindowWidth,
			Height: u.e.Config.FloatingWindowHeight,
			Title:  "!" + cmdline,
		})
		u.e.PushUi(fw)
		u.e.Redraw()
	}
}
