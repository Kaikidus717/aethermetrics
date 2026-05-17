package bal

import (
	"bytes"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type BALEntry struct {
	Address common.Address
	Slots   []common.Hash
}

func BuildDeterministicBAL(accesses []struct {
	Address common.Address
	Slot    common.Hash
}) []BALEntry {
	grouped := make(map[common.Address]map[common.Hash]bool)
	for _, acc := range accesses {
		if _, exists := grouped[acc.Address]; !exists {
			grouped[acc.Address] = make(map[common.Hash]bool)
		}
		grouped[acc.Address][acc.Slot] = true
	}

	entries := make([]BALEntry, 0, len(grouped))
	for addr, slotsMap := range grouped {
		slots := make([]common.Hash, 0, len(slotsMap))
		for slot := range slotsMap {
			slots = append(slots, slot)
		}

		sort.Slice(slots, func(i, j int) bool {
			return bytes.Compare(slots[i].Bytes(), slots[j].Bytes()) < 0
		})

		entries = append(entries, BALEntry{
			Address: addr,
			Slots:   slots,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].Address.Bytes(), entries[j].Address.Bytes()) < 0
	})

	return entries
}

func SerializeBAL(entries []BALEntry) ([]byte, error) {
	return rlp.EncodeToBytes(entries)
}
