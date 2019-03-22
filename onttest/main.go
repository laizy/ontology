package main

import (
	"fmt"
	"bytes"
	"encoding/hex"
	"os"
	"runtime"
	"io/ioutil"
	"runtime/pprof"
	"time"
	"encoding/binary"

	"github.com/ontio/ontology/common/constants"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology/cmd/utils"
	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/genesis"
	"github.com/ontio/ontology/core/ledger"
	"github.com/ontio/ontology/core/payload"
	"github.com/ontio/ontology/core/signature"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/events"
	"github.com/ontio/ontology/common/serialization"
	httpcom "github.com/ontio/ontology/http/base/common"
)

const (
	DefaultMultiCoreNum = 4
	Transfers           = 100000
	TxPerBlk            = 5000
	testFile            = "./wasm_demo_prune_no_custom.wasm"
)

func init() {
	log.Init(log.PATH, log.Stdout)
	runtime.GOMAXPROCS(4)
}

var blockBuf *bytes.Buffer

func main() {

	datadir := "testdata"
	_ = os.RemoveAll(datadir)
	log.Trace("Node version: ", config.Version)

	acct := account.NewAccount("")
	buf := keypair.SerializePublicKey(acct.PublicKey)
	config.DefConfig.Genesis.ConsensusType = "solo"
	config.DefConfig.Genesis.SOLO.GenBlockTime = 3
	config.DefConfig.Genesis.SOLO.Bookkeepers = []string{hex.EncodeToString(buf)}
	log.Debug("The Node's PublicKey ", acct.PublicKey)

	bookkeepers := []keypair.PublicKey{acct.PublicKey}
	//Init event hub
	events.Init()

	log.Info("1. Loading the Ledger")
	var err error
	ledger.DefLedger, err = ledger.NewLedger(datadir, 100000000)
	if err != nil {
		log.Fatalf("NewLedger error %s", err)
		os.Exit(1)
	}
	genblock, err := genesis.BuildGenesisBlock(bookkeepers, config.DefConfig.Genesis)
	if err != nil {
		log.Error(err)
		return
	}
	err = ledger.DefLedger.Init(bookkeepers, genblock)
	if err != nil {
		log.Fatalf("DefLedger.Init error %s", err)
		os.Exit(1)
	}

	wallet, err := account.Open("./wallet.dat")
	if err != nil {
		fmt.Printf("getWallet error:%s\n", err.Error())
		return
	}
	acct1, err := wallet.GetDefaultAccount([]byte("123456"))
	if err != nil {
		fmt.Printf("GetDefaultAccount error:%s\n", err.Error())
		return
	}

	signer := &account.Account{PrivateKey: acct1.PrivateKey,
		PublicKey: acct1.PublicKey,
		Address:   acct1.Address,
		SigScheme: acct1.SigScheme}

	testWasm := false

	if testWasm {
		//deploy wasmcontract
		log.Info("2. Deploy wasm contract")

		//testFile := "./cOEP4.wasm"
		code, err := ioutil.ReadFile(testFile)
		if err != nil {
			fmt.Printf("open wasmfile error:%s\n", err.Error())
			return
		}
		tx := NewDeployWasmVMTx(code)
		dptx, err := tx.IntoImmutable()
		if err != nil {
			fmt.Printf("IntoImmutable error:%s\n", err.Error())
			return
		}

		contractAddr :=  common.AddressFromVmCode(code)
		if err != nil {
			fmt.Printf("GetContractAddress error:%s\n", err.Error())
			return
		}
		fmt.Printf("contractAddr:%s\n", contractAddr.ToBase58())
		err = signTransaction(signer, tx)
		if err != nil {
			fmt.Printf("signTransaction error:%s\n", err.Error())
			return
		}

		dpblk, err := makeBlock(acct, []*types.Transaction{dptx})

		err = ledger.DefLedger.AddBlock(dpblk, common.UINT256_EMPTY)
		if err != nil {
			fmt.Println("persist block error", err)
			return
		}

		//invoke init()
		log.Info("3. invoke oep4 init method")

		//tx, err = NewWasmInvokeTx(contractAddr, "initialize", []interface{}{signer.Address}, sdk)
		//tx, err = NewWasmInvokeTx(contractAddr, "init", []interface{}{}, sdk)
		if err != nil {
			fmt.Println("NewWasmInvokeTx error", err)
			return
		}
		fmt.Printf("txtype is %x\n", tx.TxType)
		err = signTransaction(signer, tx)
		if err != nil {
			fmt.Printf("signTransaction error:%s\n", err.Error())
			return
		}
		inittx, _ := tx.IntoImmutable()
		blk, err := makeBlock(acct, []*types.Transaction{inittx})

		err = ledger.DefLedger.AddBlock(blk, common.UINT256_EMPTY)
		if err != nil {
			fmt.Println("persist block error", err)
			return
		}

		toacct := account.NewAccount("")
		fmt.Printf("to acct address is %s\n", toacct.Address.ToBase58())
		TxTest(acct, signer, toacct, contractAddr)
	}

	testNeo := true
	if testNeo {
		codeHash := "0133c56b6a00527ac46a51527ac46a00c304696e69749c640900652e096c7566616a00c3046e616d659c6409006505096c7566616a00c30673796d626f6c9c64090065de086c7566616a00c308646563696d616c739c64090065b8086c7566616a00c30b746f74616c537570706c799c640900654a086c7566616a00c30962616c616e63654f669c6424006a51c3c0519e640700006c7566616a51c300c36a52527ac46a52c365a3076c7566616a00c3087472616e736665729c6488006a51c3c0539e640700006c7566616a51c300c36a53527ac46a51c351c36a54527ac46a51c352c36a55527ac4006a5a527ac40002e8037c65f1096a59527ac46a59c3c06a5b527ac4616a5ac36a5bc39f642f006a59c36a5ac3c36a56527ac46a5ac351936a5a527ac46a53c36a54c36a55c302e8039652726572057562ccff6161616c7566616a00c30d7472616e736665724d756c74699c640c006a51c365b3046c7566616a00c30c7472616e7366657246726f6d9c645f006a51c3c0549e640700006c7566616a51c300c36a57527ac46a51c351c36a53527ac46a51c352c36a54527ac46a51c353c36a55527ac46a57c36a53c36a54c36a55c3537951795572755172755279527954727552727565fd006c7566616a00c307617070726f76659c6440006a51c3c0539e640700006c7566616a51c300c36a58527ac46a51c351c36a57527ac46a51c352c36a55527ac46a58c36a57c36a55c3527265f0026c7566616a00c309616c6c6f77616e63659c6432006a51c3c0529e640700006c7566616a51c300c36a58527ac46a51c351c36a57527ac46a58c36a57c37c650b006c756661006c756658c56b6a00527ac46a51527ac4681953797374656d2e53746f726167652e476574436f6e74657874616a52527ac401026a53527ac46a53c36a00c37e6a51c37e6a54527ac46a52c36a54c37c681253797374656d2e53746f726167652e476574616c7566011fc56b6a00527ac46a51527ac46a52527ac46a53527ac4681953797374656d2e53746f726167652e476574436f6e74657874616a54527ac401016a55527ac401026a56527ac46a00c3c001149e6317006a51c3c001149e630d006a52c3c001149e641a00611461646472657373206c656e677468206572726f72f0616a00c3681b53797374656d2e52756e74696d652e436865636b5769746e65737361009c640700006c7566616a55c36a51c37e6a57527ac46a54c36a57c37c681253797374656d2e53746f726167652e476574616a58527ac46a53c36a58c3a0640700006c7566616a56c36a51c37e6a00c37e6a59527ac46a54c36a59c37c681253797374656d2e53746f726167652e476574616a5a527ac46a55c36a52c37e6a5b527ac46a54c36a5bc37c681253797374656d2e53746f726167652e476574616a5c527ac46a53c36a5ac3a0640700006c7566616a53c36a5ac39c6449006a54c36a59c37c681553797374656d2e53746f726167652e44656c657465616a54c36a57c36a58c36a53c3945272681253797374656d2e53746f726167652e50757461624c00616a54c36a59c36a5ac36a53c3945272681253797374656d2e53746f726167652e507574616a54c36a57c36a58c36a53c3945272681253797374656d2e53746f726167652e50757461616a54c36a5bc36a5cc36a53c3935272681253797374656d2e53746f726167652e507574616a51c36a52c36a53c35272087472616e7366657254c1681553797374656d2e52756e74696d652e4e6f74696679516c75660111c56b6a00527ac46a51527ac46a52527ac4681953797374656d2e53746f726167652e476574436f6e74657874616a53527ac401026a54527ac46a51c3c001149e630d006a00c3c001149e641a00611461646472657373206c656e677468206572726f72f0616a00c3681b53797374656d2e52756e74696d652e436865636b5769746e65737361009c640700006c7566616a52c36a00c365a802a0640700006c7566616a54c36a00c37e6a51c37e6a55527ac46a53c36a55c36a52c35272681253797374656d2e53746f726167652e507574616a00c36a51c36a52c3527208617070726f76616c54c1681553797374656d2e52756e74696d652e4e6f74696679516c756659c56b6a00527ac4006a52527ac46a00c3c06a53527ac4616a52c36a53c39f6473006a00c36a52c3c36a51527ac46a52c351936a52527ac46a51c3c0539e6420001b7472616e736665724d756c746920706172616d73206572726f722ef0616a51c300c36a51c351c36a51c352c35272652900009c64a2ff157472616e736665724d756c7469206661696c65642ef06288ff616161516c75660117c56b6a00527ac46a51527ac46a52527ac4681953797374656d2e53746f726167652e476574436f6e74657874616a53527ac401016a54527ac46a51c3c001149e630d006a00c3c001149e641a00611461646472657373206c656e677468206572726f72f0616a00c3681b53797374656d2e52756e74696d652e436865636b5769746e65737361009c640700006c7566616a54c36a00c37e6a55527ac46a53c36a55c37c681253797374656d2e53746f726167652e476574616a56527ac46a52c36a56c3a0640700006c7566616a52c36a56c39c6425006a53c36a55c37c681553797374656d2e53746f726167652e44656c65746561622800616a53c36a55c36a56c36a52c3945272681253797374656d2e53746f726167652e50757461616a54c36a51c37e6a57527ac46a53c36a57c37c681253797374656d2e53746f726167652e476574616a58527ac46a53c36a57c36a58c36a52c3935272681253797374656d2e53746f726167652e507574616a00c36a51c36a52c35272087472616e7366657254c1681553797374656d2e52756e74696d652e4e6f74696679516c756658c56b6a00527ac4681953797374656d2e53746f726167652e476574436f6e74657874616a51527ac401016a52527ac46a00c3c001149e6419001461646472657373206c656e677468206572726f72f0616a51c36a52c36a00c37e7c681253797374656d2e53746f726167652e476574616c756655c56b681953797374656d2e53746f726167652e476574436f6e74657874616a00527ac40b546f74616c537570706c796a51527ac46a00c36a51c37c681253797374656d2e53746f726167652e476574616c756654c56b586a00527ac46a00c36c756654c56b034d59546a00527ac46a00c36c756654c56b074d79546f6b656e6a00527ac46a00c36c75660113c56b681953797374656d2e53746f726167652e476574436f6e74657874616a00527ac40400e1f5056a51527ac422415166344d7a7531594a72687a39663361526b6b77536d396e3371685847536834707514616f2a4a38396ff203ea01e6c070ae421bb8ce2d6a52527ac40400ca9a3b6a53527ac401016a54527ac40b546f74616c537570706c796a55527ac46a52c3c001149e6432000e4f776e657220696c6c6567616c2151c176c9681553797374656d2e52756e74696d652e4e6f7469667961006c7566616a00c36a55c37c681253797374656d2e53746f726167652e4765746164340014416c726561647920696e697469616c697a656421681553797374656d2e52756e74696d652e4e6f7469667961006c7566616a53c36a51c3956a56527ac46a00c36a55c36a56c35272681253797374656d2e53746f726167652e507574616a00c36a54c36a52c37e6a56c35272681253797374656d2e53746f726167652e50757461006a52c36a56c35272087472616e7366657254c1681553797374656d2e52756e74696d652e4e6f74696679516c7566006c75665ec56b6a00527ac46a51527ac46a51c36a00c3946a52527ac46a52c3c56a53527ac4006a54527ac46a00c36a55527ac461616a00c36a51c39f6433006a54c36a55c3936a56527ac46a56c36a53c36a54c37bc46a54c351936a54527ac46a55c36a54c3936a00527ac462c8ff6161616a53c36c7566"

		contractcode, _ := hex.DecodeString(codeHash)
		codeAddress := common.AddressFromVmCode(contractcode)
		fmt.Println("codeAddress:" + codeAddress.ToBase58())
		tx := NewDeployNeoVMTx(contractcode)

		err = signTransaction(signer, tx)
		if err != nil {
			fmt.Printf("signTransaction error:%s\n", err.Error())
			return
		}
		dptx, _ := tx.IntoImmutable()

		dpblk, err := makeBlock(acct, []*types.Transaction{dptx})

		err = ledger.DefLedger.AddBlock(dpblk, common.UINT256_EMPTY)
		if err != nil {
			fmt.Println("persist block error", err)
			return
		}
		fmt.Println("deploy end")
		//invoke init
		tx, err = NewInvokeNeoVMTx(codeAddress, []interface{}{"init", []interface{}{}})
		if err != nil {
			fmt.Printf("NewInvokeNeoVMTx error:%s\n", err.Error())
			return
		}
		err = signTransaction(signer, tx)
		if err != nil {
			fmt.Printf("signTransaction error:%s\n", err.Error())
			return
		}
		invtx, _ := tx.IntoImmutable()
		invblk, err := makeBlock(acct, []*types.Transaction{invtx})
		err = ledger.DefLedger.AddBlock(invblk, common.UINT256_EMPTY)
		if err != nil {
			fmt.Println("persist block error", err)
			return
		}
		fmt.Println("init end")

		toacct := account.NewAccount("")

		neotxTest(acct, signer, toacct, codeAddress)

	}

}

func NewDeployNeoVMTx(contractCode []byte) *types.MutableTransaction {
	deployPayload := &payload.DeployCode{
		Code:        contractCode,
		NeedStorage: false,
		Name:        "neoc",
		Version:     "test",
		Author:      "test",
		Email:       "test",
		Description: "test",
	}
	tx := &types.MutableTransaction{
		Version:  0,
		TxType:   types.Deploy,
		Nonce:    uint32(time.Now().Unix()),
		Payload:  deployPayload,
		GasPrice: 0,
		GasLimit: 20000,
		Sigs:     make([]types.Sig, 0, 0),
	}
	return tx
}

func NewInvokeNeoVMTx(contractAddress common.Address, params []interface{}) (*types.MutableTransaction, error) {
	invokeCode, err := httpcom.BuildNeoVMInvokeCode(contractAddress, params)
	if err != nil {
		return nil, err
	}
	return utils.NewInvokeTransaction(0, 20000, invokeCode), nil
}

func neotxTest(issuer *account.Account, signer *account.Account, toacct *account.Account, contractAddress common.Address) {

	loopcnt := Transfers
	txs := make([]*types.Transaction, loopcnt)
	start := time.Now().UnixNano()
	for i := 0; i < loopcnt; i++ {
		tx, _ := NewInvokeNeoVMTx(contractAddress, []interface{}{"transfer", []interface{}{signer.Address[:], toacct.Address[:], 1000}})
		err := signTransaction(signer, tx)
		if err != nil {
			fmt.Printf("signTransaction error:%s\n", err.Error())
			return
		}
		txs[i], _ = tx.IntoImmutable()
	}
	fmt.Printf("make transfer:%d, cost:%d ns\n", loopcnt, time.Now().UnixNano()-start)
	signerbalance := getOEP4NeoBalance(contractAddress, signer.Address)
	toacctBalance := getOEP4NeoBalance(contractAddress, toacct.Address)
	fmt.Printf("before test signerbalance :%d\n, toacctBalance:%d\n", signerbalance, toacctBalance)

	txPerBlock := TxPerBlk
	for j := 0; j < loopcnt/txPerBlock; j++ {
		blk, err := makeBlock(issuer, txs[j*txPerBlock:(j+1)*txPerBlock])
		if err != nil {
			fmt.Printf("makeBlock error :%s\n", err.Error())
			return
		}
		start = time.Now().UnixNano()
		f, err := os.Create("cpu.prof")
		if err != nil {
			return
		}

		pprof.StartCPUProfile(f)
		err = ledger.DefLedger.AddBlock(blk, common.UINT256_EMPTY)
		if err != nil {
			fmt.Println("persist block error", err)
			return
		}

		pprof.StopCPUProfile()
		fmt.Printf("exec transfer:%d, cost:%f s\n", txPerBlock, float64(time.Now().UnixNano()-start)/float64(time.Second))
		return
	}

	fmt.Println("done")
	signerbalance = getOEP4NeoBalance(contractAddress, signer.Address)
	toacctBalance = getOEP4NeoBalance(contractAddress, toacct.Address)
	fmt.Printf("after test signerbalance :%d\n, toacctBalance:%d\n", signerbalance, toacctBalance)
}

func NewDeployWasmVMTx(contractCode []byte) *types.MutableTransaction {
	deployPayload := &payload.DeployCode{
		Code:        contractCode,
		NeedStorage: true,
		Name:        "test",
		Version:     "1.0",
		Author:      "test",
		Email:       "test",
		Description: "test",
	}
	tx := &types.MutableTransaction{
		Version:  0,
		TxType:   types.Deploy,
		Nonce:    uint32(time.Now().Unix()),
		Payload:  deployPayload,
		GasPrice: 0,
		GasLimit: 20000000,
		Sigs:     make([]types.Sig, 0, 0),
	}
	return tx
}

//func NewWasmInvokeTx(contractAddress common.Address, method string, params []interface{}, sdk *ontology_go_sdk.OntologySdk) (*types.MutableTransaction, error) {
//	contract := &states.WasmContractParam{}
//	contract.Address = contractAddress
//	argbytes, err := buildWasmContractParam(method, params)
//	if err != nil {
//		return nil, fmt.Errorf("buildWasmContractParam error:%s\n", err.Error())
//	}
//	contract.Args = argbytes
//	sink := common.NewZeroCopySink(nil)
//	contract.Serialization(sink)
//	tx := sdk.NewInvokeWasmTransaction(0, 20000000, sink.Bytes())
//	return tx, nil
//}

//for wasm vm
//build param bytes for wasm contract
func buildWasmContractParam(method string, params []interface{}) ([]byte, error) {
	bf := bytes.NewBuffer(nil)
	serialization.WriteString(bf, method)
	for _, param := range params {
		switch param.(type) {
		case string:
			tmp := bytes.NewBuffer(nil)
			serialization.WriteString(tmp, param.(string))
			bf.Write(tmp.Bytes())
		case int:
			tmpBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(tmpBytes, uint32(param.(int)))
			bf.Write(tmpBytes)
		case int64:
			tmpBytes := make([]byte, 8)
			binary.LittleEndian.PutUint64(tmpBytes, uint64(param.(int64)))
			bf.Write(tmpBytes)
		case uint16:
			tmpBytes := make([]byte, 2)
			binary.LittleEndian.PutUint16(tmpBytes, param.(uint16))
			bf.Write(tmpBytes)
		case uint32:
			tmpBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(tmpBytes, param.(uint32))
			bf.Write(tmpBytes)
		case uint64:
			tmpBytes := make([]byte, 8)
			binary.LittleEndian.PutUint64(tmpBytes, param.(uint64))
			bf.Write(tmpBytes)
		case []byte:
			tmp := bytes.NewBuffer(nil)
			serialization.WriteVarBytes(tmp, param.([]byte))
			bf.Write(tmp.Bytes())
		case common.Uint256:
			bs := param.(common.Uint256)
			parambytes := bs[:]
			bf.Write(parambytes)
		case common.Address:
			bs := param.(common.Address)
			parambytes := bs[:]
			bf.Write(parambytes)
		case byte:
			bf.WriteByte(param.(byte))

		default:
			return nil, fmt.Errorf("not a supported type :%v\n", param)
		}
	}
	return bf.Bytes(), nil

}

func TxTest(issuer *account.Account, signer *account.Account, toacct *account.Account, contractAddress common.Address) {

	//from := signer.Address
	//to := toacct.Address
	//
	//params := []interface{}{from, to, uint64(10)}
	//loopcnt := Transfers
	//
	//txs := make([]*types.Transaction, loopcnt)
	//start := time.Now().UnixNano()
	//for i := 0; i < loopcnt; i++ {
	//	tx, err := NewWasmInvokeTx(contractAddress, "transfer", params, sdk)
	//	if err != nil {
	//		fmt.Printf("NewWasmInvokeTx error :%s\n", err.Error())
	//		return
	//	}
	//	err = signTransaction(signer, tx)
	//	if err != nil {
	//		fmt.Printf("signTransaction error :%s\n", err.Error())
	//		return
	//	}
	//	txs[i], err = tx.IntoImmutable()
	//	if err != nil {
	//		fmt.Printf("signTransaction error :%s\n", err.Error())
	//		return
	//	}
	//}
	//fmt.Printf("make transfer:%d, cost:%d ns\n",loopcnt,time.Now().UnixNano() - start)
	//
	//signerbalance := getOEP4Balance(contractAddress, signer.Address)
	//toacctBalance := getOEP4Balance(contractAddress, toacct.Address)
	//fmt.Printf("before test signerbalance :%d\n, toacctBalance:%d\n", signerbalance, toacctBalance)
	//
	//txPerBlock := TxPerBlk
	//start =time.Now().UnixNano()
	//for j := 0; j < loopcnt/txPerBlock; j++ {
	//	blk, err := makeBlock(issuer, txs[j*txPerBlock:(j+1)*txPerBlock])
	//	if err != nil {
	//		fmt.Printf("makeBlock error :%s\n", err.Error())
	//		return
	//	}
	//	err = ledger.DefLedger.AddBlock(blk, common.UINT256_EMPTY)
	//	if err != nil {
	//		fmt.Println("persist block error", err)
	//		return
	//	}
	//}
	//fmt.Printf("exec transfer:%d, cost:%d ns\n",loopcnt,(time.Now().UnixNano() - start))
	//
	//fmt.Println("done")
	//signerbalance = getOEP4Balance(contractAddress, signer.Address)
	//toacctBalance = getOEP4Balance(contractAddress, toacct.Address)
	//fmt.Printf("after test signerbalance :%d\n, toacctBalance:%d\n", signerbalance, toacctBalance)

}

func signTransaction(signer *account.Account, tx *types.MutableTransaction) error {
	hash := tx.Hash()
	sign, _ := signature.Sign(signer, hash[:])
	tx.Sigs = append(tx.Sigs, types.Sig{
		PubKeys: []keypair.PublicKey{signer.PublicKey},
		M:       1,
		SigData: [][]byte{sign},
	})
	return nil
}

func checkEq(a, b uint64) {
	if a != b {
		panic(fmt.Sprintf("not equal. a %d, b %d", a, b))
	}
}

func getOEP4Balance(contractaddr common.Address, addr common.Address) uint64 {
	//key := bytes.NewBuffer([]byte{1})
	key := bytes.NewBuffer([]byte("b"))

	key.Write(addr[:])

	balanceBytes, _ := ledger.DefLedger.GetStorageItem(contractaddr, key.Bytes())
	//fmt.Printf("balanceBytes:%v\n",balanceBytes)
	//balanceU256, err := common.Uint256ParseFromBytes(balanceBytes)
	//if err != nil {
	//	fmt.Printf("error is %s\n", err.Error())
	//}
	if len(balanceBytes) == 8 {
		return binary.LittleEndian.Uint64(balanceBytes[:8])
	}

	return 0

}

func getOEP4NeoBalance(contractaddr common.Address, addr common.Address) uint64 {
	key := bytes.NewBuffer([]byte("b"))
	key.Write(addr[:])

	balanceBytes, _ := ledger.DefLedger.GetStorageItem(contractaddr, key.Bytes())
	bi := common.BigIntFromNeoBytes(balanceBytes)
	return bi.Uint64()
}

func makeBlock(acc *account.Account, txs []*types.Transaction) (*types.Block, error) {
	nextBookkeeper, err := types.AddressFromBookkeepers([]keypair.PublicKey{acc.PublicKey})
	if err != nil {
		return nil, fmt.Errorf("GetBookkeeperAddress error:%s", err)
	}
	prevHash := ledger.DefLedger.GetCurrentBlockHash()
	height := ledger.DefLedger.GetCurrentBlockHeight()

	nonce := uint64(height)
	txHash := []common.Uint256{}
	for _, t := range txs {
		txHash = append(txHash, t.Hash())
	}

	txRoot := common.ComputeMerkleRoot(txHash)
	if err != nil {
		return nil, fmt.Errorf("ComputeRoot error:%s", err)
	}
	param := []common.Uint256{txRoot}

	blockRoot := ledger.DefLedger.GetBlockRootWithNewTxRoots(height+1, param)
	header := &types.Header{
		Version:          0,
		PrevBlockHash:    prevHash,
		TransactionsRoot: txRoot,
		BlockRoot:        blockRoot,
		Timestamp:        constants.GENESIS_BLOCK_TIMESTAMP + height + 1,
		Height:           height + 1,
		ConsensusData:    nonce,
		NextBookkeeper:   nextBookkeeper,
	}
	block := &types.Block{
		Header:       header,
		Transactions: txs,
	}

	blockHash := block.Hash()

	sig, err := signature.Sign(acc, blockHash[:])
	if err != nil {
		return nil, fmt.Errorf("[Signature],Sign error:%s.", err)
	}

	block.Header.Bookkeepers = []keypair.PublicKey{acc.PublicKey}
	block.Header.SigData = [][]byte{sig}
	return block, nil
}
