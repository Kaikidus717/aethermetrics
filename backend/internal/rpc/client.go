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
	Pc      uint64                          `json:"pc"`
	Op      string                          `json:"op"`
	Gas     uint64                          `json:"gas"`
	GasCost uint64                          `json:"gasCost"`
	Depth   int                             `json:"depth"`
	Error   string                          `json:"error,omitempty"`
	Stack   []hexutil.U256                  `json:"stack"`
	Memory  []string                        `json:"memory,omitempty"`
	Storage map[common.Hash]common.Hash     `json:"storage,omitempty"`
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

func (c *Client) FetchTxToAddress(ctx context.Context, txHash common.Hash) (*common.Address, error) {
	receipt, err := c.EthClient.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, err
	}

	if receipt.ContractAddress != (common.Address{}) {
		return &receipt.ContractAddress, nil
	}

	tx, _, err := c.EthClient.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}

	return tx.To(), nil
}

func (c *Client) FetchBlockTxHashes(ctx context.Context, blockNum *big.Int) ([]common.Hash, error) {
	block, err := c.EthClient.BlockByNumber(ctx, blockNum)
	if err != nil {
		return nil, err
	}

	hashes := make([]common.Hash, 0, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		hashes = append(hashes, tx.Hash())
	}
	return hashes, nil
}

func (c *Client) FetchBlockTraces(ctx context.Context, blockNum *big.Int) (map[common.Hash]*ExecutionTrace, error) {
	hashes, err := c.FetchBlockTxHashes(ctx, blockNum)
	if err != nil {
		return nil, err
	}

	traces := make(map[common.Hash]*ExecutionTrace, len(hashes))
	for _, h := range hashes {
		trace, err := c.FetchTxTrace(ctx, h)
		if err != nil {
			return nil, err
		}
		traces[h] = trace
	}
	return traces, nil
}
