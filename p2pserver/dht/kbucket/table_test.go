/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package kbucket

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/p2pserver/common/set"
	"github.com/ontio/ontology/p2pserver/dht/peer"
	"github.com/stretchr/testify/require"
)

func genpeerID() peer.ID {
	act := account.NewAccount("")
	return peer.IDFromPublicKey(act.PublicKey)
}

// Test basic features of the bucket struct
func TestBucket(t *testing.T) {
	t.Parallel()

	b := newBucket()

	peers := make([]peer.ID, 100)
	for i := 0; i < 100; i++ {
		peers[i] = genpeerID()
		b.PushFront(peers[i])
	}

	local := genpeerID()
	localID := ConvertPeerID(local)

	i := rand.Intn(len(peers))
	if !b.Has(peers[i]) {
		t.Errorf("Failed to find peer: %v", peers[i])
	}

	spl := b.Split(0, ConvertPeerID(local))
	llist := b.list
	for e := llist.Front(); e != nil; e = e.Next() {
		p := ConvertPeerID(e.Value.(peer.ID))
		cpl := CommonPrefixLen(p, localID)
		if cpl > 0 {
			t.Fatalf("Split failed. found id with cpl > 0 in 0 bucket")
		}
	}

	rlist := spl.list
	for e := rlist.Front(); e != nil; e = e.Next() {
		p := ConvertPeerID(e.Value.(peer.ID))
		cpl := CommonPrefixLen(p, localID)
		if cpl == 0 {
			t.Fatalf("Split failed. found id with cpl == 0 in non 0 bucket")
		}
	}
}

func TestRefreshAndGetTrackedCpls(t *testing.T) {
	t.Parallel()

	local := genpeerID()
	rt := NewRoutingTable(1, ConvertPeerID(local))

	// add cpl's for tracking
	for cpl := uint(0); cpl < maxCplForRefresh; cpl++ {
		peerID, err := rt.GenRandPeerID(cpl)
		require.NoError(t, err)
		rt.ResetCplRefreshedAtForID(ConvertPeerID(peerID), time.Now())
	}

	// fetch cpl's
	trackedCpls := rt.GetTrackedCplsForRefresh()
	require.Len(t, trackedCpls, int(maxCplForRefresh))
	actualCpls := make(map[uint]struct{})
	for i := 0; i < len(trackedCpls); i++ {
		actualCpls[trackedCpls[i].Cpl] = struct{}{}
	}

	for i := uint(0); i < maxCplForRefresh; i++ {
		_, ok := actualCpls[i]
		require.True(t, ok, "tracked cpl's should have cpl %d", i)
	}
}

func TestGenRandPeerID(t *testing.T) {
	t.Parallel()

	local := genpeerID()
	rt := NewRoutingTable(1, ConvertPeerID(local))

	// generate above maxCplForRefresh fails
	p, err := rt.GenRandPeerID(maxCplForRefresh + 1)
	require.Error(t, err)
	require.Empty(t, p)

	// test generate rand peer ID
	for cpl := uint(0); cpl <= maxCplForRefresh; cpl++ {
		for i := 0; i < 10000; i++ {
			peerID, err := rt.GenRandPeerID(cpl)
			require.NoError(t, err)

			require.True(t, uint(CommonPrefixLen(ConvertPeerID(peerID), rt.local)) == cpl, "failed for cpl=%d", cpl)
		}
	}
}

func TestTableCallbacks(t *testing.T) {
	t.Parallel()

	local := genpeerID()
	rt := NewRoutingTable(10, ConvertPeerID(local))

	peers := make([]peer.ID, 100)
	for i := 0; i < 100; i++ {
		peers[i] = genpeerID()
	}

	pset := make(map[peer.ID]struct{})
	rt.PeerAdded = func(p peer.ID) {
		pset[p] = struct{}{}
	}
	rt.PeerRemoved = func(p peer.ID) {
		delete(pset, p)
	}

	rt.Update(peers[0])
	if _, ok := pset[peers[0]]; !ok {
		t.Fatal("should have this peer")
	}

	rt.Remove(peers[0])
	if _, ok := pset[peers[0]]; ok {
		t.Fatal("should not have this peer")
	}

	for _, p := range peers {
		rt.Update(p)
	}

	out := rt.ListPeers()
	for _, outp := range out {
		if _, ok := pset[outp]; !ok {
			t.Fatal("should have peer in the peerset")
		}
		delete(pset, outp)
	}

	if len(pset) > 0 {
		t.Fatal("have peers in peerset that were not in the table", len(pset))
	}
}

// Right now, this just makes sure that it doesnt hang or crash
func TestTableUpdate(t *testing.T) {
	t.Parallel()

	local := genpeerID()
	rt := NewRoutingTable(10, ConvertPeerID(local))

	peers := make([]peer.ID, 100)
	for i := 0; i < 100; i++ {
		peers[i] = genpeerID()
	}

	// Testing Update
	for i := 0; i < 10000; i++ {
		rt.Update(peers[rand.Intn(len(peers))])
	}

	for i := 0; i < 100; i++ {
		id := ConvertPeerID(genpeerID())
		ret := rt.NearestPeers(id, 5)
		if len(ret) == 0 {
			t.Fatal("Failed to find node near ID.")
		}
	}
}

func TestTableFind(t *testing.T) {
	t.Parallel()

	local := genpeerID()
	rt := NewRoutingTable(10, ConvertPeerID(local))

	peers := make([]peer.ID, 100)
	for i := 0; i < 5; i++ {
		peers[i] = genpeerID()
		rt.Update(peers[i])
	}

	t.Logf("Searching for peer: '%s'", peers[2].Pretty())
	found, _ := rt.NearestPeer(ConvertPeerID(peers[2]))
	if !(found == peers[2]) {
		t.Fatalf("Failed to lookup known node...")
	}
}

func TestTableEldestPreferred(t *testing.T) {
	t.Parallel()

	local := genpeerID()
	rt := NewRoutingTable(10, ConvertPeerID(local))

	// generate size + 1 peers to saturate a bucket
	peers := make([]peer.ID, 15)
	for i := 0; i < 15; {
		if p := genpeerID(); CommonPrefixLen(ConvertPeerID(local), ConvertPeerID(p)) == 0 {
			peers[i] = p
			i++
		}
	}

	// test 10 first peers are accepted.
	for _, p := range peers[:10] {
		if err := rt.Update(p); err != nil {
			t.Errorf("expected all 10 peers to be accepted; instead got: %v", err)
		}
	}

	// test next 5 peers are rejected.
	for _, p := range peers[10:] {
		err := rt.Update(p)
		if err != ErrPeerRejectedNoCapacity {
			t.Errorf("expected extra 5 peers to be rejected; instead got: %v", err)
		}
		if len(rt.Buckets) != 2 {
			t.Fatalf("buket size not OK")
		}
	}
}

func TestTableFindMultiple(t *testing.T) {
	t.Parallel()

	local := genpeerID()
	rt := NewRoutingTable(20, ConvertPeerID(local))

	peers := make([]peer.ID, 100)
	for i := 0; i < 18; i++ {
		peers[i] = genpeerID()
		rt.Update(peers[i])
	}

	// put in one bucket
	t.Logf("Searching for peer: '%s'", peers[2].Pretty())
	found := rt.NearestPeers(ConvertPeerID(peers[2]), 15)
	if len(found) != 15 {
		t.Fatalf("Got back different number of peers than we expected.")
	}
}

func assertSortedPeerIdsEqual(t *testing.T, a, b []peer.ID) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatal("slices aren't the same length")
	}
	for i, p := range a {
		if p != b[i] {
			t.Fatalf("unexpected peer %d", i)
		}
	}
}

func TestTableFindMultipleBuckets(t *testing.T) {
	t.Parallel()

	local := genpeerID()
	localID := ConvertPeerID(local)

	rt := NewRoutingTable(5, localID)

	peers := make([]peer.ID, 100)
	for i := 0; i < 100; i++ {
		peers[i] = genpeerID()
		rt.Update(peers[i])
	}

	targetID := ConvertPeerID(peers[2])

	closest := SortClosestPeers(rt.ListPeers(), targetID)
	require.Equal(t, closest[0], peers[2], "first one should be the 0 distance with local")

	targetCpl := CommonPrefixLen(localID, targetID)

	// Split the peers into closer, same, and further from the key (than us).
	var (
		closer, same, further []peer.ID
	)
	var i int
	for i = 0; i < len(closest); i++ {
		cpl := CommonPrefixLen(ConvertPeerID(closest[i]), targetID)
		if targetCpl >= cpl {
			break
		}
	}
	closer = closest[:i]

	var j int
	for j = len(closer); j < len(closest); j++ {
		cpl := CommonPrefixLen(ConvertPeerID(closest[j]), targetID)
		if targetCpl > cpl {
			break
		}
	}
	same = closest[i:j]
	further = closest[j:]

	// should be able to find at least 30
	// ~31 (logtwo(100) * 5)
	found := rt.NearestPeers(targetID, 20)
	if len(found) != 20 {
		t.Fatalf("asked for 20 peers, got %d", len(found))
	}

	// We expect this list to include:
	// * All peers with a common prefix length > than the CPL between our key
	//   and the target (peers in the target bucket).
	// * Then a subset of the peers with the _same_ CPL (peers in buckets to the right).
	// * Finally, other peers to the left of the target bucket.

	// First, handle the peers that are _closer_ than us.
	if len(found) <= len(closer) {
		// All of our peers are "closer".
		assertSortedPeerIdsEqual(t, found, closer[:len(found)])
		return
	}
	assertSortedPeerIdsEqual(t, found[:len(closer)], closer)

	// We've worked through closer peers, let's work through peers at the
	// "same" distance.
	found = found[len(closer):]

	// Next, handle peers that are at the same distance as us.
	if len(found) <= len(same) {
		// Ok, all the remaining peers are at the same distance.
		// unfortunately, that means we may be missing peers that are
		// technically closer.

		// Make sure all remaining peers are _somewhere_ in the "closer" set.
		pset := set.NewStringSet()
		for _, p := range same {
			pset.Insert(string(p.ID))
		}
		for _, p := range found {
			if !pset.Has(string(p.ID)) {
				t.Fatalf("unexpected peer %s", p.Pretty())
			}
		}
		return
	}
	assertSortedPeerIdsEqual(t, found[:len(same)], same)

	// We've worked through closer peers, let's work through the further
	// peers.
	found = found[len(same):]

	// All _further_ peers will be correctly sorted.
	assertSortedPeerIdsEqual(t, found, further[:len(found)])

	// Finally, test getting _all_ peers. These should be in the correct
	// order.

	// Ok, now let's try finding all of them.
	found = rt.NearestPeers(ConvertPeerID(peers[2]), 100)
	if len(found) != rt.Size() {
		t.Fatalf("asked for %d peers, got %d", rt.Size(), len(found))
	}

	// We should get _all_ peers in sorted order.
	for i, p := range found {
		if p != closest[i] {
			t.Fatalf("unexpected peer %d", i)
		}
	}
}

// Looks for race conditions in table operations. For a more 'certain'
// test, increase the loop counter from 1000 to a much higher number
// and set GOMAXPROCS above 1
func TestTableMultithreaded(t *testing.T) {
	t.Parallel()

	local := genpeerID()
	tab := NewRoutingTable(20, ConvertPeerID(local))
	var peers []peer.ID
	for i := 0; i < 500; i++ {
		peers = append(peers, genpeerID())
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Update(peers[n])
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Update(peers[n])
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Find(peers[n])
		}
		done <- struct{}{}
	}()
	<-done
	<-done
	<-done
}

func BenchmarkUpdates(b *testing.B) {
	b.StopTimer()
	local := ConvertPeerID(genpeerID())
	tab := NewRoutingTable(20, local)

	var peers []peer.ID
	for i := 0; i < b.N; i++ {
		peers = append(peers, genpeerID())
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tab.Update(peers[i])
	}
}

func BenchmarkFinds(b *testing.B) {
	b.StopTimer()
	local := ConvertPeerID(genpeerID())
	tab := NewRoutingTable(20, local)

	var peers []peer.ID
	for i := 0; i < b.N; i++ {
		peers = append(peers, genpeerID())
		tab.Update(peers[i])
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tab.Find(peers[i])
	}
}
