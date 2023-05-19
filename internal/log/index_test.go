package log

import (
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func TestIndex(t *testing.T) {
	file, err := os.CreateTemp("", "index_test")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	c := Config{}
	c.Segment.MaxIndexBytes = 1024
	index, err := newIndex(file, c)
	require.NoError(t, err)

	_, _, err = index.Read(-1)
	require.NoError(t, err)
	require.Equal(t, file.Name(), index.Name())

	entries := []struct {
		Off uint32
		Pos uint64
	}{
		{Off: 0, Pos: 0},
		{Off: 1, Pos: 10},
	}

	for _, want := range entries {
		err := index.Write(want.Off, want.Pos)
		require.NoError(t, err)

		_, pos, err := index.Read(int64(want.Off))
		require.NoError(t, err)
		require.Equal(t, pos, want.Pos)
	}
	_, _, err = index.Read(int64(len(entries)))
	require.Equal(t, io.EOF, err)
	_ = index.Close()

	file, _ = os.OpenFile(file.Name(), os.O_RDWR, 0600)
	idx, err := newIndex(file, c)
	require.NoError(t, err)
	off, pos, err := idx.Read(-1)
	require.NoError(t, err)
	require.Equal(t, uint32(1), off)
	require.Equal(t, entries[1].Pos, pos)
}
