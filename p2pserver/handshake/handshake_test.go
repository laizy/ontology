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
package handshake

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ontio/ontology/p2pserver/dht/kbucket"
	"github.com/ontio/ontology/p2pserver/message/types"
	"github.com/ontio/ontology/p2pserver/peer"
	"github.com/stretchr/testify/assert"
)

func init() {
	kbucket.Difficulty = 1
	HANDSHAKE_DURATION = 1 * time.Second
}

type Node struct {
	Id   *kbucket.KadKeyId
	Info *peer.PeerInfo
	Conn net.Conn
}

func NewNode(conn net.Conn) Node {
	node := Node{
		Id:   kbucket.RandKadKeyId(),
		Info: &peer.PeerInfo{},
		Conn: conn,
	}
	node.Info.Id = node.Id.Id
	node.Info.SoftVersion = "v1.9.0-beta"

	return node
}

func NewPair() (client Node, server Node) {
	c, s := net.Pipe()

	client = NewNode(c)
	server = NewNode(s)
	return
}

func TestHandshakeNormal(t *testing.T) {
	client, server := NewPair()

	wg := sync.WaitGroup{}
	wg.Add(2)
	result := make([]struct {
		info [2]*peer.PeerInfo
		err  error
	}, 2)
	go func() {
		info, err := HandshakeClient(client.Info, client.Id, client.Conn)
		result[0].err = err
		result[0].info = [2]*peer.PeerInfo{info, server.Info}
		wg.Done()
	}()
	go func() {
		info, err := HandshakeServer(server.Info, server.Id, server.Conn)
		result[1].err = err
		result[1].info = [2]*peer.PeerInfo{info, client.Info}
		wg.Done()
	}()
	wg.Wait()

	for _, res := range result {
		assert.Nil(t, res.err)
		assert.Equal(t, res.info[0], res.info[1])
	}
}

func TestHandshakeTimeout(t *testing.T) {
	client, _ := NewPair()

	_, err := HandshakeClient(client.Info, client.Id, client.Conn)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "deadline exceeded")
}

func TestHandshakeWrongMsg(t *testing.T) {
	client, server := NewPair()
	go func() {
		err := sendMsg(client.Conn, &types.Addr{})
		assert.Nil(t, err)
	}()

	_, err := HandshakeServer(server.Info, server.Id, server.Conn)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "expected version message")
}

func TestVersion(t *testing.T) {
	assert.True(t, supportDHT("1.9.0"))
	assert.True(t, supportDHT("v1.10.0"))
	assert.True(t, supportDHT("v1.10"))
	assert.True(t, supportDHT("v2.0"))
	assert.True(t, supportDHT("v1.9.0"))
	assert.True(t, supportDHT("1.9.0-beta"))
	assert.True(t, supportDHT("v1.9.0-beta"))
	assert.True(t, supportDHT("1.9.0-beta-9"))
	assert.True(t, supportDHT("1.9.0-beta-9-geeaeewwf"))

	assert.False(t, supportDHT("1.9.0-alpha"))
	assert.False(t, supportDHT("1.8.0-beta-9-geeaeewwf"))
	assert.False(t, supportDHT("1.8.0"))
}
