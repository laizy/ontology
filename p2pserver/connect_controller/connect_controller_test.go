package connect_controller

import (
	"github.com/ontio/ontology/p2pserver/dht/kbucket"
	"github.com/ontio/ontology/p2pserver/handshake"
	"github.com/ontio/ontology/p2pserver/peer"
	"github.com/stretchr/testify/assert"
	"net"
	"sync"
	"testing"
)

func init() {
	kbucket.Difficulty = 1
}

type Transport struct {
	c      <-chan net.Conn
	dialer Dialer
	t      *testing.T
}

func NewTransport(t *testing.T) *Transport {
	listener, err := net.Listen("tcp", ":20338")
	assert.Nil(t, err)
	c := make(chan net.Conn)
	go func() {
		for {
			conn, err := listener.Accept()
			assert.Nil(t, err)
			c <- conn
		}
	}()

	return &Transport{
		c:      c,
		t:      t,
		dialer: &noTlsDialer{},
	}
}

func (self *Transport) Pipe() (net.Conn, net.Conn) {
	client, err := self.dialer.Dial("127.0.0.1:20338")
	assert.Nil(self.t, err)

	server := <-self.c

	return client, server
}

type Node struct {
	*ConnectController
	Info *peer.PeerInfo
	Key  *kbucket.KadKeyId
}

func NewNode(option ConnCtrlOption) Node {
	key := kbucket.RandKadKeyId()
	info := &peer.PeerInfo{
		Id:          key.Id,
		SoftVersion: "v1.9.0-beta",
	}

	return Node{
		ConnectController: NewConnectController(info, key, option),
		Info:              info,
		Key:               key,
	}
}

func TestConnectController_AcceptConnect_MaxInBound(t *testing.T) {
	trans := NewTransport(t)
	maxInboud := 5
	server := NewNode(NewConnCtrlOption().MaxInBound(uint(maxInboud)))
	client := NewNode(NewConnCtrlOption().MaxOutBound(uint(maxInboud * 2)))

	var clientConns []net.Conn
	wg := &sync.WaitGroup{}
	wg.Add(maxInboud * 2)
	for i := 0; i < maxInboud*2; i++ {
		conn1, conn2 := trans.Pipe()
		go func(i int) {
			defer wg.Done()
			_, err := handshake.HandshakeClient(client.peerInfo, client.Key, conn1)
			if i < int(maxInboud) {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		}(i)

		info, conn, err := server.AcceptConnect(conn2)
		if i >= int(maxInboud) {
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "reach max limit")
			continue
		}
		assert.Nil(t, err)
		assert.Equal(t, info, client.Info)

		assert.Equal(t, server.inoutbounds[INBOUND_INDEX].Size(), i+1)
		assert.Equal(t, server.connecting.Size(), 0)
		clientConns = append(clientConns, conn)
	}

	for _, conn := range clientConns {
		_ = conn.Close()
	}

	assert.Equal(t, server.inoutbounds[INBOUND_INDEX].Size(), 0)
}
