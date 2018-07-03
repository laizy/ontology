#!/bin/bash

password="passwordtest"

# 生成各节点公钥
pubkeys=""
rm -rf solo
mkdir solo

cp ontology.exe ./solo
cp config.json ./solo
cd ./solo
./nodectl.exe wallet -c -p $password
pubkey=$(./nodectl.exe wallet -l -p $password | grep public)
# 把"public key:" 字符串替换为空
pubkey=${pubkey/public key:/ }
pubkey=$(echo $pubkey)
pubkeys=\"$pubkey\"

cd ..

# 生成各节点的配置文件， 并启动节点
cd ./solo
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
    "PrintLevel": 1,
    "IsTLS": false,
    "CertPath": "./sample-cert.pem",
    "KeyPath": "./sample-cert-key.pem",
    "CAPath": "./sample-ca.pem",
    "MultiCoreNum":4,
	"MaxTxInBlock":50000,
    "ConsensusType": "solo"
  }
}
EOF

./ont.exe <<EOF
$password
EOF

cd ..

