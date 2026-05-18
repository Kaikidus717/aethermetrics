package bal

import (
	"github.com/ethereum/go-ethereum/common"
)

type StorageCoordinate struct {
	Address common.Address `json:"address"`
	Slot    common.Hash    `json:"slot"`
}

type TxAccessSummary struct {
	TxHash  common.Hash `json:"txHash"`
	IsWrite bool        `json:"isWrite"`
	Opcode  string      `json:"opcode"`
}

type ContentionNode struct {
	Coordinate StorageCoordinate `json:"coordinate"`
	TxTouches  []TxAccessSummary `json:"txTouches"`
	Collision  bool              `json:"collision"`
	ReadCount  int               `json:"readCount"`
	WriteCount int               `json:"writeCount"`
}

type AccessEntry struct {
	Address common.Address
	Slot    common.Hash
	IsWrite bool
	Opcode  string
}

func DiagnoseContention(txsAccesses map[common.Hash][]AccessEntry) []ContentionNode {
	slotRegistry := make(map[StorageCoordinate][]TxAccessSummary)

	for txHash, accesses := range txsAccesses {
		for _, acc := range accesses {
			coord := StorageCoordinate{
				Address: acc.Address,
				Slot:    acc.Slot,
			}

			summary := TxAccessSummary{
				TxHash:  txHash,
				IsWrite: acc.IsWrite,
				Opcode:  acc.Opcode,
			}

			slotRegistry[coord] = append(slotRegistry[coord], summary)
		}
	}

	contentionNodes := make([]ContentionNode, 0)
	for coord, touches := range slotRegistry {
		isCollision := false
		readCount := 0
		writeCount := 0

		if len(touches) > 1 {
			uniqueTxs := make(map[common.Hash]bool)

			for _, t := range touches {
				uniqueTxs[t.TxHash] = true
				if t.IsWrite {
					writeCount++
				} else {
					readCount++
				}
			}

			if len(uniqueTxs) > 1 && writeCount > 0 {
				isCollision = true
			}
		} else {
			for _, t := range touches {
				if t.IsWrite {
					writeCount++
				} else {
					readCount++
				}
			}
		}

		contentionNodes = append(contentionNodes, ContentionNode{
			Coordinate: coord,
			TxTouches:  touches,
			Collision:  isCollision,
			ReadCount:  readCount,
			WriteCount: writeCount,
		})
	}

	return contentionNodes
}
