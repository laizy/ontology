#!/bin/bash

# 需要生成的总节点数
N=4
# 初始启动节点数
SN=4

password="123"

# 生成各节点公钥
pubkeys=""
for ((i=1; i<=$N; i=i+1)); do
    rm -rf node$i
    mkdir node$i

    cp ontology.exe ./node$i
    #cp wallets/node$i.dat ./node$i
    cd ./node$i
	echo -e $password\\n$password | ./ontology.exe account add -d
    pubkey=$(./ontology.exe account list -w wallet.dat -v | grep Public)
    # 把"public key:" 字符串替换为空
    pubkey=${pubkey/Public key:/ }
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
    cat > ./config.json << EOF
{
  "SeedList": [
    "127.0.0.1:21338",
    "127.0.0.1:22338",
    "127.0.0.1:23338",
    "127.0.0.1:24338"
  ],
  "ConsensusType":"dbft",
  "DBFT":{
    "Bookkeepers":[ $pubkeys ],
    "GenBlockTime":6
  }
}
EOF

    if  [ $i -le $SN ]; then
		nodeport=2${i}338
		consport=2${i}339 
		rpcport=2${i}336
		restport=2${i}334
		echo $password | ./ontology.exe --enableconsensus --config=config.json --nodeport=$nodeport --consensusport=$consport --rest --restport=$restport --rpcport=$rpcport -w wallet.dat & 
	fi
    cd ..
done

