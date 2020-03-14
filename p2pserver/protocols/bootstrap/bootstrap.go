package bootstrap

import (
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/p2pserver/common"
	msgpack "github.com/ontio/ontology/p2pserver/message/msg_pack"
	"github.com/ontio/ontology/p2pserver/message/types"
	p2p "github.com/ontio/ontology/p2pserver/net/protocol"
	"github.com/ontio/ontology/p2pserver/peer"
	"math/rand"
	"net"
	"strconv"
	"time"
)

const activeConnect = 4 // when connection num less than this value, we connect seeds node actively.

type BootstrapService struct {
	seeds     []string
	connected uint
	net       p2p.P2P
	quit      chan bool
}

func NewBootstrapService(net p2p.P2P, seeds []string) *BootstrapService {
	return &BootstrapService{
		seeds: seeds,
		net:   net,
		quit:  make(chan bool),
	}
}

func (self *BootstrapService) Start() {
	go self.connectSeedService()
}

func (self *BootstrapService) Stop() {
	close(self.quit)
}

func (self *BootstrapService) OnAddPeer(info *peer.PeerInfo) {
	self.connected += 1
}

func (self *BootstrapService) OnDelPeer(info *peer.PeerInfo) {
	self.connected -= 1
}

//connectSeedService make sure seed peer be connected
func (self *BootstrapService) connectSeedService() {
	t := time.NewTimer(time.Second * common.CONN_MONITOR)
	for {
		select {
		case <-t.C:
			self.connectSeeds()
			t.Stop()
			if self.connected >= activeConnect {
				t.Reset(time.Second * time.Duration(10*common.CONN_MONITOR))
			} else {
				t.Reset(time.Second * common.CONN_MONITOR)
			}
		case <-self.quit:
			t.Stop()
			return
		}
	}
}

//connectSeeds connect the seeds in seedlist and call for nbr list
func (self *BootstrapService) connectSeeds() {
	seedNodes := make([]string, 0)
	for _, n := range self.seeds {
		ip, err := common.ParseIPAddr(n)
		if err != nil {
			log.Warnf("[p2p]seed peer %s address format is wrong", n)
			continue
		}
		ns, err := net.LookupHost(ip)
		if err != nil {
			log.Warnf("[p2p]resolve err: %s", err.Error())
			continue
		}
		port, err := common.ParseIPPort(n)
		if err != nil {
			log.Warnf("[p2p]seed peer %s address format is wrong", n)
			continue
		}
		seedNodes = append(seedNodes, ns[0]+port)
	}

	connPeers := make(map[string]*peer.Peer)
	np := self.net.GetNp()
	np.Lock()
	for _, tn := range np.List {
		ipAddr, _ := tn.GetAddr16()
		ip := net.IP(ipAddr[:])
		addrString := ip.To16().String() + ":" + strconv.Itoa(int(tn.GetPort()))
		if tn.GetState() == common.ESTABLISH {
			connPeers[addrString] = tn
		}
	}
	np.Unlock()

	seedConnList := make([]*peer.Peer, 0)
	seedDisconn := make([]string, 0)
	isSeed := false
	for _, nodeAddr := range seedNodes {
		if p, ok := connPeers[nodeAddr]; ok {
			seedConnList = append(seedConnList, p)
		} else {
			seedDisconn = append(seedDisconn, nodeAddr)
		}

		if self.net.IsOwnAddress(nodeAddr) {
			isSeed = true
		}
	}

	if len(seedConnList) > 0 {
		rand.Seed(time.Now().UnixNano())
		// close NewAddrReq
		index := rand.Intn(len(seedConnList))
		self.reqNbrList(seedConnList[index])
		if isSeed && len(seedDisconn) > 0 {
			index := rand.Intn(len(seedDisconn))
			go self.net.Connect(seedDisconn[index])
		}
	} else { //not found
		for _, nodeAddr := range seedNodes {
			go self.net.Connect(nodeAddr)
		}
	}
}

func (this *BootstrapService) reqNbrList(p *peer.Peer) {
	id := p.GetID()
	var msg types.Message
	if id.IsPseudoPeerId() {
		msg = msgpack.NewAddrReq()
	} else {
		msg = msgpack.NewFindNodeReq(this.net.GetID())
	}

	go this.net.Send(p, msg)
}