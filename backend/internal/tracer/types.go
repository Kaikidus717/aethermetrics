package tracer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type StorageAccess struct {
	Address common.Address `json:"address"`
	Slot    common.Hash    `json:"slot"`
	Value   common.Hash    `json:"value,omitempty"`
	IsWrite bool           `json:"isWrite"`
	Opcode  string         `json:"opcode"`
	PC      uint64         `json:"pc"`
	GasCost uint64         `json:"gasCost"`
	Depth   int            `json:"depth"`
}

type AccountTouch struct {
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Value   *big.Int       `json:"value,omitempty"`
	Opcode  string         `json:"opcode"`
	GasCost uint64         `json:"gasCost"`
	Depth   int            `json:"depth"`
}

type OpcodeStat struct {
	Count        uint64 `json:"count"`
	TotalGasCost uint64 `json:"totalGasCost"`
}

type TraceResult struct {
	TxHash          common.Hash            `json:"txHash"`
	GasUsed         uint64                 `json:"gasUsed"`
	StorageAccesses []StorageAccess        `json:"storageAccesses"`
	AccountTouches  []AccountTouch         `json:"accountTouches"`
	OpcodeStats     map[string]*OpcodeStat `json:"opcodeStats"`
}
