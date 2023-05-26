package discovery

import (
	"fmt"
	"github.com/hashicorp/serf/serf"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
	"testing"
	"time"
)

func TestMembership(t *testing.T) {
	m, handle := setupMember(t, nil)
	m, _ = setupMember(t, m)
	m, _ = setupMember(t, m)
	// check status
	require.Eventually(t, func() bool {
		return 2 == len(handle.joins) && 3 == len(m[0].Members()) && 0 == len(handle.leaves)
	}, 3*time.Second, 250*time.Millisecond)

	require.NoError(t, m[2].Leave())

	// check status
	require.Eventually(t, func() bool {
		return 2 == len(handle.joins) && 3 == len(m[0].Members()) && serf.StatusLeft == m[0].Members()[2].Status &&
			1 == len(handle.leaves)
	}, 3*time.Second, 250*time.Millisecond)
	require.Equal(t, fmt.Sprintf("%d", 2), <-handle.leaves)
}

func setupMember(t *testing.T, members []*Membership) ([]*Membership, *handler) {
	id := len(members)
	ports := dynaport.Get(1)
	addr := fmt.Sprintf("%s:%d", "127.0.0.1", ports[0])
	tags := map[string]string{"rpc_addr": addr}
	c := Config{
		NodeName: fmt.Sprintf("%d", id),
		BindAddr: addr,
		Tags:     tags,
	}
	h := &handler{}
	if len(members) == 0 {
		h.joins = make(chan map[string]string, 3)
		h.leaves = make(chan string, 3)
	} else {
		c.StartJoinAdds = []string{members[0].BindAddr}
	}
	m, err := New(h, c)
	require.NoError(t, err)
	members = append(members, m)
	return members, h
}

type handler struct {
	joins  chan map[string]string
	leaves chan string
}

func (h *handler) Join(id, addr string) error {
	if h.joins != nil {
		h.joins <- map[string]string{
			"id":   id,
			"addr": addr,
		}
	}
	return nil
}

func (h *handler) Leave(name string) error {
	if h.leaves != nil {
		h.leaves <- name
	}
	return nil
}
