package log

import (
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"io"
	api "logd/api/v1"
	"os"
	"testing"
)

func TestLog(t *testing.T) {
	for scenario, fn := range map[string]func(t *testing.T, log *Log){
		"append adn read a record succeeds": testAppendRead,
		"offset out of range error":         testOutOfRangeErr,
		"init with existing segments":       testInitExisting,
		"reader":                            testReader,
		"truncate":                          testTruncate,
	} {
		t.Run(scenario, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "store-test")
			require.NoError(t, err)
			defer func() {
				err := os.RemoveAll(dir)
				require.NoError(t, err)
			}()

			c := Config{}
			c.Segment.MaxStoreBytes = 32

			log, err := NewLog(dir, c)
			require.NoError(t, err)
			fn(t, log)
		})
	}
}

func testAppendRead(t *testing.T, log *Log) {
	record := &api.Record{Value: []byte("hello world")}
	off, err := log.Append(record)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	read, err := log.Read(off)
	require.NoError(t, err)
	require.Equal(t, record.Value, read.Value)
}

func testOutOfRangeErr(t *testing.T, log *Log) {
	read, err := log.Read(1)
	require.Nil(t, read)
	apiErr := err.(api.ErrOffsetOutOfRange)
	require.Equal(t, uint64(1), apiErr.Offset)
}

func testInitExisting(t *testing.T, log *Log) {
	record := &api.Record{Value: []byte("hello world")}
	for i := 0; i < 3; i++ {
		_, err := log.Append(record)
		require.NoError(t, err)
	}
	require.NoError(t, log.Close())

	lowestOffset, err := log.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), lowestOffset)
	highestOffset, err := log.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), highestOffset)

	newLog, err := NewLog(log.Dir, log.Config)
	require.NoError(t, err)

	offset, err := newLog.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), offset)

	hOffset, err := newLog.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), hOffset)
}

func testReader(t *testing.T, log *Log) {
	record := &api.Record{Value: []byte("hello world")}
	off, err := log.Append(record)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	reader := log.Reader()
	//ioutil.ReadAll()
	all, err := io.ReadAll(reader)
	require.NoError(t, err)

	read := &api.Record{}
	err = proto.Unmarshal(all[lenWidth:], read)
	require.NoError(t, err)
	require.Equal(t, record.Value, read.Value)
}

func testTruncate(t *testing.T, log *Log) {
	record := &api.Record{Value: []byte("hello world")}
	for i := 0; i < 3; i++ {
		_, err := log.Append(record)
		require.NoError(t, err)
	}
	err := log.Truncate(1)
	require.NoError(t, err)

	_, err = log.Read(0)
	require.Error(t, err)
}
