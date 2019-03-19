# Relayer节点使用介绍

## 构建开发环境

成功编译relayer节点需要以下准备：

* Golang版本在1.9及以上
* 安装第三方包管理工具glide
* 正确的Go语言开发环境
* Golang所支持的操作系统

## 获取ontology relayer节点代码

```git
git clone https://github.com/laizy/ontology/tree/block-relayer
```

用第三方包管理工具glide拉取依赖库

````shell
$ cd $GOPATH/src/github.com/ontio/ontology
$ glide install
````

如果项目有新的第三方依赖包，使用glide更新依赖库

````shell
$ cd $GOPATH/src/github.com/ontio/ontology
$ glide update
````

编译源码
```
go build block-relayer.go
```

成功编译后会生成可以执行程序

* `block-relayer`: 节点程序/以命令行方式提供的节点控制程序

## 运行relayer节点

在运行relayer节点之前，清先准备好`peers.upstream`文件,该文件名可以变化,该文件用于配置同步区块所需要的节点ip地址和端口。
`peers.upstream`文件内容示例:
```
{
"upstream":["23.99.134.190:20338"]
}
```

执行下面的命令启动relayer节点
```
./block-relayer --upstream-file upstream节点配置文件名
```
如果upstream节点配置文件名是`peers.upstream`
启动命令可以省略为
```
./block-relayer
```