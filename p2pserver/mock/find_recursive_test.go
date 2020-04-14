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
	"context"
	"testing"
	"time"

	"github.com/ontio/ontology/p2pserver/net/netserver"
	"github.com/stretchr/testify/require"
)

func TestRecursiveSearch(t *testing.T) {
	a := require.New(t)

	// topo
	//  a -----------> b
	//                | \
	//                c  \
	//                    d
	//                     \
	//                      e
	nw := NewNetwork()
	nodeb := NewDiscoveryNode(nil, nw)
	nodebAddr := nodeb.GetHostInfo().Addr

	seedb := []string{nodebAddr}
	nodea := NewDiscoveryNode(seedb, nw)
	nodec := NewDiscoveryNode(seedb, nw)
	noded := NewDiscoveryNode(seedb, nw)
	nodee := NewDiscoveryNode([]string{noded.GetHostInfo().Addr}, nw)

	lst := []*netserver.NetServer{nodea, nodeb, nodec, noded, nodee}
	// allow connecting
	for i := 0; i < len(lst); i++ {
		for j := i + 1; j < len(lst); j++ {
			nw.AllowConnect(lst[i].GetHostInfo().Id, lst[j].GetHostInfo().Id)
		}
	}
	// start
	for _, n := range lst {
		go n.Start()
	}
	// make sure node started and attach the dht struct
	time.Sleep(time.Second * 3)
	a.Equal(nodea.GetConnectionCnt(), uint32(4), "fail")

	// comment dht refresh and find myself to test
	ctx, cf := context.WithTimeout(context.Background(), time.Second*3)
	defer cf()
	ret, err := nodea.CustomFindPeerAddress(ctx, nodee.GetHostInfo().Id)
	a.Nil(err, "find fail")
	a.Equal(nodee.GetHostInfo().Addr, ret, "find fail")
}
