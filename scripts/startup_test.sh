#!/bin/bash

rm -r ./testnet
mkdir ./testnet
cd ./testnet
cp ../ontology.exe .

./ontology.exe --networkid=2

