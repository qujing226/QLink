package did

import (
	"github.com/qujing226/QLink/did/blockchain"
)

// 使用统一的区块链接口定义
type BlockchainInterface = blockchain.BlockchainInterface
type BlockchainTransaction = blockchain.Transaction
type BlockchainBlock = blockchain.Block
type BlockchainConfig = blockchain.BlockchainConfig
