package wig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/firstrow/wig/testutils"
	"github.com/stretchr/testify/require"
)

func TestWriteToCopy(t *testing.T) {
	e := NewEditor(testutils.Viewport, nil)
	buf, _ := e.OpenFile(testutils.Filepath("buffer_test.txt"))
	ctx := Context{Editor: e, Buf: buf}

	orig := buf.FilePath
	dst := filepath.Join(t.TempDir(), "copy.txt")
	require.NoError(t, WriteTo(ctx, dst))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, buf.String(), string(got))
	// the buffer keeps its own file
	require.Equal(t, orig, buf.FilePath)
}

func TestEditForce(t *testing.T) {
	e := NewEditor(testutils.Viewport, nil)
	path := filepath.Join(t.TempDir(), "f.txt")
	require.NoError(t, os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644))

	buf, err := e.OpenFile(path)
	require.NoError(t, err)
	e.ActiveWindow().ShowBuffer(buf)
	ctx := Context{Editor: e, Buf: buf, Win: e.ActiveWindow()}

	// change the buffer and leave the cursor on the last line
	ReloadBufferContent(ctx, "changed\n")
	buf.Dirty = true
	ContextCursorGet(ctx).Line = 5

	require.NoError(t, EditForce(ctx))
	require.Equal(t, "one\ntwo\nthree\n", buf.String())
	require.False(t, buf.Dirty)
	require.Equal(t, 2, ContextCursorGet(ctx).Line)

	// the file on disk changed under us
	require.NoError(t, os.WriteFile(path, []byte("new\n"), 0o644))
	require.NoError(t, EditForce(ctx))
	require.Equal(t, "new\n", buf.String())
}

func TestEditForceNoFile(t *testing.T) {
	e := NewEditor(testutils.Viewport, nil)
	buf := NewBuffer()
	require.Error(t, EditForce(Context{Editor: e, Buf: buf}))

	buf.FilePath = filepath.Join(t.TempDir(), "missing.txt")
	require.Error(t, EditForce(Context{Editor: e, Buf: buf}))
}
