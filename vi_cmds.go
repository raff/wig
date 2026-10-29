// Standard vi motions and edits
package wig

import "unicode"

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
