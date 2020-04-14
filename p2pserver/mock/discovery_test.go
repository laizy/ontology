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
	"fmt"
	"testing"
	"time"

	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/p2pserver/common"
	"github.com/ontio/ontology/p2pserver/mock/mock_discovery"
	"github.com/ontio/ontology/p2pserver/net/netserver"
	"github.com/ontio/ontology/p2pserver/peer"
	"github.com/stretchr/testify/assert"
)

func init() {
	common.Difficulty = 1
}

func TestDiscoveryNode(t *testing.T) {
	N := 5
	net := NewNetwork()
	seedNode := NewDiscoveryNode(nil, net)
	var nodes []*netserver.NetServer
	go seedNode.Start()
	seedAddr := seedNode.GetHostInfo().Addr
	log.Errorf("seed addr: %s", seedAddr)
	for i := 0; i < N; i++ {
		node := NewDiscoveryNode([]string{seedAddr}, net)
		net.AllowConnect(seedNode.GetHostInfo().Id, node.GetHostInfo().Id)
		go node.Start()
		nodes = append(nodes, node)
	}

	time.Sleep(time.Second * 1)
	assert.Equal(t, seedNode.GetConnectionCnt(), uint32(N))
	for i, node := range nodes {
		assert.Equal(t, node.GetConnectionCnt(), uint32(1), fmt.Sprintf("node %d", i))
	}

	log.Info("start allow node connection")
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			net.AllowConnect(nodes[i].GetHostInfo().Id, nodes[j].GetHostInfo().Id)
		}
	}
	time.Sleep(time.Second * 1)
	for i, node := range nodes {
		assert.True(t, node.GetConnectionCnt() > uint32(N/3), fmt.Sprintf("node %d", i))
	}
}

func NewDiscoveryNode(seeds []string, net Network) *netserver.NetServer {
	seedId := common.RandPeerKeyId()
	info := peer.NewPeerInfo(seedId.Id, 0, 0, true, 0,
		0, 0, "1.10", "")

	dis := mock_discovery.NewDiscoveryProtocol(seeds, nil)
	dis.RefleshInterval = time.Millisecond * 10

	return NewNode(seedId, info, dis, net, nil)
}
