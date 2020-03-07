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
package netserver

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/ledger"
	"github.com/ontio/ontology/p2pserver/common"
	"github.com/ontio/ontology/p2pserver/handshake"
	"github.com/ontio/ontology/p2pserver/message/msg_pack"
	"github.com/ontio/ontology/p2pserver/peer"
)

func handshakeClient(netServer *NetServer, conn net.Conn) error {
	version := msgpack.NewVersion(netServer, ledger.DefLedger.GetCurrentBlockHeight())
	remotePeer, err := handshake.HandshakeClient(version, netServer.GetKadKeyId(), conn)
	if err != nil {
		log.Warn(err)
		return err
	}

	if err = isHandWithSelf(netServer, remotePeer); err != nil {
		return err
	}

	kid := remotePeer.GetKId()
	// Obsolete node
	err = removeOldPeer(netServer, kid.ToUint64(), conn.RemoteAddr().String())
	if err != nil {
		return err
	}

	if !netServer.UpdateDHT(kid) {
		err := fmt.Errorf("[HandshakeClient] UpdateDHT failed, kadId: %s", kid.ToHexString())
		log.Error(err)
		return err
	}

	remoteAddr := remotePeer.GetAddr()
	remotePeer.AttachChan(netServer.NetChan)
	netServer.AddOutConnRecord(remoteAddr)
	netServer.AddPeerAddress(remoteAddr, remotePeer)
	netServer.AddNbrNode(remotePeer)
	log.Infof("remotePeer.GetId():%d,addr: %s, link id: %d", remotePeer.GetID(), remoteAddr, remotePeer.Link.GetID())
	go remotePeer.Link.Rx()
	remotePeer.SetState(common.ESTABLISH)

	if netServer.pid != nil {
		input := &common.AppendPeerID{
			ID: kid.ToUint64(),
		}
		netServer.pid.Tell(input)
	}

	return nil
}

func isHandWithSelf(netServer *NetServer, remotePeer *peer.Peer) error {
	remoteAddr := remotePeer.GetAddr()
	addrIp, err := common.ParseIPAddr(remoteAddr)
	if err != nil {
		log.Warn(err)
		return err
	}
	nodeAddr := addrIp + ":" + strconv.Itoa(int(remotePeer.GetPort()))
	if remotePeer.GetKId() == netServer.GetKId() {
		log.Warn("[createPeer]the node handshake with itself:", remoteAddr)
		netServer.SetOwnAddress(nodeAddr)
		return fmt.Errorf("[createPeer]the node handshake with itself: %s", remoteAddr)
	}
	return nil
}

func handshakeServer(netServer *NetServer, conn net.Conn) error {
	ver := msgpack.NewVersion(netServer, ledger.DefLedger.GetCurrentBlockHeight())
	remotePeer, err := handshake.HandshakeServer(ver, netServer.GetKadKeyId(), conn)
	if err != nil {
		log.Info(err)
		return err
	}

	if err = isHandWithSelf(netServer, remotePeer); err != nil {
		return err
	}

	// Obsolete node
	kid := remotePeer.GetKId()
	err = removeOldPeer(netServer, kid.ToUint64(), conn.RemoteAddr().String())
	if err != nil {
		return err
	}

	netServer.dht.Update(kid)

	remotePeer.AttachChan(netServer.NetChan)
	addr := conn.RemoteAddr().String()
	netServer.AddNbrNode(remotePeer)
	netServer.AddInConnRecord(addr)
	netServer.AddPeerAddress(addr, remotePeer)

	go remotePeer.Link.Rx()
	if netServer.pid != nil {
		input := &common.AppendPeerID{
			ID: kid.ToUint64(),
		}
		netServer.pid.Tell(input)
	}
	return nil
}

func checkReservedPeers(remoteAddr string) error {
	if config.DefConfig.P2PNode.ReservedPeersOnly && len(config.DefConfig.P2PNode.ReservedCfg.ReservedPeers) > 0 {
		found := false
		for _, addr := range config.DefConfig.P2PNode.ReservedCfg.ReservedPeers {
			if strings.HasPrefix(remoteAddr, addr) {
				log.Debug("[createPeer]peer in reserved list", remoteAddr)
				found = true
				break
			}
		}
		if !found {
			log.Debug("[createPeer]peer not in reserved list,close", remoteAddr)
			return fmt.Errorf("the remote addr: %s not in ReservedPeers", remoteAddr)
		}
	}

	return nil
}

func removeOldPeer(p2p *NetServer, pid uint64, remoteAddr string) error {
	p := p2p.GetPeer(pid)
	if p != nil {
		ipOld, err := common.ParseIPAddr(p.GetAddr())
		if err != nil {
			log.Warn("[createPeer]exist peer %d ip format is wrong %s", pid, p.GetAddr())
			return fmt.Errorf("[createPeer]exist peer %d ip format is wrong %s", pid, p.GetAddr())
		}
		ipNew, err := common.ParseIPAddr(remoteAddr)
		if err != nil {
			log.Warn("[createPeer]connecting peer %d ip format is wrong %s, close", pid, remoteAddr)
			return fmt.Errorf("[createPeer]connecting peer %d ip format is wrong %s, close", pid, remoteAddr)
		}
		if ipNew == ipOld {
			//same id and same ip
			n, delOK := p2p.DelNbrNode(pid)
			if delOK {
				log.Infof("[createPeer]peer reconnect %d", pid, remoteAddr)
				// Close the connection and release the node source
				n.Close()
				if p2p.pid != nil {
					input := &common.RemovePeerID{
						ID: pid,
					}
					p2p.pid.Tell(input)
				}
			}
		} else {
			err := fmt.Errorf("[createPeer]same peer id from different addr: %s, %s close latest one", ipOld, ipNew)
			log.Warn(err)
			return err
		}
	}

	return nil
}
