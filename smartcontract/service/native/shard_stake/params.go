package shard_stake

import (
	"fmt"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/serialization"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/smartcontract/service/native/utils"
	"io"
)

type PeerInitStakeParam struct {
	ShardId        types.ShardID
	StakeAssetAddr common.Address
	PeerOwner      common.Address
	PeerPubKey     string
	StakeAmount    uint64
}

func (this *PeerInitStakeParam) Serialize(w io.Writer) error {
	err := utils.WriteVarUint(w, this.ShardId.ToUint64())
	if err != nil {
		return fmt.Errorf("serialize: write shard id failed, err: %s", err)
	}
	err = utils.WriteAddress(w, this.StakeAssetAddr)
	if err != nil {
		return fmt.Errorf("serialize: write stake asset addr failed, err: %s", err)
	}
	err = utils.WriteAddress(w, this.PeerOwner)
	if err != nil {
		return fmt.Errorf("serialize: write peer owner failed, err: %s", err)
	}
	err = serialization.WriteString(w, this.PeerPubKey)
	if err != nil {
		return fmt.Errorf("serialize: write peer pub key failed, err: %s", err)
	}
	err = utils.WriteVarUint(w, this.StakeAmount)
	if err != nil {
		return fmt.Errorf("serialize: write stake amount failed, err: %s", err)
	}
	return nil
}

func (this *PeerInitStakeParam) Deserialize(r io.Reader) error {
	id, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read shard id failed, err: %s", err)
	}
	this.ShardId, err = types.NewShardID(id)
	if err != nil {
		return fmt.Errorf("deserialize: generate shard id failed, err: %s", err)
	}
	stakeAssetAddr, err := utils.ReadAddress(r)
	if err != nil {
		return fmt.Errorf("deserialize: read stake asset addr failed, err: %s", err)
	}
	this.StakeAssetAddr = stakeAssetAddr
	owner, err := utils.ReadAddress(r)
	if err != nil {
		return fmt.Errorf("deserialize: read peer owner failed, err: %s", err)
	}
	this.PeerOwner = owner
	peer, err := serialization.ReadString(r)
	if err != nil {
		return fmt.Errorf("deserialize: read peer pub key failed, err: %s", err)
	}
	this.PeerPubKey = peer
	amount, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read amount failed, err: %s", err)
	}
	this.StakeAmount = amount
	return nil
}

type UnfreezeFromShardParam struct {
	ShardId    types.ShardID
	Address    common.Address
	PeerPubKey []string
	Amount     []uint64
}

func (this *UnfreezeFromShardParam) Serialize(w io.Writer) error {
	err := utils.WriteVarUint(w, this.ShardId.ToUint64())
	if err != nil {
		return fmt.Errorf("serialize: write shard id failed, err: %s", err)
	}
	err = utils.WriteAddress(w, this.Address)
	if err != nil {
		return fmt.Errorf("serialize: write addr failed, err: %s", err)
	}
	if len(this.PeerPubKey) != len(this.Amount) {
		return fmt.Errorf("serialize: peer length not equals amount length")
	}
	err = utils.WriteVarUint(w, uint64(len(this.PeerPubKey)))
	if err != nil {
		return fmt.Errorf("serialize: write peer len failed, err: %s", err)
	}
	for index, peer := range this.PeerPubKey {
		err = serialization.WriteString(w, peer)
		if err != nil {
			return fmt.Errorf("serialize: write peer %s failed, index %d, err: %s", peer, index, err)
		}
		err = utils.WriteVarUint(w, this.Amount[index])
		if err != nil {
			return fmt.Errorf("serialize: write amount failed, index %d, err: %s", index, err)
		}
	}
	return nil
}

func (this *UnfreezeFromShardParam) Deserialize(r io.Reader) error {
	id, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read shard id failed, err: %s", err)
	}
	this.ShardId, err = types.NewShardID(id)
	if err != nil {
		return fmt.Errorf("deserialize: generate shard id failed, err: %s", err)
	}
	addr, err := utils.ReadAddress(r)
	if err != nil {
		return fmt.Errorf("deserialize: read addr failed, err: %s", err)
	}
	this.Address = addr
	peerNum, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read addr failed, err: %s", err)
	}
	peers := make([]string, peerNum)
	amount := make([]uint64, peerNum)
	for i := uint64(0); i < peerNum; i++ {
		peer, err := serialization.ReadString(r)
		if err != nil {
			return fmt.Errorf("deserialize: read peer failed, index %d, err: %s", i, err)
		}
		num, err := utils.ReadVarUint(r)
		if err != nil {
			return fmt.Errorf("deserialize: read amount failed, index %d, err: %s", i, err)
		}
		peers[i] = peer
		amount[i] = num
	}
	this.PeerPubKey = peers
	this.Amount = amount
	return nil
}

type WithdrawStakeAssetParam struct {
	ShardId types.ShardID
	User    common.Address
}

func (this *WithdrawStakeAssetParam) Serialize(w io.Writer) error {
	if err := utils.WriteVarUint(w, this.ShardId.ToUint64()); err != nil {
		return fmt.Errorf("serialize: write shard id failed, err: %s", err)
	}
	if err := utils.WriteAddress(w, this.User); err != nil {
		return fmt.Errorf("serialize: write address failed, err: %s", err)
	}
	return nil
}

func (this *WithdrawStakeAssetParam) Deserialize(r io.Reader) error {
	id, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read shard id failed, err: %s", err)
	}
	this.ShardId, err = types.NewShardID(id)
	if err != nil {
		return fmt.Errorf("deserialize: generate shard id failed, err: %s", err)
	}
	addr, err := utils.ReadAddress(r)
	if err != nil {
		return fmt.Errorf("deserialize: read address failed, err: %s", err)
	}
	this.User = addr
	return nil
}

type WithdrawFeeParam struct {
	ShardId types.ShardID
	User    common.Address
}

func (this *WithdrawFeeParam) Serialize(w io.Writer) error {
	if err := utils.WriteVarUint(w, this.ShardId.ToUint64()); err != nil {
		return fmt.Errorf("serialize: write shard id failed, err: %s", err)
	}
	if err := utils.WriteAddress(w, this.User); err != nil {
		return fmt.Errorf("serialize: write address failed, err: %s", err)
	}
	return nil
}

func (this *WithdrawFeeParam) Deserialize(r io.Reader) error {
	id, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read shard id failed, err: %s", err)
	}
	this.ShardId, err = types.NewShardID(id)
	if err != nil {
		return fmt.Errorf("deserialize: generate shard id failed, err: %s", err)
	}
	addr, err := utils.ReadAddress(r)
	if err != nil {
		return fmt.Errorf("deserialize: read address failed, err: %s", err)
	}
	this.User = addr
	return nil
}

type CommitDposParam struct {
	ShardId    types.ShardID
	View       View
	Amount     []uint64
	PeerPubKey []keypair.PublicKey
}

func (this *CommitDposParam) Serialize(w io.Writer) error {
	err := utils.WriteVarUint(w, this.ShardId.ToUint64())
	if err != nil {
		return fmt.Errorf("serialize: write shard id failed, err: %s", err)
	}
	if len(this.PeerPubKey) != len(this.Amount) {
		return fmt.Errorf("serialize: peer length not equals amount length")
	}
	err = utils.WriteVarUint(w, uint64(len(this.PeerPubKey)))
	if err != nil {
		return fmt.Errorf("serialize: write peer len failed, err: %s", err)
	}
	for index, peer := range this.PeerPubKey {
		err = serialization.WriteVarBytes(w, keypair.SerializePublicKey(peer))
		if err != nil {
			return fmt.Errorf("serialize: write peer failed, index %d, err: %s", index, err)
		}
		err = utils.WriteVarUint(w, this.Amount[index])
		if err != nil {
			return fmt.Errorf("serialize: write amount failed, index %d, err: %s", index, err)
		}
	}
	return nil
}

func (this *CommitDposParam) Deserialize(r io.Reader) error {
	id, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read shard id failed, err: %s", err)
	}
	this.ShardId, err = types.NewShardID(id)
	if err != nil {
		return fmt.Errorf("deserialize: generate shard id failed, err: %s", err)
	}
	peerNum, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read peer len, err: %s", err)
	}
	peers := make([]keypair.PublicKey, peerNum)
	amount := make([]uint64, peerNum)
	for i := uint64(0); i < peerNum; i++ {
		pubKeyData, err := serialization.ReadVarBytes(r)
		if err != nil {
			return fmt.Errorf("deserialize: read peer failed, index %d, err: %s", i, err)
		}
		peer, err := keypair.DeserializePublicKey(pubKeyData)
		if err != nil {
			return fmt.Errorf("deserialize: deserialize pub key failed, index %d, err: %s", i, err)
		}
		num, err := utils.ReadVarUint(r)
		if err != nil {
			return fmt.Errorf("deserialize: read amount failed, index %d, err: %s", i, err)
		}
		peers[i] = peer
		amount[i] = num
	}
	this.PeerPubKey = peers
	this.Amount = amount
	return nil
}

type SetMinStakeParam struct {
	ShardId types.ShardID
	Amount  uint64
}

func (this *SetMinStakeParam) Serialize(w io.Writer) error {
	if err := utils.WriteVarUint(w, this.ShardId.ToUint64()); err != nil {
		return fmt.Errorf("serialize: write shard id failed, err: %s", err)
	}
	if err := utils.WriteVarUint(w, this.Amount); err != nil {
		return fmt.Errorf("serialize: write amount failed, err: %s", err)
	}
	return nil
}

func (this *SetMinStakeParam) Deserialize(r io.Reader) error {
	id, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read shard id failed, err: %s", err)
	}
	this.ShardId, err = types.NewShardID(id)
	if err != nil {
		return fmt.Errorf("deserialize: generate shard id failed, err: %s", err)
	}
	amount, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read amount failed, err: %s", err)
	}
	this.Amount = amount
	return nil
}

type UserStakeParam struct {
	ShardId    types.ShardID  `json:"shard_id"`
	User       common.Address `json:"user"`
	PeerPubKey []string       `json:"peer_pub_key"`
	Amount     []uint64       `json:"amount"`
}

func (this *UserStakeParam) Serialize(w io.Writer) error {
	err := utils.WriteVarUint(w, this.ShardId.ToUint64())
	if err != nil {
		return fmt.Errorf("serialze: write shard id failed, err: %s", err)
	}
	err = utils.WriteAddress(w, this.User)
	if err != nil {
		return fmt.Errorf("serialze: write addr failed, err: %s", err)
	}
	if len(this.PeerPubKey) != len(this.Amount) {
		return fmt.Errorf("serialize: peer pub key num not equals amount num")
	}
	err = utils.WriteVarUint(w, uint64(len(this.PeerPubKey)))
	if err != nil {
		return fmt.Errorf("serialize: write peer length failed, err: %s", err)
	}
	for index, peer := range this.PeerPubKey {
		err = serialization.WriteString(w, peer)
		if err != nil {
			return fmt.Errorf("serialize: write peer pub key failed, index %d, err: %s", index, err)
		}
		err = utils.WriteVarUint(w, this.Amount[index])
		if err != nil {
			return fmt.Errorf("serialize: write peer stake amount failed, index %d, err: %s", index, err)
		}
	}
	return nil
}

func (this *UserStakeParam) Deserialize(r io.Reader) error {
	id, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read shard id failed, err: %s", err)
	}
	this.ShardId, err = types.NewShardID(id)
	if err != nil {
		return fmt.Errorf("deserialize: generate shard id failed, err: %s", err)
	}
	user, err := utils.ReadAddress(r)
	if err != nil {
		return fmt.Errorf("deserialize: read addr failed, err: %s", err)
	}
	this.User = user
	peerNum, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: peer length failed, err: %s", err)
	}
	peers := make([]string, peerNum)
	amount := make([]uint64, peerNum)
	for i := uint64(0); i < peerNum; i++ {
		peer, err := serialization.ReadString(r)
		if err != nil {
			return fmt.Errorf("deserialize: read peer failed, index %d, err: %s", i, err)
		}
		num, err := utils.ReadVarUint(r)
		if err != nil {
			return fmt.Errorf("deserialize: read amount failed, index %d, err: %s", i, err)
		}
		peers[i] = peer
		amount[i] = num
	}
	this.PeerPubKey = peers
	this.Amount = amount
	return nil
}

type ChangeMaxAuthorizationParam struct {
	ShardId    types.ShardID
	User       common.Address
	PeerPubKey string
	Amount     uint64
}

func (this *ChangeMaxAuthorizationParam) Serialize(w io.Writer) error {
	if err := utils.WriteVarUint(w, this.ShardId.ToUint64()); err != nil {
		return fmt.Errorf("serialize: write shard id failed, err: %s", err)
	}
	if err := utils.WriteAddress(w, this.User); err != nil {
		return fmt.Errorf("serialize: write addr failed, err: %s", err)
	}
	if err := serialization.WriteString(w, this.PeerPubKey); err != nil {
		return fmt.Errorf("serialize: write pub key failed, err: %s", err)
	}
	if err := utils.WriteVarUint(w, this.Amount); err != nil {
		return fmt.Errorf("serialize: write amount failed, err: %s", err)
	}
	return nil
}

func (this *ChangeMaxAuthorizationParam) Deserialize(r io.Reader) error {
	id, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read shard id failed, err: %s", err)
	}
	this.ShardId, err = types.NewShardID(id)
	if err != nil {
		return fmt.Errorf("deserialize: generate shard id failed, err: %s", err)
	}
	user, err := utils.ReadAddress(r)
	if err != nil {
		return fmt.Errorf("deserialize: read addr failed, err: %s", err)
	}
	this.User = user
	pubKey, err := serialization.ReadString(r)
	if err != nil {
		return fmt.Errorf("deserialize: read pub key failed, err: %s", err)
	}
	this.PeerPubKey = pubKey
	amount, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read amount failed, err: %s", err)
	}
	this.Amount = amount
	return nil
}

type ChangeProportionParam struct {
	ShardId    types.ShardID
	User       common.Address
	PeerPubKey string
	Amount     uint64
}

func (this *ChangeProportionParam) Serialize(w io.Writer) error {
	if err := utils.WriteVarUint(w, this.ShardId.ToUint64()); err != nil {
		return fmt.Errorf("serialize: write shard id failed, err: %s", err)
	}
	if err := utils.WriteAddress(w, this.User); err != nil {
		return fmt.Errorf("serialize: write addr failed, err: %s", err)
	}
	if err := serialization.WriteString(w, this.PeerPubKey); err != nil {
		return fmt.Errorf("serialize: write pub key failed, err: %s", err)
	}
	if err := utils.WriteVarUint(w, this.Amount); err != nil {
		return fmt.Errorf("serialize: write amount failed, err: %s", err)
	}
	return nil
}

func (this *ChangeProportionParam) Deserialize(r io.Reader) error {
	id, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read shard id failed, err: %s", err)
	}
	this.ShardId, err = types.NewShardID(id)
	if err != nil {
		return fmt.Errorf("deserialize: generate shard id failed, err: %s", err)
	}
	user, err := utils.ReadAddress(r)
	if err != nil {
		return fmt.Errorf("deserialize: read addr failed, err: %s", err)
	}
	this.User = user
	pubKey, err := serialization.ReadString(r)
	if err != nil {
		return fmt.Errorf("deserialize: read pub key failed, err: %s", err)
	}
	this.PeerPubKey = pubKey
	amount, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read amount failed, err: %s", err)
	}
	this.Amount = amount
	return nil
}

type DeletePeerParam struct {
	ShardId types.ShardID
	Peers   []keypair.PublicKey
}

func (this *DeletePeerParam) Serialize(w io.Writer) error {
	if err := utils.WriteVarUint(w, this.ShardId.ToUint64()); err != nil {
		return fmt.Errorf("serialize: write shard id failed, err: %s", err)
	}
	if err := utils.WriteVarUint(w, uint64(len(this.Peers))); err != nil {
		return fmt.Errorf("serialize: write peer num failed, err: %s", err)
	}
	for index, peer := range this.Peers {
		if err := serialization.WriteVarBytes(w, keypair.SerializePublicKey(peer)); err != nil {
			return fmt.Errorf("serialize: write pub key failed, index %d, err: %s", index, err)
		}
	}
	return nil
}

func (this *DeletePeerParam) Deserialize(r io.Reader) error {
	id, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read shard id failed, err: %s", err)
	}
	this.ShardId, err = types.NewShardID(id)
	if err != nil {
		return fmt.Errorf("deserialize: generate shard id failed, err: %s", err)
	}
	peersNum, err := utils.ReadVarUint(r)
	if err != nil {
		return fmt.Errorf("deserialize: read peers num failed, err: %s", err)
	}
	peers := make([]keypair.PublicKey, 0)
	for i := uint64(0); i < peersNum; i++ {
		pubKeyData, err := serialization.ReadVarBytes(r)
		if err != nil {
			return fmt.Errorf("deserialize: read peer failed, index %d, err: %s", i, err)
		}
		peer, err := keypair.DeserializePublicKey(pubKeyData)
		if err != nil {
			return fmt.Errorf("deserialize: deserialize pub key failed, index %d, err: %s", i, err)
		}
		peers = append(peers, peer)
	}
	this.Peers = peers
	return nil
}