package wig

import (
	"testing"

	"github.com/firstrow/wig/testutils"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	e := NewEditor(testutils.Viewport, nil)
	buf, _ := e.OpenFile(testutils.Filepath("buffer_test.txt"))

	ctx := Context{
		Editor: e,
		Buf:    buf,
	}
	cur := CursorGet(e, buf)

	SearchNext(ctx, "one")
	require.Equal(t, 0, cur.Line)
	require.Equal(t, 5, cur.Char)

	SearchNext(ctx, "three")
	require.Equal(t, 2, cur.Line)
	require.Equal(t, 5, cur.Char)

	SearchPrev(ctx, "one")
	require.Equal(t, 0, cur.Line)
	require.Equal(t, 5, cur.Char)
}

func TestSearchDirection(t *testing.T) {
	e := NewEditor(testutils.Viewport, nil)
	buf, _ := e.OpenFile(testutils.Filepath("buffer_test.txt"))

	ctx := Context{
		Editor: e,
		Buf:    buf,
	}
	cur := CursorGet(e, buf)

	t.Cleanup(func() {
		LastSearchPattern = ""
		LastSearchBackward = false
	})

	// "e " matches at char 3 of every line.
	LastSearchPattern = "e "

	// after "/" n goes forward and N goes back
	LastSearchBackward = false
	CmdSearchNext(ctx)
	require.Equal(t, 0, cur.Line)
	CmdSearchNext(ctx)
	require.Equal(t, 1, cur.Line)
	CmdSearchNext(ctx)
	require.Equal(t, 2, cur.Line)
	CmdSearchPrev(ctx)
	require.Equal(t, 1, cur.Line)

	// after "?" n goes back and N goes forward
	LastSearchBackward = true
	CmdSearchNext(ctx)
	require.Equal(t, 0, cur.Line)
	require.Equal(t, 3, cur.Char)
	CmdSearchPrev(ctx)
	require.Equal(t, 1, cur.Line)
	require.Equal(t, 3, cur.Char)
}
