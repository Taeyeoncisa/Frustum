#! /bin/bash
# shellcheck shell=bash

help() {
  echo "Usage:"
  echo "./start.sh [-n NodeID] [-l LogLevel]"
  echo "NodeID: 1-1(default)"
  echo "LogLevel: 1[DEBUG](default), 0[INFO]"
  exit 10
}

#IP地址
host=$(ifconfig -a|grep inet|grep -v 127.0.0.1|grep -v inet6|awk '{print $2}'|tr -d "addr:"|grep "172.16.2")
# 默认启动选项值
ShardNum=1
IntraShardNodeNum=8
ShardIDStartPoint=1
loglevel=1
consensusType=0

# option func
while getopts 's:i:p:c:h' OPT; do

    case $OPT in
      s) ShardNum="$OPTARG";;
      i) IntraShardNodeNum="$OPTARG" ;;
      p) ShardIDStartPoint="$OPTARG" ;;
      c) consensusType="$OPTARG" ;;
      h) help ;;
      ?) help ;;
    esac

done

cpuCoreNum=0
port=10000
ShardIDEndPoint=$(expr "$ShardIDStartPoint" + "$ShardNum" - 1)

for i in $(seq "$ShardIDStartPoint" "$ShardIDEndPoint"); do

  for j in $(seq 1 "$IntraShardNodeNum"); do

#  port=$(expr "$j" + "$i" \* 10000 + 10)
  echo "NodeID:""$i"-"$j" "start at:""$host":"$port"
  echo "Node is working on CPU Core $cpuCoreNum"

#  taskset -c $cpuCoreNum ./node -a "$host":"$port" -n "$i"-"$j" -c "$consensusType" -l $loglevel &
  taskset -c $cpuCoreNum ./node -a "$host":"$port" -n "$i"-"$j" -c "$consensusType" -l $loglevel >>"$i"-"$j".log 2>&1 &

  cpuCoreNum=$(expr $cpuCoreNum + 1)
  port=$(expr "$port" + 1)

  sleep 0.1
  done

done
