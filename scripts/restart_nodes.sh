#!/bin/bash

# 需要生成的总节点数
N=4
# 初始启动节点数
SN=4

password="passwordtest"

# 生成各节点的配置文件， 并启动节点
for ((i=1; i<=$N; i=i+1)); do
    cd ./node$i
	cp ../ont.exe .
	cp ../nodectl.exe .

    if  [ $i -le $SN ]; then
        ./ont.exe -p $password &
    fi

    cd ..
done

