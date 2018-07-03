#!/bin/bash

password="passwordtest"

# 生成各节点公钥
pubkeys=""
rm -rf miner_block
mkdir miner_block


go build -o ont.exe ../onttest.go

cp ont.exe ./miner_block
cp config.json ./miner_block
cp nodectl.exe ./miner_block
cd ./miner_block
./nodectl.exe wallet -c -p $password
pubkey=$(./nodectl.exe wallet -l -p $password | grep public)
# 把"public key:" 字符串替换为空
pubkey=${pubkey/public key:/ }
pubkey=$(echo $pubkey)
pubkeys=\"$pubkey\"

cd ..

# 生成各节点的配置文件， 并启动节点
istest=true
cd ./miner_block
cat > ./config.json << EOF
{
  "Configuration": {
    "Magic": 7630401,
    "Version": 23,
    "SeedList": [
      "127.0.0.1:11338"
    ],
    "Bookkeepers":[ $pubkeys, $pubkeys, $pubkeys, $pubkeys ],
	"TxValidInterval":50,
    "HttpRestPort": 20334,
    "HttpWsPort": 20335,
    "HttpJsonPort": 20336,
    "HttpLocalPort": 20337,
    "NodePort": 20338,
	"TimestampSources":["http://timestamp.sheca.com/Timestamp/pdftime.do"],
    "PrintLevel": 2,
    "IsTLS": false,
    "CertPath": "./sample-cert.pem",
    "KeyPath": "./sample-cert-key.pem",
    "CAPath": "./sample-ca.pem",
    "MultiCoreNum":4,
	"MaxTxInBlock":50000,
    "RunTest": $istest,
    "ConsensusType": "miner_block"
  }
}
EOF

./ont.exe -p $password miner_block 2> /dev/null

cd ..

