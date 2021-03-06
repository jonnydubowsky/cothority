package eventlog

import (
	"fmt"
	"testing"
	"time"

	"github.com/dedis/cothority/omniledger/darc"
	omniledger "github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/cothority/skipchain"
	"github.com/dedis/protobuf"
	"github.com/stretchr/testify/require"
)

// Use this block interval for logic tests. Stress test often use a different
// block interval.
var testBlockInterval = 100 * time.Millisecond

func TestClient_Log(t *testing.T) {
	s := newSer(t)
	leader := s.services[0]
	defer s.close()

	owner := darc.NewSignerEd25519(nil, nil)
	c := NewClient(s.roster)
	err := c.Init(owner, testBlockInterval)
	require.Nil(t, err)

	ids, err := c.Log(NewEvent("auth", "user alice logged out"),
		NewEvent("auth", "user bob logged out"))
	require.Nil(t, err)
	require.True(t, len(ids) == 2)

	// Loop while we wait for the next block to be created.
	found := false
	for ct := 0; ct < 10; ct++ {
		req := &omniledger.GetProof{
			Version: omniledger.CurrentVersion,
			Key:     ids[1],
			ID:      c.ID,
		}
		resp, err := leader.omni.GetProof(req)
		if err == nil {
			p := resp.Proof.InclusionProof
			if p.Match() {
				found = true
				break
			}
		} else {
			t.Log("err", err)
		}
		time.Sleep(1 * time.Second)
	}
	if !found {
		t.Fatal("timeout")
	}

	// Fetch index, and check its length.
	idx := checkProof(t, leader.omni, indexKey.Slice(), c.ID)
	expected := 64
	require.Equal(t, len(idx), expected, fmt.Sprintf("index key content is %v, expected %v", len(idx), expected))

	// Fetch the bucket and check its length.
	bucketBuf := checkProof(t, leader.omni, idx, c.ID)
	var b bucket
	require.Nil(t, protobuf.Decode(bucketBuf, &b))
	require.Equal(t, 2, len(b.EventRefs))
	require.Equal(t, 0, len(b.Prev))

	// Use the client API to get the event back
	for _, key := range ids {
		_, err = c.GetEvent(key)
		require.Nil(t, err)
	}
}

// TODO this test only passes when the block interval is long enough
func TestClient_Log1000(t *testing.T) {
	if testing.Short() {
		return
	}
	s := newSer(t)
	leader := s.services[0]
	defer s.close()

	owner := darc.NewSignerEd25519(nil, nil)
	c := NewClient(s.roster)
	err := c.Init(owner, time.Second)
	require.Nil(t, err)

	logCount := 1000
	for ct := 0; ct < logCount; ct++ {
		_, err := c.Log(NewEvent("auth", fmt.Sprintf("user %v logged in", ct)))
		require.Nil(t, err)
	}
	leader.waitForBlock(c.ID)
	leader.waitForBlock(c.ID)

	// Fetch index, and check its length.
	idx := checkProof(t, leader.omni, indexKey.Slice(), c.ID)
	expected := 64
	require.Equal(t, len(idx), expected, fmt.Sprintf("index key content is %v, expected %v", len(idx), expected))

	// Fetch the bucket and check its length.
	bucketID := idx
	var eventCount int
	var eventIDs [][]byte
	for {
		if len(bucketID) == 0 {
			break
		}
		bucketBuf := checkProof(t, leader.omni, bucketID, c.ID)
		var b bucket
		require.Nil(t, protobuf.Decode(bucketBuf, &b))
		require.NotEqual(t, bucketID, b.Prev)
		eventCount += len(b.EventRefs)
		eventIDs = append(eventIDs, b.EventRefs...)
		bucketID = b.Prev
	}
	require.Equal(t, logCount, eventCount)

	for _, eventID := range eventIDs {
		eventBuf := checkProof(t, leader.omni, eventID, c.ID)
		var e Event
		require.Nil(t, protobuf.Decode(eventBuf, &e))
	}
}

func checkProof(t *testing.T, omni *omniledger.Service, key []byte, scID skipchain.SkipBlockID) []byte {
	req := &omniledger.GetProof{
		Version: omniledger.CurrentVersion,
		Key:     key,
		ID:      scID,
	}
	resp, err := omni.GetProof(req)
	require.Nil(t, err)

	p := resp.Proof.InclusionProof
	require.True(t, p.Match(), "proof of exclusion of index")

	v, _ := p.Values()
	require.Equal(t, 2, len(v), "wrong values length")

	return v[0].([]byte)
}
