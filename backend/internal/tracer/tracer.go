package tracer

import (
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type AetherTracer struct {
	result *TraceResult
	mu     sync.Mutex
}

func NewAetherTracer(txHash common.Hash) *AetherTracer {
	return &AetherTracer{
		result: &TraceResult{
			TxHash:          txHash,
			OpcodeStats:     make(map[string]*OpcodeStat),
			StorageAccesses: make([]StorageAccess, 0),
			AccountTouches:  make([]AccountTouch, 0),
		},
	}
}

func (t *AetherTracer) GetResult() *TraceResult {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.result
}

func (t *AetherTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

func (t *AetherTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	opName := op.String()

	t.mu.Lock()
	defer t.mu.Unlock()

	stat, exists := t.result.OpcodeStats[opName]
	if !exists {
		stat = &OpcodeStat{}
		t.result.OpcodeStats[opName] = stat
	}
	stat.Count++
	stat.TotalGasCost += cost

	if op == vm.SLOAD || op == vm.SSTORE {
		stack := scope.Stack
		if stack.Len() > 0 {
			slotVal := stack.Back(0)
			slotHash := common.BigToHash(slotVal)

			access := StorageAccess{
				Address: scope.Contract.Address(),
				Slot:    slotHash,
				IsWrite: op == vm.SSTORE,
				Opcode:  opName,
				PC:      pc,
				GasCost: cost,
			}

			if op == vm.SSTORE && stack.Len() > 1 {
				val := stack.Back(1)
				access.Value = common.BigToHash(val)
			}

			t.result.StorageAccesses = append(t.result.StorageAccesses, access)
		}
	}

	if op == vm.CALL || op == vm.DELEGATECALL || op == vm.STATICCALL || op == vm.CALLCODE {
		stack := scope.Stack
		if stack.Len() >= 2 {
			toVal := stack.Back(1)
			toAddr := common.BigToAddress(toVal)

			touch := AccountTouch{
				From:    scope.Contract.Address(),
				To:      toAddr,
				Opcode:  opName,
				GasCost: cost,
			}

			if (op == vm.CALL || op == vm.CALLCODE) && stack.Len() >= 3 {
				touch.Value = stack.Back(2)
			}

			t.result.AccountTouches = append(t.result.AccountTouches, touch)
		}
	}
}

func (t *AetherTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}

func (t *AetherTracer) CaptureEnd(output []byte, gasUsed uint64, duration time.Duration, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.result.GasUsed = gasUsed
}

func (t *AetherTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

func (t *AetherTracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

func (t *AetherTracer) CaptureTxStart(gasLimit uint64) {
}

func (t *AetherTracer) CaptureTxEnd(restGas uint64) {
}
