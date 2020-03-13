package heatbeat

import (
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/ledger"
	"github.com/ontio/ontology/p2pserver/common"
	"github.com/ontio/ontology/p2pserver/message/msg_pack"
	"github.com/ontio/ontology/p2pserver/net/protocol"
	"github.com/ontio/ontology/p2pserver/peer"
	"time"
)

type HeartBeat struct {
	net    p2p.P2P
	id     common.PeerId
	quit   chan bool
	ledger *ledger.Ledger //ledger
}

func NewHeartBeat(net p2p.P2P, ld *ledger.Ledger) *HeartBeat {
	return &HeartBeat{
		id:     net.GetID(),
		net:    net,
		quit:   make(chan bool),
		ledger: ld,
	}
}

func (self *HeartBeat) Start() {
	go self.heartBeatService()
}

func (this *HeartBeat) heartBeatService() {
	var periodTime uint
	periodTime = config.DEFAULT_GEN_BLOCK_TIME / common.UPDATE_RATE_PER_BLOCK
	t := time.NewTicker(time.Second * (time.Duration(periodTime)))

	for {
		select {
		case <-t.C:
			this.ping()
			this.timeout()
		case <-this.quit:
			t.Stop()
			return
		}
	}
}

func (this *HeartBeat) ping() {
	peers := this.net.GetNeighbors()
	this.pingTo(peers)
}

//pings send pkgs to get pong msg from others
func (this *HeartBeat) pingTo(peers []*peer.Peer) {
	for _, p := range peers {
		if p.GetState() == common.ESTABLISH {
			height := this.ledger.GetCurrentBlockHeight()
			ping := msgpack.NewPingMsg(uint64(height))
			go this.net.Send(p, ping)
		}
	}
}

//timeout trace whether some peer be long time no response
func (this *HeartBeat) timeout() {
	peers := this.net.GetNeighbors()
	var periodTime uint
	periodTime = config.DEFAULT_GEN_BLOCK_TIME / common.UPDATE_RATE_PER_BLOCK
	for _, p := range peers {
		if p.GetState() == common.ESTABLISH {
			t := p.GetContactTime()
			if t.Before(time.Now().Add(-1 * time.Second *
				time.Duration(periodTime) * common.KEEPALIVE_TIMEOUT)) {
				log.Warnf("[p2p]keep alive timeout!!!lost remote peer %d - %s from %s", p.GetID(), p.Link.GetAddr(), t.String())
				p.Close()
			}
		}
	}
}
