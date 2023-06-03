#! /bin/bash
# shellcheck shell=bash

help() {
  echo "Usage:"
  echo "./start.sh [-n NodeID] [-l LogLevel]"
  echo "NodeID: 1-1(default)"
  echo "LogLevel: 1[DEBUG](default), 0[INFO]"
  exit 10
}

# resolve local IP Addr
#hostIP=$(ifconfig -a|grep inet|grep -v 127.0.0.1|grep -v inet6|awk '{print $2}'|tr -d "addr:"|grep "172.16.2")
# shard id, default: 1
nodeID="1-1"
# log level，default: 1(DEBUG)
loglevel=1
# consensusType, default: 0(SS-PBFT)
consensusType=0
# option func
while getopts 'n:l:c:h' OPT; do

    case $OPT in
      n) nodeID="$OPTARG";;
      l) loglevel="$OPTARG" ;;
      c) consensusType="$OPTARG" ;;
      h) help ;;
      ?) help ;;
    esac

done

echo "nodeID:$nodeID"
echo "logLevel:$loglevel"

# split string to get ShardID and IntraShardID
IFS='-'
read -a strArr <<<"$nodeID"

echo "ShardID: ${strArr[0]}"
echo "IntraShardID: ${strArr[1]}"

# build new executable file
go build -o node

port=$(expr "${strArr[1]}" + "${strArr[0]}" \* 10000 + 10)
#echo "Node NetWork Addr: $hostIP:$port"

#./node -a "$hostIP":"$port" -n "$nodeID" -c "$consensusType" -l "$loglevel"
./node -p "$port" -n "$nodeID" -c "$consensusType" -l "$loglevel"


