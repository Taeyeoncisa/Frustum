package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/gob"
	"flag"
	"time"
)

func main() {
	// Program starts here. This function will spawn the x RC instances.

	// defaults in defaults.go
	// ？？coordinator的作用是什么？？
	functionPtr := flag.String("function", default_function, "coordinator or node")
	vCPUs := flag.Uint("vpcus", default_vCPUs, "amount of VCPUs available")
	instancesPerVCPUPtr := flag.Uint("instances", default_instances, "Instances per VCPU")
	// 结点总数
	// 结点运行在不同的CPU上
	nPtr := flag.Uint("n", default_n, "Total amount of nodes")
	// 委员会数目
	mPtr := flag.Uint("m", default_m, "Number of committees")
	// 总体坏节点比例，默认为1/3
	totalFPtr := flag.Uint("totalF", default_totalF, "Total adversary tolerance in the form of the divisor (1/x)")
	committeeFPtr := flag.Uint("committeeF", default_committeeF, "Committee adversary tolerance in the form of the divisor (1/x)")
	// 每个块的字节数
	BPtr := flag.Uint("B", default_B, "block size in bytes")
	// 用户数量
	nUsersPtr := flag.Uint("nUsers", default_nUsers, "users in system")
	// 交易数量？
	totalCointsPtr := flag.Uint("totalCoins", default_totalCoins, "total coins in system")
	tpsPtr := flag.Uint("tps", default_tps, "transactions per second")
	localPtr := flag.Bool("local", true, "local run on this computer")
	deltaPtr := flag.Uint("delta", default_delta, "delta")

	flag.Parse()

	var flagArgs FlagArgs

	flagArgs.function = *functionPtr
	flagArgs.vCPUs = *vCPUs
	flagArgs.instances = *instancesPerVCPUPtr
	flagArgs.n = *nPtr
	flagArgs.m = *mPtr
	flagArgs.totalF = *totalFPtr
	flagArgs.committeeF = *committeeFPtr
	// flagArgs.d = *dPtr
	flagArgs.B = *BPtr
	flagArgs.nUsers = *nUsersPtr
	flagArgs.totalCoins = *totalCointsPtr
	flagArgs.tps = *tpsPtr
	flagArgs.local = *localPtr
	flagArgs.delta = *deltaPtr

	// generate a random key to send the P256 curve interface to gob.Register because it wouldnt cooperate
	randomKey := new(PrivKey)
	randomKey.gen()

	// gob.Register的作用是什么
	// 注册后才可进行编码和解码

	// register structs with gob
	gob.Register(IDAGossipMsg{})
	gob.Register([32]uint8{})
	gob.Register(ProposedBlock{})
	gob.Register(KademliaFindNodeMsg{})
	gob.Register(KademliaFindNodeResponse{})
	gob.Register(elliptic.CurveParams{})
	gob.Register(ecdsa.PublicKey{})
	gob.Register(PubKey{})
	gob.Register(randomKey.Pub.Pub.Curve)
	gob.Register(ConsensusMsg{})
	gob.Register(Transaction{})
	gob.Register(FinalBlock{})
	dur := time.Now().Sub(time.Now())
	gob.Register(dur)
	gob.Register(ByteArrayAndTimestamp{})
	gob.Register(RequestBlockAnswer{})

	if flagArgs.local {
		coord = coord_local
	} else {
		coord = coord_aws
	}
	// log.Println("Coordinator IP: ", coord_local)

	// ensure some invariants
	if default_kappa > 256 {
		errFatal(nil, "Default kappa was over 256/1byte")
	}

	// runtime.GOMAXPROCS(int(flagArgs.vCPUs))

	if *functionPtr == "coordinator" {
		// log.Println("Launching coordinator")
		launchCoordinator(&flagArgs)
	} else {
		launchNodes(&flagArgs)
	}
	// log.Println("finished")
}

func launchNodes(flagArgs *FlagArgs) {
	// log.Println("Launcing ", flagArgs.instances, " instances")
	for i := uint(0); i < flagArgs.instances; i++ {
		go launchNode(flagArgs)
	}
	for {

	}
}
