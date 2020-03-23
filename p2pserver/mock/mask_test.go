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
package mock

import (
	"strings"
	"testing"
	"time"

	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/p2pserver/common"
	"github.com/ontio/ontology/p2pserver/net/netserver"
	"github.com/ontio/ontology/p2pserver/peer"
	"github.com/stretchr/testify/assert"
)

func TestMask(t *testing.T) {
	nw := NewNetwork()

	// topo:
	//      seed
	//    /     \
	//  normal   resv
	seed := NewTestNode(nw, nil, nil, nil)
	seedAddr := seed.GetHostInfo().Addr
	seedIP := strings.Split(seedAddr, ":")[0]
	log.Errorf("seed ip: %s", seedIP)

	// to make sure resvnode will not active connect to normal node
	nodeWithResv := NewTestNode(nw, []string{seedAddr}, []string{seedIP}, []string{seedIP})
	nodeNormal := NewTestNode(nw, []string{seedAddr}, nil, nil)

	// normal will not reach nodeWithResv, dht won't give it nodeWithResv IP, it's beed masked
	nw.AllowConnect(seed.GetHostInfo().Id, nodeWithResv.GetHostInfo().Id)
	nw.AllowConnect(seed.GetHostInfo().Id, nodeNormal.GetHostInfo().Id)
	nw.AllowConnect(nodeNormal.GetHostInfo().Id, nodeWithResv.GetHostInfo().Id)

	go seed.Start()
	go nodeWithResv.Start()
	go nodeNormal.Start()

	time.Sleep(time.Second * 2)
	assert.Equal(t, uint32(2), seed.GetConnectionCnt())
	assert.Equal(t, uint32(1), nodeNormal.GetConnectionCnt())
	assert.Equal(t, uint32(1), nodeWithResv.GetConnectionCnt())
}

func NewTestNode(nw Network, seeds, maskPeers, resv []string) *netserver.NetServer {
	seedId := common.RandPeerKeyId()
	info := peer.NewPeerInfo(seedId.Id, 0, 0, true, 0,
		0, 0, "1.10", "")

	dis := NewDiscoveryProtocol(seeds, maskPeers)
	dis.RefleshInterval = time.Millisecond * 1

	return NewNode(seedId, info, dis, nw, resv)
}
