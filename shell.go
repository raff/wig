// Shell commands: :!cmd, :{range}!filter and :r !cmd
package wig

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const shellTimeout = 30 * time.Second

// RunShell runs cmdline with the user's shell, feeding it stdin, and returns
// the combined stdout and stderr. It gives up after shellTimeout so a command
// waiting for input can't hang the editor.
func RunShell(cmdline, stdin string) (string, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), shellTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell, "-c", cmdline)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("timed out after %s", shellTimeout)
	}
	return string(out), err
}

// ParseLineRange parses an ex line range such as ".", "%", "$", "5", "2,4",
// ".,+2" or "-1,." into 0-based inclusive line numbers.
func ParseLineRange(spec string, cur, last int) (from, to int, err error) {
	spec = strings.TrimSpace(spec)
	if spec == "%" {
		return 0, last, nil
	}

	addrs := strings.Split(spec, ",")
	if len(addrs) > 2 {
		return 0, 0, errors.New("Invalid range")
	}
	nums := make([]int, len(addrs))
	for i, a := range addrs {
		if nums[i], err = parseLineAddr(a, cur, last); err != nil {
			return 0, 0, err
		}
		if nums[i] < 0 || nums[i] > last {
			return 0, 0, errors.New("Invalid range")
		}
	}
	from, to = nums[0], nums[len(nums)-1]
	if from > to {
		from, to = to, from
	}
	return from, to, nil
}

func parseLineAddr(s string, cur, last int) (int, error) {
	s = strings.TrimSpace(s)
	base, i := cur, 0
	switch {
	case s == "":
		return 0, errors.New("Invalid range")
	case s[0] == '.':
		i = 1
	case s[0] == '$':
		base, i = last, 1
	case s[0] >= '0' && s[0] <= '9':
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		n, _ := strconv.Atoi(s[:i])
		base = n - 1
	}

	// offsets: +N, -N (a bare + or - means 1)
	for i < len(s) {
		sign := s[i]
		if sign != '+' && sign != '-' {
			return 0, errors.New("Invalid range")
		}
		i++
		j := i
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		n := 1
		if j > i {
			n, _ = strconv.Atoi(s[i:j])
		}
		if sign == '+' {
			base += n
		} else {
			base -= n
		}
		i = j
	}
	return base, nil
}

// FilterLines pipes the lines from..to (0-based, inclusive) through cmdline
// and replaces them with its output. The buffer is left untouched if the
// command fails.
func FilterLines(ctx Context, from, to int, cmdline string) error {
	var sb strings.Builder
	line := CursorLineByNum(ctx.Buf, from)
	for n := from; n <= to && line != nil; n++ {
		sb.WriteString(line.Value.String())
		line = line.Next()
	}

	out, err := RunShell(cmdline, sb.String())
	if err != nil {
		first, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
		if first == "" {
			return err
		}
		return fmt.Errorf("%v: %s", err, first)
	}

	if ctx.Buf.TxStart() {
		defer ctx.Buf.TxEnd()
	}
	cur := ContextCursorGet(ctx)
	last := CursorLineByNum(ctx.Buf, to)

	if out == "" {
		// the filter produced nothing: remove the lines
		ctx.Buf.Selection = &Selection{
			Start: Cursor{Line: from},
			End:   Cursor{Line: to, Char: len(last.Value) - 1},
		}
		SelectionDelete(ctx)
	} else {
		// clear the text but keep the last newline, then put the output in its place
		sel := &Selection{
			Start: Cursor{Line: from},
			End:   Cursor{Line: to, Char: len(last.Value) - 2},
		}
		if from < to || sel.End.Char >= 0 {
			ctx.Buf.Selection = sel
			SelectionDelete(ctx)
		}
		TextInsert(ctx.Buf, CursorLineByNum(ctx.Buf, from), 0, strings.TrimSuffix(out, "\n"))
	}

	cur.Line = min(from, ctx.Buf.Lines.Len-1)
	cur.Char = 0
	CmdCursorFirstNonBlank(ctx)
	CmdEnsureCursorVisible(ctx)
	return nil
}
