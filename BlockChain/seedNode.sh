#! /bin/bash
# shellcheck shell=bash

# 节点启动地址
hostIP=$(ifconfig -a|grep inet|grep -v 127.0.0.1|grep -v inet6|awk '{print $2}'|tr -d "addr:"|grep "172.16.2")
#echo $host
# 节点启动端口
port=10000

# 日志等级，默认为1（debug）
loglevel=1


# build new executable file
go build -o seedNode

#./seedNode -a "$hostIP":$port -l $loglevel
./seedNode -p $port -l $loglevel
