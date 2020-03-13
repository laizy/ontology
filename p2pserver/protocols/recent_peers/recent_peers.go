package recent_peers

import (
	"encoding/json"
	common2 "github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/p2pserver/common"
	"github.com/ontio/ontology/p2pserver/net/protocol"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type PersistRecentPeerService struct {
	net             p2p.P2P
	quit            chan bool
	recentPeers     map[uint32][]*RecentPeer
	lock            sync.RWMutex
}

func (this *PersistRecentPeerService) IsHave(addr string) bool {
	this.lock.RLock()
	defer this.lock.RUnlock()
	netID := config.DefConfig.P2PNode.NetworkMagic
	for i := 0; i < len(this.recentPeers[netID]); i++ {
		if this.recentPeers[netID][i].Addr == addr {
			return true
		}
	}
	return false
}

func (this *PersistRecentPeerService) AddNodeAddr(addr string) {
	this.lock.Lock()
	this.recentPeers[config.DefConfig.P2PNode.NetworkId] = append(this.recentPeers[config.DefConfig.P2PNode.NetworkId],
		&RecentPeer{
			Addr:  addr,
			Birth: time.Now().Unix(),
		})
	this.lock.Unlock()
}

func (this *PersistRecentPeerService) DelNodeAddr(addr string) {
	this.lock.Lock()
	netID := config.DefConfig.P2PNode.NetworkMagic
	for i := 0; i < len(this.recentPeers[netID]); i++ {
		if this.recentPeers[netID][i].Addr == addr {
			this.recentPeers[netID] = append(this.recentPeers[netID][:i], this.recentPeers[netID][i+1:]...)
		}
	}
	this.lock.Unlock()
}

type RecentPeer struct {
	Addr  string
	Birth int64
}

func (this *PersistRecentPeerService) saveToFile() {
	temp := make(map[uint32][]string)
	for networkId, rps := range this.recentPeers {
		temp[networkId] = make([]string, 0)
		for _, rp := range rps {
			elapse := time.Now().Unix() - rp.Birth
			if elapse > config.DefConfig.P2PNode.RecentPeerElapse {
				temp[networkId] = append(temp[networkId], rp.Addr)
			}
		}
	}
	buf, err := json.Marshal(temp)
	if err != nil {
		log.Warn("[p2p]package recent peer fail: ", err)
		return
	}
	err = ioutil.WriteFile(common.RECENT_FILE_NAME, buf, os.ModePerm)
	if err != nil {
		log.Warn("[p2p]write recent peer fail: ", err)
	}
}

func NewPersistRecentPeerService(net p2p.P2P) *PersistRecentPeerService {
	return &PersistRecentPeerService{
		net:  net,
		quit: make(chan bool),
	}
}

func (this *PersistRecentPeerService) LoadRecentPeers() {
	this.recentPeers = make(map[uint32][]*RecentPeer)
	if common2.FileExisted(common.RECENT_FILE_NAME) {
		buf, err := ioutil.ReadFile(common.RECENT_FILE_NAME)
		if err != nil {
			log.Warn("[p2p]read %s fail:%s, connect recent peers cancel", common.RECENT_FILE_NAME, err.Error())
			return
		}

		temp := make(map[uint32][]string)
		err = json.Unmarshal(buf, &temp)
		if err != nil {
			log.Warn("[p2p]parse recent peer file fail: ", err)
			return
		}
		for networkId, addrs := range temp {
			for _, addr := range addrs {
				this.recentPeers[networkId] = append(this.recentPeers[networkId], &RecentPeer{
					Addr:  addr,
					Birth: time.Now().Unix(),
				})
			}
		}
	}
}

//tryRecentPeers try connect recent contact peer when service start
func (this *PersistRecentPeerService) TryRecentPeers() {
	netID := config.DefConfig.P2PNode.NetworkMagic
	if len(this.recentPeers[netID]) > 0 {
		log.Info("[p2p] try to connect recent peer")
	}
	for _, v := range this.recentPeers[netID] {
		go this.net.Connect(v.Addr)
	}
}

//syncUpRecentPeers sync up recent peers periodically
func (this *PersistRecentPeerService) SyncUpRecentPeers() {
	periodTime := common.RECENT_TIMEOUT
	t := time.NewTicker(time.Second * (time.Duration(periodTime)))
	for {
		select {
		case <-t.C:
			this.saveToFile()
		case <-this.quit:
			t.Stop()
			return
		}
	}
}
