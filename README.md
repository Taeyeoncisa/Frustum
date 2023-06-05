# Frustum
#### Implementation

We implement a prototype of Frustum and its comparative algorithms in Golang to evaluate their performance. To enable the distributed nodes and consensus process, we leverage coroutines in Golang. These coroutines allow us to simulate the behavior of individual nodes effectively. Communication between nodes is facilitated through pipelines, whereby messages are passed between them. We also implement the necessary modules for experimental evaluation, such as blockchain data storage, transaction pools, and network communication. 

In the implementation, a client has a corresponding F-shard leader and continuously sends transaction requests to it. The node caches them in its transaction pool. Once the node becomes a global leader and obtains the right to issue blocks, it packs the transactions from the transaction pool to generate new blocks and then initiates the consensus process.

#### DataSet

We employ a real blockchain transaction trace, named Ethereum transactions, extracted from XBlockETH. The dataset comes from the real Ether transaction data pulled up using the XBlock-ETH tool.（https://github.com/tczpl/XBlock-ETH）

The original transaction data is shown below.

![image-20230605160508707](https://p.ipic.vip/zd6m8m.png)

This paper only retains some information and the transaction data structure used is as follows.

 ```go
type Transaction struct {
	Hash		[32]byte	
	Inputs		int				
	Outputs		int				
	Value		int			
}
 ```


