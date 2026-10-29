// Standard vi motions and edits
package wig

import (
	"strings"
	"unicode"
)

func scrollHalfPage(ctx Context, dir int) {
	cur := ContextCursorGet(ctx)
	step := max(viewHeight(ctx)/2, 1)
	maxOffset := max(ctx.Buf.Lines.Len-minVisibleLines, 0)
	cur.ScrollOffset = min(max(cur.ScrollOffset+dir*step, 0), maxOffset)
	cur.Line = min(max(cur.Line+dir*step, 0), ctx.Buf.Lines.Len-1)
	restoreCharPosition(ctx.Buf, cur)
}

func CmdScrollUpHalfPage(ctx Context) {
	scrollHalfPage(ctx, -1)
}

func CmdScrollDownHalfPage(ctx Context) {
	scrollHalfPage(ctx, 1)
}

// CmdCursorScreenTop moves to the first (or count-th) visible line.
func CmdCursorScreenTop(ctx Context) {
	cur := ContextCursorGet(ctx)
	bottom := min(cur.ScrollOffset+viewHeight(ctx)-1, ctx.Buf.Lines.Len-1)
	cur.Line = min(cur.ScrollOffset+int(max(ctx.Count, 1))-1, bottom)
	CmdCursorFirstNonBlank(ctx)
}

func CmdCursorScreenMiddle(ctx Context) {
	cur := ContextCursorGet(ctx)
	bottom := min(cur.ScrollOffset+viewHeight(ctx)-1, ctx.Buf.Lines.Len-1)
	cur.Line = cur.ScrollOffset + (bottom-cur.ScrollOffset)/2
	CmdCursorFirstNonBlank(ctx)
}

// CmdCursorScreenBottom moves to the last (or count-th from last) visible line.
func CmdCursorScreenBottom(ctx Context) {
	cur := ContextCursorGet(ctx)
	bottom := min(cur.ScrollOffset+viewHeight(ctx)-1, ctx.Buf.Lines.Len-1)
	cur.Line = max(bottom-int(max(ctx.Count, 1))+1, cur.ScrollOffset)
	CmdCursorFirstNonBlank(ctx)
}

func paragraphMove(ctx Context, dir int) {
	defer CmdEnsureCursorVisible(ctx)
	cur := ContextCursorGet(ctx)
	ctx.Editor.ActiveWindow().Jumps.Push(ctx.Buf, cur)

	l := CursorLine(ctx.Buf, cur)
	if l == nil {
		return
	}
	n := cur.Line
	blank := func() bool { return len(l.Value) <= 1 }
	advance := func() bool {
		next := l.Next()
		if dir < 0 {
			next = l.Prev()
		}
		if next == nil {
			return false
		}
		l = next
		n += dir
		return true
	}

	for i := uint32(0); i < max(ctx.Count, 1); i++ {
		for blank() && advance() {
		}
		for !blank() && advance() {
		}
	}

	cur.Line = n
	cur.Char = 0
	cur.PreserveCharPosition = 0
	ctx.Editor.ActiveWindow().Jumps.Push(ctx.Buf, cur)
}

func CmdParagraphForward(ctx Context) {
	paragraphMove(ctx, 1)
}

func CmdParagraphBackward(ctx Context) {
	paragraphMove(ctx, -1)
}

func CmdNextLineFirstNonBlank(ctx Context) {
	CmdCursorLineDown(ctx)
	CmdCursorFirstNonBlank(ctx)
}

func CmdPrevLineFirstNonBlank(ctx Context) {
	CmdCursorLineUp(ctx)
	CmdCursorFirstNonBlank(ctx)
}

// CmdForwardWordEnd moves to the end of the next word.
func CmdForwardWordEnd(ctx Context) {
	defer CmdEnsureCursorVisible(ctx)
	cur := ContextCursorGet(ctx)

	for i := uint32(0); i < max(ctx.Count, 1); i++ {
		if !CursorInc(ctx.Buf, cur) {
			return
		}
		for CursorChClass(ctx.Buf, cur) == chWhitespace {
			if !CursorInc(ctx.Buf, cur) {
				return
			}
		}
		cls := CursorChClass(ctx.Buf, cur)
		for {
			saved := *cur
			if !CursorInc(ctx.Buf, cur) {
				break
			}
			if cur.Line != saved.Line || CursorChClass(ctx.Buf, cur) != cls {
				*cur = saved
				break
			}
		}
	}
}

func CmdInsertFirstNonBlank(ctx Context) {
	CmdCursorFirstNonBlank(ctx)
	CmdEnterInsertMode(ctx)
}

// CmdSubstituteChar deletes the character under the cursor and enters insert mode.
func CmdSubstituteChar(ctx Context) {
	cur := ContextCursorGet(ctx)
	line := CursorLine(ctx.Buf, cur)
	if line == nil {
		return
	}
	CmdEnterInsertMode(ctx)
	if cur.Char >= len(line.Value)-1 {
		return
	}
	ctx.Buf.Selection = &Selection{Start: *cur, End: *cur}
	SelectionDelete(ctx)
}

// CmdToggleCase reverses the case of the character under the cursor and moves right.
func CmdToggleCase(ctx Context) {
	cur := ContextCursorGet(ctx)
	line := CursorLine(ctx.Buf, cur)
	if line == nil || cur.Char >= len(line.Value)-1 {
		return
	}
	r := line.Value[cur.Char]
	toggled := unicode.ToUpper(r)
	if toggled == r {
		toggled = unicode.ToLower(r)
	}
	if toggled != r {
		if ctx.Buf.TxStart() {
			defer ctx.Buf.TxEnd()
		}
		ctx.Buf.Selection = &Selection{Start: *cur, End: *cur}
		SelectionDelete(ctx)
		TextInsert(ctx.Buf, line, cur.Char, string(toggled))
	}
	CmdCursorRight(ctx)
}

// CmdRedrawScreen re-renders the editor and forces a full terminal repaint.
func CmdRedrawScreen(ctx Context) {
	ctx.Editor.Redraw()
	ctx.Editor.ScreenSync()
}

func cursorOnSpace(ctx Context, cur *Cursor) bool {
	return CursorChClass(ctx.Buf, cur) == chWhitespace
}

// CmdForwardWORD moves to the start of the next whitespace-delimited WORD.
func CmdForwardWORD(ctx Context) {
	defer CmdEnsureCursorVisible(ctx)
	cur := ContextCursorGet(ctx)

	for i := uint32(0); i < max(ctx.Count, 1); i++ {
		for !cursorOnSpace(ctx, cur) {
			if !CursorInc(ctx.Buf, cur) {
				return
			}
		}
		for cursorOnSpace(ctx, cur) {
			if !CursorInc(ctx.Buf, cur) {
				return
			}
		}
	}
}

// CmdBackwardWORD moves to the start of the previous whitespace-delimited WORD.
func CmdBackwardWORD(ctx Context) {
	defer CmdEnsureCursorVisible(ctx)
	cur := ContextCursorGet(ctx)

	for i := uint32(0); i < max(ctx.Count, 1); i++ {
		if !CursorDec(ctx.Buf, cur) {
			return
		}
		for cursorOnSpace(ctx, cur) {
			if !CursorDec(ctx.Buf, cur) {
				return
			}
		}
		for {
			saved := *cur
			if !CursorDec(ctx.Buf, cur) {
				break
			}
			if cursorOnSpace(ctx, cur) {
				*cur = saved
				break
			}
		}
	}
}

// CmdForwardWORDEnd moves to the end of the next whitespace-delimited WORD.
func CmdForwardWORDEnd(ctx Context) {
	defer CmdEnsureCursorVisible(ctx)
	cur := ContextCursorGet(ctx)

	for i := uint32(0); i < max(ctx.Count, 1); i++ {
		if !CursorInc(ctx.Buf, cur) {
			return
		}
		for cursorOnSpace(ctx, cur) {
			if !CursorInc(ctx.Buf, cur) {
				return
			}
		}
		for {
			saved := *cur
			if !CursorInc(ctx.Buf, cur) {
				break
			}
			if cursorOnSpace(ctx, cur) {
				*cur = saved
				break
			}
		}
	}
}

// CmdCursorBackward moves one character back, wrapping to the previous line.
func CmdCursorBackward(ctx Context) {
	defer CmdEnsureCursorVisible(ctx)
	cur := ContextCursorGet(ctx)
	for i := uint32(0); i < max(ctx.Count, 1); i++ {
		if !CursorDec(ctx.Buf, cur) {
			return
		}
	}
}

// markFromCtx returns the mark named by ctx.Char, if it's set in the current buffer.
func markFromCtx(ctx Context) (Mark, bool) {
	name := strings.TrimPrefix(ctx.Char, "shift+")
	if name == "" {
		return Mark{}, false
	}
	win := ctx.Win
	if win == nil {
		win = ctx.Editor.ActiveWindow()
	}
	var m Mark
	ok := false
	if win != nil {
		m, ok = win.Marks[[]rune(name)[0]]
	}
	if !ok || (m.Buf != nil && m.Buf != ctx.Buf) {
		ctx.Editor.EchoMessage("Mark '" + name + "' not set")
		return Mark{}, false
	}
	return m, true
}

// yankLinesTo yanks the whole lines from the cursor line to target (inclusive).
func yankLinesTo(ctx Context, target int) {
	cur := ContextCursorGet(ctx)
	from, to := min(cur.Line, target), max(cur.Line, target)

	var sb strings.Builder
	line := CursorLineByNum(ctx.Buf, from)
	for n := from; n <= to && line != nil; n++ {
		sb.WriteString(line.Value.String())
		line = line.Next()
	}

	ctx.Buf.Selection = nil
	yankSave(ctx, sb.String())
	if from < cur.Line {
		cur.Line = from
		restoreCharPosition(ctx.Buf, cur)
		CmdEnsureCursorVisible(ctx)
	}
}

// CmdYankLinesToMark yanks the lines from the cursor to a mark (y'a).
func CmdYankLinesToMark(_ Context) func(Context) {
	return func(ctx Context) {
		if m, ok := markFromCtx(ctx); ok {
			yankLinesTo(ctx, m.Cursor.Line)
		}
	}
}

// markCharSelection returns the selection from the cursor to a mark position.
// The motion is exclusive, like vi's `a: it stops just before the later position.
func markCharSelection(ctx Context, m Mark) (*Selection, bool) {
	start, end := *ContextCursorGet(ctx), m.Cursor
	if end.Line < start.Line || (end.Line == start.Line && end.Char < start.Char) {
		start, end = end, start
	}
	if start.Line == end.Line && start.Char == end.Char {
		return nil, false
	}

	if end.Char > 0 {
		end.Char--
	} else {
		end.Line--
		prev := CursorLineByNum(ctx.Buf, end.Line)
		end.Char = max(len(prev.Value)-1, 0)
	}

	sel := &Selection{Start: start, End: end}
	if SelectionToString(ctx.Buf, sel) == "" {
		return nil, false
	}
	return sel, true
}

// CmdYankToMark yanks from the cursor to a mark position, exclusive (y`a).
func CmdYankToMark(_ Context) func(Context) {
	return func(ctx Context) {
		m, ok := markFromCtx(ctx)
		if !ok {
			return
		}
		sel, ok := markCharSelection(ctx, m)
		if !ok {
			return
		}
		ctx.Buf.Selection = sel
		yankSave(ctx, SelectionToString(ctx.Buf, sel))
		ctx.Buf.Selection = nil
		*ContextCursorGet(ctx) = sel.Start
		CmdEnsureCursorVisible(ctx)
	}
}

// linesSelection moves the cursor to the first line between the cursor and
// target and selects the whole lines in visual-line mode.
func linesSelection(ctx Context, target int) {
	cur := ContextCursorGet(ctx)
	from, to := min(cur.Line, target), max(cur.Line, target)
	cur.Line = from
	CmdVisualLineMode(ctx)
	ctx.Buf.Selection.End.Line = to
	ctx.Buf.Selection.End.Char = len(CursorLineByNum(ctx.Buf, to).Value) - 1
}

// deleteLinesTo deletes the whole lines from the cursor line to target (inclusive).
func deleteLinesTo(ctx Context, target int) {
	if ctx.Buf.TxStart() {
		defer ctx.Buf.TxEnd()
	}
	linesSelection(ctx, target)
	yankSave(ctx)
	SelectionDelete(ctx)
	CmdNormalMode(ctx)
	CmdEnsureCursorVisible(ctx)
}

// CmdDeleteLinesToMark deletes the lines from the cursor to a mark (d'a).
func CmdDeleteLinesToMark(_ Context) func(Context) {
	return func(ctx Context) {
		if m, ok := markFromCtx(ctx); ok {
			deleteLinesTo(ctx, m.Cursor.Line)
		}
	}
}

// CmdDeleteToMark deletes from the cursor to a mark position, exclusive (d`a).
func CmdDeleteToMark(_ Context) func(Context) {
	return func(ctx Context) {
		m, ok := markFromCtx(ctx)
		if !ok {
			return
		}
		sel, ok := markCharSelection(ctx, m)
		if !ok {
			return
		}
		if ctx.Buf.TxStart() {
			defer ctx.Buf.TxEnd()
		}
		ctx.Buf.Selection = sel
		yankSave(ctx)
		SelectionDelete(ctx)
		CmdEnsureCursorVisible(ctx)
	}
}

// changeLinesTo replaces the lines from the cursor line to target with an
// empty line and enters insert mode.
func changeLinesTo(ctx Context, target int) {
	linesSelection(ctx, target)
	yankSave(ctx)

	// keep the last line's newline so an empty line is left behind
	ctx.Buf.Selection.End.Char--
	CmdEnterInsertMode(ctx)
	SelectionDelete(ctx)
	CmdEnsureCursorVisible(ctx)
}

// CmdChangeLinesToMark replaces the lines from the cursor to a mark with an
// empty line and enters insert mode (c'a).
func CmdChangeLinesToMark(_ Context) func(Context) {
	return func(ctx Context) {
		if m, ok := markFromCtx(ctx); ok {
			changeLinesTo(ctx, m.Cursor.Line)
		}
	}
}

// CmdChangeToMark deletes from the cursor to a mark position, exclusive, and
// enters insert mode (c`a).
func CmdChangeToMark(_ Context) func(Context) {
	return func(ctx Context) {
		m, ok := markFromCtx(ctx)
		if !ok {
			return
		}
		sel, ok := markCharSelection(ctx, m)
		if !ok {
			return
		}
		CmdEnterInsertMode(ctx)
		ctx.Buf.Selection = sel
		yankSave(ctx)
		SelectionDelete(ctx)
		CmdEnsureCursorVisible(ctx)
	}
}

// lineMotionTarget is the line a G (or gg) motion goes to: the count-th line
// if given, otherwise the last (or first) line.
func lineMotionTarget(ctx Context, last bool) int {
	if ctx.Count > 0 {
		return min(int(ctx.Count)-1, ctx.Buf.Lines.Len-1)
	}
	if last {
		return ctx.Buf.Lines.Len - 1
	}
	return 0
}

// dG, yG, cG and {count}G variants: linewise, to the end of file or line N.
func CmdYankToLastLine(ctx Context)   { yankLinesTo(ctx, lineMotionTarget(ctx, true)) }
func CmdDeleteToLastLine(ctx Context) { deleteLinesTo(ctx, lineMotionTarget(ctx, true)) }
func CmdChangeToLastLine(ctx Context) { changeLinesTo(ctx, lineMotionTarget(ctx, true)) }

// dgg, ygg, cgg and {count}gg variants: linewise, to the start of file or line N.
func CmdYankToFirstLine(ctx Context)   { yankLinesTo(ctx, lineMotionTarget(ctx, false)) }
func CmdDeleteToFirstLine(ctx Context) { deleteLinesTo(ctx, lineMotionTarget(ctx, false)) }
func CmdChangeToFirstLine(ctx Context) { changeLinesTo(ctx, lineMotionTarget(ctx, false)) }
