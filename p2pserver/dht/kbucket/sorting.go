package kbucket

import (
	"container/list"
	"sort"
)

// A helper struct to sort peers by their distance to the local node
type peerDistance struct {
	p        uint64
	distance ID
}

// peerDistanceSorter implements sort.Interface to sort peers by xor distance
type peerDistanceSorter struct {
	peers  []peerDistance
	target ID
}

func (pds *peerDistanceSorter) Len() int      { return len(pds.peers) }
func (pds *peerDistanceSorter) Swap(a, b int) { pds.peers[a], pds.peers[b] = pds.peers[b], pds.peers[a] }
func (pds *peerDistanceSorter) Less(a, b int) bool {
	return pds.peers[a].distance.less(pds.peers[b].distance)
}

// Append the peer.ID to the sorter's slice. It may no longer be sorted.
func (pds *peerDistanceSorter) appendPeer(p uint64) {
	pds.peers = append(pds.peers, peerDistance{
		p:        p,
		distance: xor(pds.target, ConvertPeerID(p)),
	})
}

// Append the peer.ID values in the list to the sorter's slice. It may no longer be sorted.
func (pds *peerDistanceSorter) appendPeersFromList(l *list.List) {
	for e := l.Front(); e != nil; e = e.Next() {
		pds.appendPeer(e.Value.(uint64))
	}
}

func (pds *peerDistanceSorter) sort() {
	sort.Sort(pds)
}

// Sort the given peers by their ascending distance from the target. A new slice is returned.
func SortClosestPeers(peers []uint64, target ID) []uint64 {
	sorter := peerDistanceSorter{
		peers:  make([]peerDistance, 0, len(peers)),
		target: target,
	}
	for _, p := range peers {
		sorter.appendPeer(p)
	}
	sorter.sort()
	out := make([]uint64, 0, sorter.Len())
	for _, p := range sorter.peers {
		out = append(out, p.p)
	}
	return out
}
