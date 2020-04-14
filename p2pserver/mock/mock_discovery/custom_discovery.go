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

package mock_discovery

import (
	"time"

	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/p2pserver/message/types"
	msgTypes "github.com/ontio/ontology/p2pserver/message/types"
	p2p "github.com/ontio/ontology/p2pserver/net/protocol"
	"github.com/ontio/ontology/p2pserver/protocols/bootstrap"
	"github.com/ontio/ontology/p2pserver/protocols/discovery"
)

type DiscoveryProtocol struct {
	MaskPeers       []string
	RefleshInterval time.Duration
	seeds           []string

	discovery *discovery.Discovery
	bootstrap *bootstrap.BootstrapService
}

func NewDiscoveryProtocol(seeds []string, maskPeers []string) *DiscoveryProtocol {
	return &DiscoveryProtocol{seeds: seeds, MaskPeers: maskPeers}
}

func (self *DiscoveryProtocol) start(net p2p.P2P) {
	self.discovery = discovery.NewDiscovery(net, self.MaskPeers, self.RefleshInterval)
	self.bootstrap = bootstrap.NewBootstrapService(net, self.seeds)
	go self.discovery.Start()
	go self.bootstrap.Start()
}

func (self *DiscoveryProtocol) HandleSystemMessage(net p2p.P2P, msg p2p.SystemMessage) {
	switch m := msg.(type) {
	case p2p.NetworkStart:
		self.start(net)
	case p2p.PeerConnected:
		self.discovery.OnAddPeer(m.Info)
		self.bootstrap.OnAddPeer(m.Info)
	case p2p.PeerDisConnected:
		self.discovery.OnDelPeer(m.Info)
		self.bootstrap.OnDelPeer(m.Info)
	case p2p.NetworkStop:
		self.discovery.Stop()
		self.bootstrap.Stop()
	}
}

func (self *DiscoveryProtocol) HandlePeerMessage(ctx *p2p.Context, msg msgTypes.Message) {
	log.Trace("[p2p]receive message", ctx.Sender().GetAddr(), ctx.Sender().GetID())
	switch m := msg.(type) {
	case *types.AddrReq:
		self.discovery.AddrReqHandle(ctx)
	case *msgTypes.FindNodeResp:
		self.discovery.FindNodeResponseHandle(ctx, m)
	case *msgTypes.FindNodeReq:
		self.discovery.FindNodeHandle(ctx, m)
	default:
		msgType := msg.CmdType()
		log.Warn("unknown message handler for the msg: ", msgType)
	}
}

func (self *DiscoveryProtocol) Discovery() *discovery.Discovery {
	return self.discovery
}
