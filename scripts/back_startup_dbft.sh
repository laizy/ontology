#!/bin/bash

# 需要生成的总节点数
N=4
# 初始启动节点数
SN=4

password="passwordtest"

# 生成各节点公钥
pubkeys=""
for ((i=1; i<=$N; i=i+1)); do
    rm -rf node$i
    mkdir node$i

    cp ont.exe ./node$i
    cp config.json ./node$i
    cp nodectl.exe ./node$i
    cd ./node$i
    ./nodectl.exe wallet -c -p $password
    pubkey=$(./nodectl.exe wallet -l -p $password | grep public)
    # 把"public key:" 字符串替换为空
    pubkey=${pubkey/public key:/ }
    pubkey=$(echo $pubkey)
                
    if [ $i -eq 1 ]; then
        pubkeys=\"$pubkey\"
    elif [ $i -le $SN ]; then
        pubkeys="$pubkeys",\"$pubkey\"
    fi

    cd ..
done

# 生成各节点的配置文件， 并启动节点
for ((i=1; i<=$N; i=i+1)); do
    if [ $i -eq 1 ]; then
        istest=true
    else
        istest=false
    fi
    cd ./node$i
    cp ./wallet.dat ../node1/wallet$i.dat
    cat > ./config.json << EOF
{
  "Configuration": {
    "Magic": 7630401,
    "Version": 23,
    "SeedList": [
      "127.0.0.1:11338"
    ],
    "Bookkeepers":[ $pubkeys ],
	"TxValidInterval":50,
    "HttpRestPort": 1${i}334,
    "HttpWsPort": 1${i}335,
    "HttpJsonPort": 1${i}336,
    "HttpLocalPort": 1${i}337,
    "NodePort": 1${i}338,
    "NodeConsensusPort": 1${i}339,
    "PrintLevel": 1,
    "IsTLS": false,
    "CertPath": "./sample-cert.pem",
    "KeyPath": "./sample-cert-key.pem",
    "CAPath": "./sample-ca.pem",
    "MultiCoreNum":4,
	"MaxTxInBlock":50000,
    "RunTest": $istest
  }
}
EOF

    if  [ $i -le $SN ]; then
		echo $password | ./ont.exe & 
	fi
    cd ..
done

