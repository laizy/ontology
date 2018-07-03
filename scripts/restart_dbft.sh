#!/bin/bash

# 需要生成的总节点数
N=4
# 初始启动节点数
SN=4

password="123"

# 生成各节点的配置文件， 并启动节点
for ((i=1; i<=$N; i=i+1)); do
    cd ./node$i
	cp ../ontology.exe .

    if  [ $i -le $SN ]; then
		nodeport=2${i}338
		consport=2${i}339 
		rpcport=2${i}336
		restport=2${i}334
		echo $password | ./ontology.exe --nodeport=$nodeport --consensusport=$consport --rest --restport=$restport --rpcport=$rpcport -w wallet.dat & 
    fi

    cd ..
done

