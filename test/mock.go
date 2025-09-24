package test

import (
	"context"
	"time"

	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/pkg/types"
)

// MockBlockchain 模拟区块链实现
type MockBlockchain struct{}

// 连接管理方法
func (m *MockBlockchain) Connect(ctx context.Context) error {
	return nil
}

func (m *MockBlockchain) Disconnect() error {
	return nil
}

func (m *MockBlockchain) IsConnected() bool {
	return true
}

// DID操作方法
func (m *MockBlockchain) RegisterDID(ctx context.Context, doc *types.DIDDocument) (*did.BlockchainTransaction, error) {
	return &did.BlockchainTransaction{
		Hash:      "test-hash",
		Timestamp: time.Now(),
	}, nil
}

func (m *MockBlockchain) UpdateDID(ctx context.Context, didStr string, doc *types.DIDDocument, proof *types.Proof) (*did.BlockchainTransaction, error) {
	return &did.BlockchainTransaction{
		Hash:      "test-hash",
		Timestamp: time.Now(),
	}, nil
}

func (m *MockBlockchain) RevokeDID(ctx context.Context, didStr string, proof *types.Proof) (*did.BlockchainTransaction, error) {
	return &did.BlockchainTransaction{
		Hash:      "test-hash",
		Timestamp: time.Now(),
	}, nil
}

func (m *MockBlockchain) GetDIDDocument(ctx context.Context, didStr string) (*types.DIDDocument, error) {
	return nil, nil
}

func (m *MockBlockchain) GetTransaction(ctx context.Context, txHash string) (*did.BlockchainTransaction, error) {
	return nil, nil
}

func (m *MockBlockchain) GetLatestBlock(ctx context.Context) (*did.BlockchainBlock, error) {
	return &did.BlockchainBlock{
		Hash:      "test-block-hash",
		Height:    1,
		Timestamp: time.Now(),
	}, nil
}

func (m *MockBlockchain) GetBlockHeight(ctx context.Context) (int64, error) {
	return 1, nil
}
