package rpc

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type StructLog struct {
	Pc      uint64             `json:"pc"`
	Op      string             `json:"op"`
	Gas     uint64             `json:"gas"`
	GasCost uint64             `json:"gasCost"`
	Depth   int                `json:"depth"`
	Error   string             `json:"error,omitempty"`
	Stack   []hexutil.U256     `json:"stack"`
	Memory  []string           `json:"memory,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
}

type ExecutionTrace struct {
	Gas         uint64      `json:"gas"`
	Failed      bool        `json:"failed"`
	ReturnValue string      `json:"returnValue"`
	StructLogs  []StructLog `json:"structLogs"`
}

type Client struct {
	EthClient *ethclient.Client
	rpcClient *rpc.Client
	Endpoint  string
}

func Dial(endpoint string) (*Client, error) {
	rpcCli, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, err
	}

	ethCli := ethclient.NewClient(rpcCli)

	return &Client{
		EthClient: ethCli,
		rpcClient: rpcCli,
		Endpoint:  endpoint,
	}, nil
}

func (c *Client) Close() {
	if c.rpcClient != nil {
		c.rpcClient.Close()
	}
}

func (c *Client) FetchTxTrace(ctx context.Context, txHash common.Hash) (*ExecutionTrace, error) {
	if c.rpcClient == nil {
		return nil, errors.New("rpc client not initialized")
	}

	config := map[string]interface{}{
		"disableStorage": false,
		"disableMemory":  true,
		"disableStack":   false,
	}

	var trace ExecutionTrace
	err := c.rpcClient.CallContext(ctx, &trace, "debug_traceTransaction", txHash.Hex(), config)
	if err != nil {
		return nil, err
	}

	return &trace, nil
}

func (c *Client) FetchTxGasLimit(ctx context.Context, txHash common.Hash) (uint64, *big.Int, error) {
	tx, _, err := c.EthClient.TransactionByHash(ctx, txHash)
	if err != nil {
		return 0, nil, err
	}
	return tx.Gas(), tx.Value(), nil
}
