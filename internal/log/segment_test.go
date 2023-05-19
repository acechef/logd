package log

import (
	"github.com/stretchr/testify/require"
	"io"
	api "logd/api/v1"
	"os"
	"testing"
)

func TestSegment(t *testing.T) {
	dir, _ := os.MkdirTemp("", "segment-test")
	defer os.RemoveAll(dir)

	want := &api.Record{
		Value: []byte("hello world"),
	}

	c := Config{}
	c.Segment.MaxIndexBytes = endWidth * 3
	c.Segment.MaxStoreBytes = 1024

	s, err := newSegment(dir, 16, c)
	require.NoError(t, err)
	require.Equal(t, uint64(16), s.nextOffset, s.nextOffset)
	require.False(t, s.IsMaxed())

	for i := uint64(0); i < 3; i++ {
		offset, err := s.Append(want)
		require.NoError(t, err)
		require.Equal(t, 16+i, offset)

		got, err := s.Read(offset)
		require.NoError(t, err)
		require.Equal(t, got.Value, want.Value)
	}

	_, err = s.Append(want)
	require.Equal(t, io.EOF, err)

	// maxed index
	require.True(t, s.IsMaxed())

	c.Segment.MaxStoreBytes = uint64(len(want.Value) * 3)
	c.Segment.MaxIndexBytes = 1024

	s, err = newSegment(dir, 16, c)
	require.NoError(t, err)
	// maxed store
	require.True(t, s.IsMaxed())

	err = s.Remove()
	require.NoError(t, err)
	s, err = newSegment(dir, 16, c)
	require.NoError(t, err)
	require.False(t, s.IsMaxed())

}
