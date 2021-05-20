package ethrpc

import (
	"github.com/ethereum/go-ethereum/rpc"
	cfg "github.com/ontio/ontology/common/config"
	"net/http"
	"strconv"
)

func StartEthServer() error {
	calculator := new(EthereumAPI)
	server := rpc.NewServer()
	err := server.RegisterName("eth", calculator)
	if err != nil {
		return err
	}
	netRpcService := new(PublicNetAPI)
	err = server.RegisterName("net", netRpcService)
	if err != nil {
		return err
	}
	err = http.ListenAndServe(":"+strconv.Itoa(int(cfg.DefConfig.Rpc.EthJsonPort)), server)
	if err != nil {
		return err
	}
	return nil
}
