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

package discovery

import (
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/p2pserver/common"
	"github.com/ontio/ontology/p2pserver/dht"
	msgpack "github.com/ontio/ontology/p2pserver/message/msg_pack"
	"github.com/ontio/ontology/p2pserver/message/types"
	p2p "github.com/ontio/ontology/p2pserver/net/protocol"
	"github.com/ontio/ontology/p2pserver/peer"
	"github.com/scylladb/go-set/strset"
)

const (
	cleanRSearchTime = time.Minute * 5
)

type RecurSearch struct {
	AddrChan  chan string                // the final address put through here
	Visited   map[common.PeerId]struct{} // the visisted set for search target
	Todo      map[common.PeerId]struct{} // when connected to those node, we will send request to it
	StartTime time.Time                  // create time
}

func NewRecurSearch(ch chan string) *RecurSearch {
	return &RecurSearch{
		AddrChan:  ch,
		Visited:   make(map[common.PeerId]struct{}),
		Todo:      make(map[common.PeerId]struct{}),
		StartTime: time.Now(),
	}
}

func (rs *RecurSearch) TryVisit(id common.PeerId) bool {
	if _, exist := rs.Visited[id]; exist {
		return false
	}

	rs.Visited[id] = struct{}{}
	return true
}

func (rs *RecurSearch) AddTodo(id common.PeerId) {
	rs.Todo[id] = struct{}{}
}

func (rs *RecurSearch) TryTodo(id common.PeerId) bool {
	_, ok := rs.Todo[id]
	delete(rs.Todo, id)

	return ok
}

type Discovery struct {
	dht     *dht.DHT
	net     p2p.P2P
	id      common.PeerId
	quit    chan bool
	maskSet *strset.Set

	rcLock     sync.Mutex
	recurCache map[common.PeerId]*RecurSearch
}

func NewDiscovery(net p2p.P2P, maskLst []string, refleshInterval time.Duration) *Discovery {
	dht := dht.NewDHT(net.GetID())
	if refleshInterval != 0 {
		dht.RtRefreshPeriod = refleshInterval
	}
	return &Discovery{
		id:         net.GetID(),
		dht:        dht,
		net:        net,
		quit:       make(chan bool),
		maskSet:    strset.New(maskLst...),
		recurCache: make(map[common.PeerId]*RecurSearch),
	}
}

func (self *Discovery) Start() {
	go self.findSelf()
	go self.refreshCPL()
	go self.cleanRecurSearch()
}

func (self *Discovery) Stop() {
	close(self.quit)
}

func (self *Discovery) OnAddPeer(info *peer.PeerInfo) {
	self.dht.Update(info.Id, info.RemoteListenAddress())

	self.rcLock.Lock()
	defer self.rcLock.Unlock()
	// newly connected node is in todo list
	for id, entry := range self.recurCache {
		if entry.TryTodo(info.Id) && entry.TryVisit(info.Id) {
			self.net.SendTo(info.Id, msgpack.NewRecursiveFindNodeReq(id))
		}
	}

	// re find those searched id
	for id, entry := range self.recurCache {
		self.reSearchWithLock(id, entry)
	}
}

func (self *Discovery) OnDelPeer(info *peer.PeerInfo) {
	self.dht.Remove(info.Id)
}

func (self *Discovery) findSelf() {
	tick := time.NewTicker(self.dht.RtRefreshPeriod)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			log.Debug("[dht] start to find myself")
			closer := self.dht.BetterPeers(self.id, dht.AlphaValue)
			for _, curPair := range closer {
				log.Debugf("[dht] find closr peer %s", curPair.ID.ToHexString())

				var msg types.Message
				if curPair.ID.IsPseudoPeerId() {
					msg = msgpack.NewAddrReq()
				} else {
					msg = msgpack.NewFindNodeReq(curPair.ID)
				}
				self.net.SendTo(curPair.ID, msg)
			}
		case <-self.quit:
			return
		}
	}
}

func (self *Discovery) refreshCPL() {
	tick := time.NewTicker(self.dht.RtRefreshPeriod)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			for curCPL := range self.dht.RouteTable().Buckets {
				log.Debugf("[dht] start to refresh bucket: %d", curCPL)
				randPeer := self.dht.RouteTable().GenRandKadId(uint(curCPL))
				closer := self.dht.BetterPeers(randPeer, dht.AlphaValue)
				for _, pair := range closer {
					log.Debugf("[dht] find closr peer %s", pair.ID.ToHexString())
					var msg types.Message
					if pair.ID.IsPseudoPeerId() {
						msg = msgpack.NewAddrReq()
					} else {
						msg = msgpack.NewFindNodeReq(randPeer)
					}
					self.net.SendTo(pair.ID, msg)
				}
			}
		case <-self.quit:
			return
		}
	}
}

func (self *Discovery) FindNodeHandle(ctx *p2p.Context, freq *types.FindNodeReq) {
	// we recv message must from establised peer
	remotePeer := ctx.Sender()

	var fresp types.FindNodeResp
	// check the target is my self
	log.Debugf("[dht] find node for peerid: %d", freq.TargetID)

	if freq.TargetID == self.id {
		fresp.Success = true
		fresp.TargetID = freq.TargetID
		// you've already connected with me so there's no need to give you my address
		// omit the address
		if err := remotePeer.Send(&fresp); err != nil {
			log.Warn(err)
		}
		return
	}

	fresp.TargetID = freq.TargetID
	fresp.Recursive = freq.Recursive
	// search dht
	fresp.CloserPeers = self.dht.BetterPeers(freq.TargetID, dht.AlphaValue)

	//hide mask node if necessary
	remoteAddr, _ := remotePeer.GetAddr16()
	remoteIP := net.IP(remoteAddr[:])

	// mask peer see everyone, but other's will not see mask node
	// if remotePeer is in msk-list, give them everything
	// not in mask set means they are in the other side
	if self.maskSet.Size() > 0 && !self.maskSet.Has(remoteIP.String()) {
		mskedAddrs := make([]common.PeerIDAddressPair, 0)
		// filter out the masked node
		for _, pair := range fresp.CloserPeers {
			ip, _, err := net.SplitHostPort(pair.Address)
			if err != nil {
				continue
			}
			// hide mask node
			if self.maskSet.Has(ip) {
				continue
			}
			mskedAddrs = append(mskedAddrs, pair)
		}
		// replace with masked nodes
		fresp.CloserPeers = mskedAddrs
	}

	log.Debugf("[dht] find %d more closer peers:", len(fresp.CloserPeers))
	for _, curpa := range fresp.CloserPeers {
		log.Debugf("    dht: pid: %s, addr: %s", curpa.ID.ToHexString(), curpa.Address)
	}

	if err := remotePeer.Send(&fresp); err != nil {
		log.Warn(err)
	}
}

func (self *Discovery) FindNodeResponseHandle(ctx *p2p.Context, fresp *types.FindNodeResp) {
	if fresp.Success {
		log.Debugf("[p2p dht] %s", "find peer success, do nothing")
		if fresp.Recursive {
			self.doRecursiveResp(fresp)
		}
		return
	}
	p2p := ctx.Network()

	// for connected node we should send the req again
	// so another palce is connected callback
	if fresp.Recursive {
		self.rcLock.Lock()
		for _, curpa := range fresp.CloserPeers {
			if curpa.ID == p2p.GetID() {
				continue
			}
			entry, ok := self.recurCache[fresp.TargetID]
			if !ok {
				continue
			}

			// disconnected
			if p2p.GetPeer(curpa.ID) == nil {
				// add it to todo
				entry.AddTodo(curpa.ID)
				continue
			}

			// now this peer is connected
			if entry.TryVisit(curpa.ID) {
				p2p.SendTo(curpa.ID, msgpack.NewRecursiveFindNodeReq(fresp.TargetID))
			}
		}
		self.rcLock.Unlock()
	}
	// we should connect to closer peer to ask them them where should we go
	for _, curpa := range fresp.CloserPeers {
		// already connected
		if p2p.GetPeer(curpa.ID) != nil {
			continue
		}
		// do nothing about
		if curpa.ID == p2p.GetID() {
			continue
		}
		log.Debugf("[dht] try to connect to another peer by dht: %s ==> %s", curpa.ID.ToHexString(), curpa.Address)
		go p2p.Connect(curpa.Address)
	}
}

// neighborAddresses get address from dht routing table
func (self *Discovery) neighborAddresses() []common.PeerAddr {
	// e.g. ["127.0.0.1:20338"]
	ipPortAdds := self.dht.RouteTable().ListPeers()
	ret := []common.PeerAddr{}
	for _, curIPPort := range ipPortAdds {
		host, port, err := net.SplitHostPort(curIPPort.Address)
		if err != nil {
			continue
		}

		ipadd := net.ParseIP(host)
		if ipadd == nil {
			continue
		}

		p, err := strconv.Atoi(port)
		if err != nil {
			continue
		}

		curAddr := common.PeerAddr{
			Port: uint16(p),
		}
		copy(curAddr.IpAddr[:], ipadd.To16())

		ret = append(ret, curAddr)
	}

	return ret
}

func (self *Discovery) AddrReqHandle(ctx *p2p.Context) {
	remotePeer := ctx.Sender()

	addrs := self.neighborAddresses()

	// get remote peer IP
	// if get remotePeerAddr failed, do masking anyway
	remoteAddr, _ := remotePeer.GetAddr16()
	remoteIP := net.IP(remoteAddr[:])

	// mask peer see everyone, but other's will not see mask node
	// if remotePeer is in msk-list, give them everthing
	// not in mask set means they are in the other side
	if self.maskSet.Size() > 0 && !self.maskSet.Has(remoteIP.String()) {
		mskedAddrs := make([]common.PeerAddr, 0)
		for _, addr := range addrs {
			ip := net.IP(addr.IpAddr[:])
			address := ip.To16().String()
			// hide mask node
			if self.maskSet.Has(address) {
				continue
			}
			mskedAddrs = append(mskedAddrs, addr)
		}
		// replace with mskedAddrs
		addrs = mskedAddrs
	}

	msg := msgpack.NewAddrs(addrs)
	err := remotePeer.Send(msg)

	if err != nil {
		log.Warn(err)
		return
	}
}

func (self *Discovery) AddrHandle(ctx *p2p.Context, msg *types.Addr) {
	p2p := ctx.Network()
	for _, v := range msg.NodeAddrs {
		if v.Port == 0 || v.ID == p2p.GetID() {
			continue
		}
		ip := net.IP(v.IpAddr[:])
		address := ip.To16().String() + ":" + strconv.Itoa(int(v.Port))

		if self.dht.Contains(v.ID) {
			continue
		}

		log.Debug("[p2p]connect ip address:", address)
		go p2p.Connect(address)
	}
}

func (self *Discovery) DHT() *dht.DHT {
	return self.dht
}

func (self *Discovery) MakeRecursiveEntry(target common.PeerId, ch chan string) *RecurSearch {
	rs := NewRecurSearch(ch)

	self.rcLock.Lock()
	defer self.rcLock.Unlock()

	self.recurCache[target] = rs

	return rs
}

func (self *Discovery) doRecursiveResp(resp *types.FindNodeResp) {
	self.rcLock.Lock()
	defer self.rcLock.Unlock()

	entry, ok := self.recurCache[resp.TargetID]
	if !ok {
		return
	}
	go func(ch chan<- string) {
		select {
		case entry.AddrChan <- resp.Address:
		default:
		}
		close(entry.AddrChan)
	}(entry.AddrChan)

	delete(self.recurCache, resp.TargetID)
}

func (self *Discovery) cleanRecurSearch() {
	tick := time.NewTicker(cleanRSearchTime)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			self.rcLock.Lock()
			for target, entry := range self.recurCache {
				if time.Since(entry.StartTime) > time.Minute*3 {
					delete(self.recurCache, target)
				}
			}
			self.rcLock.Unlock()
		case <-self.quit:
			return
		}
	}
}

func (self *Discovery) TryVisit(key, id common.PeerId) bool {
	self.rcLock.Lock()
	defer self.rcLock.Unlock()
	rs, ok := self.recurCache[key]
	if !ok {
		return false
	}

	return rs.TryVisit(id)
}

func (self *Discovery) reSearchWithLock(target common.PeerId, entry *RecurSearch) {
	betters := self.dht.BetterPeers(target, dht.AlphaValue)
	for _, b := range betters {
		if b.ID == target {
			go func(ch chan<- string) {
				select {
				case entry.AddrChan <- b.Address:
				default:
				}
				close(ch)
			}(entry.AddrChan)
			delete(self.recurCache, target)
			return
		}

		// send request to better neighbors
		if entry.TryVisit(b.ID) {
			go self.net.SendTo(b.ID, msgpack.NewRecursiveFindNodeReq(target))
		}
	}
}
