#!/usr/bin/env bash
set -ex

VERSION=$(git describe --always --tags --long)
PLATFORM=""

if [ $TRAVIS_OS_NAME == 'linux' ]; then
	echo "linux sys"
	PLATFORM="linux"
	env GO111MODULE=on make all

	env GO111MODULE=on go mod vendor
	cd ./wasmtest && bash ./run-wasm-tests.sh && cd ../
	bash ./.travis.check-license.sh
	bash ./.travis.gofmt.sh
	bash ./.travis.gotest.sh
elif [ $TRAVIS_OS_NAME == 'osx' ]; then
	echo "osx sys"
	PLATFORM="darwin"
	env GO111MODULE=on make all
else
	PLATFORM="windows"
	echo "win sys"
	env GO111MODULE=on CGO_ENABLED=1 go build  -ldflags "-X github.com/ontio/ontology/common/config.Version=${VERSION}" -o ontology-windows-amd64 main.go
	env GO111MODULE=on go build  -ldflags "-X github.com/ontio/ontology/common/config.Version=${VERSION}" -o sigsvr-windows-amd64 sigsvr.go
fi

mkdir releases
cd releases
cp ../ontology ontology-{PLATFORM}-amd64
mkdir tool-{PLATFORM}
cp ../ontology/tools/abi tool-{PLATFORM} -a
cp /opt/gopath/src/github.com/ontio/ontology/tools/abi $target_dir/tool-linux -a
cp /opt/gopath/src/github.com/ontio/ontology/tools/abi $target_dir/tool-windows -a
cp /opt/gopath/src/github.com/ontio/ontology/tools/sigsvr-darwin-amd64 $target_dir/tool-darwin
cp /opt/gopath/src/github.com/ontio/ontology/tools/sigsvr-linux-amd64 $target_dir/tool-linux
cp /opt/gopath/src/github.com/ontio/ontology/tools/sigsvr-windows-amd64.exe $target_dir/tool-windows
zip -q -r tool-darwin.zip tool-darwin;
zip -q -r tool-linux.zip tool-linux;
zip -q -r tool-windows.zip tool-windows;
rm -r tool-darwin;
rm -r tool-linux;
rm -r tool-windows;

set +x
echo "ontology-darwin-amd64 |" $(md5sum ontology-darwin-amd64|cut -d ' ' -f1)
echo "ontology-linux-amd64 |" $(md5sum ontology-linux-amd64|cut -d ' ' -f1)
echo "ontology-windows-amd64.exe |" $(md5sum ontology-windows-amd64.exe|cut -d ' ' -f1)
echo "tool-darwin.zip |" $(md5sum tool-darwin.zip|cut -d ' ' -f1)
echo "tool-linux.zip |" $(md5sum tool-linux.zip|cut -d ' ' -f1)
echo "tool-windows.zip |" $(md5sum tool-windows.zip|cut -d ' ' -f1)

