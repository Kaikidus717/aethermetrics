package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"

	"aethermetrics/internal/bal"
	"aethermetrics/internal/gas"
	"aethermetrics/internal/rpc"
	pkgTypes "aethermetrics/pkg/types"
)

func main() {
	app := &cli.App{
		Name:    "aethermetrics",
		Usage:   "Local-first developer readiness suite and gas simulator for Ethereum Glamsterdam",
		Version: "1.0.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "rpc",
				Aliases: []string{"r"},
				Value:   "http://127.0.0.1:8545",
				Usage:   "Ethereum JSON-RPC HTTP endpoint",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "report.json",
				Usage:   "Path to output the JSON report",
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "trace",
				Usage:     "Trace a transaction and simulate Glamsterdam gas changes",
				ArgsUsage: "<tx_hash>",
				Action:    runTraceCommand,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "fuzz-predeclared",
						Value: true,
						Usage: "Assume storage slots were pre-declared in EIP-7928 BAL",
					},
				},
			},
			{
				Name:      "contention",
				Usage:     "Analyze a block for state contention across transactions",
				ArgsUsage: "<block_number>",
				Action:    runContentionCommand,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func connectAndTrace(c *cli.Context, txHash common.Hash) (*rpc.Client, *rpc.ExecutionTrace, error) {
	rpcEndpoint := c.String("rpc")

	fmt.Printf("⚡ Connecting to RPC Node: %s...\n", rpcEndpoint)
	client, err := rpc.Dial(rpcEndpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RPC node: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("📊 Running execution trace for %s...\n", txHash.Hex())
	trace, err := client.FetchTxTrace(ctx, txHash)
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("failed to fetch tx execution trace: %w", err)
	}

	return client, trace, nil
}

func runTraceCommand(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("missing required argument: transaction hash")
	}

	txHashStr := c.Args().First()
	txHash := common.HexToHash(txHashStr)
	outputPath := c.String("output")
	fuzzPredeclared := c.Bool("fuzz-predeclared")

	client, trace, err := connectAndTrace(c, txHash)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("🔍 Fetching transaction metadata...")
	gasLimit, value, err := client.FetchTxGasLimit(ctx, txHash)
	if err != nil {
		return fmt.Errorf("failed to fetch tx parameters: %w", err)
	}

	toAddr, err := client.FetchTxToAddress(ctx, txHash)
	if err != nil {
		return fmt.Errorf("failed to resolve tx target address: %w", err)
	}

	contractAddress := common.Address{}
	if toAddr != nil {
		contractAddress = *toAddr
	}

	fmt.Println("📈 Simulating Glamsterdam gas recalculations...")

	var accesses []bal.AccessEntry
	opcodeStats := make(map[string]pkgTypes.OpcodeSummary)
	heatmapMap := make(map[string]*pkgTypes.HeatmapSlot)

	var legacyGasAccumulated uint64
	var simulatedGasAccumulated uint64

	// Address stack for tracking active contract address
	addrStack := []common.Address{contractAddress}
	var lastCallTarget common.Address
	var lastCallOp string

	for _, logEntry := range trace.StructLogs {
		op := logEntry.Op
		opCost := logEntry.GasCost

		// Update address stack based on logEntry.Depth
		currDepth := logEntry.Depth
		if currDepth > len(addrStack) {
			// Depth increased: grow the stack
			if lastCallOp == gas.OpCALL || lastCallOp == gas.OpSTATICCALL {
				addrStack = append(addrStack, lastCallTarget)
			} else {
				// For DELEGATECALL, CALLCODE, active contract address remains the current one
				addrStack = append(addrStack, addrStack[len(addrStack)-1])
			}
		} else if currDepth < len(addrStack) {
			// Depth decreased: shrink the stack
			if currDepth > 0 {
				addrStack = addrStack[:currDepth]
			} else {
				addrStack = addrStack[:1]
			}
		}

		activeAddress := contractAddress
		if len(addrStack) > 0 {
			activeAddress = addrStack[len(addrStack)-1]
		}

		// Keep track of call opcode parameters for potential depth changes in the next step
		if op == gas.OpCALL || op == gas.OpSTATICCALL || op == gas.OpDELEGATECALL || op == gas.OpCALLCODE {
			lastCallOp = op
			if len(logEntry.Stack) >= 2 {
				targetU256 := logEntry.Stack[len(logEntry.Stack)-2]
				lastCallTarget = common.HexToAddress(targetU256.String())
			} else {
				lastCallTarget = common.Address{}
			}
		} else {
			lastCallOp = ""
		}

		isStorage := gas.IsStorageOp(op)
		isWarm := false
		isPreDeclared := fuzzPredeclared

		var coordStr string
		var slotHash common.Hash

		if isStorage && len(logEntry.Stack) > 0 {
			slotHexStr := logEntry.Stack[len(logEntry.Stack)-1].String()
			slotHash = common.HexToHash(slotHexStr)
			coordStr = fmt.Sprintf("%s:%s", activeAddress.Hex(), slotHash.Hex())

			accesses = append(accesses, bal.AccessEntry{
				Address: activeAddress,
				Slot:    slotHash,
				IsWrite: op == gas.OpSSTORE,
				Opcode:  op,
			})

			if _, exists := heatmapMap[coordStr]; exists {
				isWarm = true
			}
		}

		sim := gas.SimulateOpcodeGas(op, opCost, isWarm, isPreDeclared)
		legacyGasAccumulated += sim.LegacyCost
		simulatedGasAccumulated += sim.GlamsterdamCost

		summary := opcodeStats[op]
		summary.Count++
		summary.LegacyGas += sim.LegacyCost
		summary.SimulatedGas += sim.GlamsterdamCost
		summary.Delta += sim.Delta
		summary.Category = gas.GetOpcodeCategory(op)
		if summary.LegacyGas > 0 {
			summary.PctChange = (float64(summary.Delta) / float64(summary.LegacyGas)) * 100
		}
		opcodeStats[op] = summary

		if isStorage {
			node, exists := heatmapMap[coordStr]
			if !exists {
				node = &pkgTypes.HeatmapSlot{
					Address:       activeAddress.Hex(),
					Slot:          slotHash.Hex(),
					IsPreDeclared: isPreDeclared,
				}
				heatmapMap[coordStr] = node
			}
			node.TotalAccesses++
			if op == gas.OpSSTORE {
				node.WriteCount++
			} else {
				node.ReadCount++
			}
			if node.WriteCount > 0 && node.TotalAccesses > 1 {
				node.IsContention = true
			}
		}
	}

	fmt.Println("📦 Packing EIP-7928 Block-Level Access List...")
	balEntries := bal.BuildBAL(accesses)
	rlpBytes, err := bal.SerializeBAL(balEntries)
	if err != nil {
		return fmt.Errorf("failed to RLP encode Access List: %w", err)
	}

	jsonBAL := make([]map[string]interface{}, 0)
	for _, entry := range balEntries {
		slots := make([]string, 0)
		for _, s := range entry.Slots {
			slots = append(slots, s.Hex())
		}
		jsonBAL = append(jsonBAL, map[string]interface{}{
			"address": entry.Address.Hex(),
			"slots":   slots,
		})
	}

	heatmapSlice := make([]pkgTypes.HeatmapSlot, 0)
	for _, node := range heatmapMap {
		heatmapSlice = append(heatmapSlice, *node)
	}

	// Compute high-fidelity non-opcode gas to make gas simulation exact
	nonOpcodeGas := uint64(0)
	if trace.Gas > legacyGasAccumulated {
		nonOpcodeGas = trace.Gas - legacyGasAccumulated
	}
	legacyGasAccumulated += nonOpcodeGas
	simulatedGasAccumulated += nonOpcodeGas

	gasRep := gas.CalculateTotalGasDelta(legacyGasAccumulated, simulatedGasAccumulated)

	toHex := "0x0"
	if toAddr != nil {
		toHex = toAddr.Hex()
	}

	report := pkgTypes.SimulationReport{
		TxHash: txHash.Hex(),
		TxMeta: pkgTypes.TxMeta{
			GasLimit: gasLimit,
			Value:    value.String(),
			To:       toHex,
		},
		GasSummary: pkgTypes.GasReport{
			LegacyTotal:    gasRep.LegacyCost,
			SimulatedTotal: gasRep.GlamsterdamCost,
			DeltaTotal:     gasRep.Delta,
			PctTotalChange: gasRep.PercentChange,
		},
		Opcodes:    opcodeStats,
		Heatmap:    heatmapSlice,
		AccessList: pkgTypes.BALDetails{
			RLPHex:     "0x" + hex.EncodeToString(rlpBytes),
			JSONFormat: jsonBAL,
		},
		Timestamp: time.Now().Unix(),
	}

	return writeReport(outputPath, report)
}

func runContentionCommand(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("missing required argument: block number")
	}

	blockNumStr := c.Args().First()
	blockNum := new(big.Int)
	if _, ok := blockNum.SetString(blockNumStr, 10); !ok {
		return fmt.Errorf("invalid block number: %s", blockNumStr)
	}

	rpcEndpoint := c.String("rpc")
	outputPath := c.String("output")

	fmt.Printf("⚡ Connecting to RPC Node: %s...\n", rpcEndpoint)
	client, err := rpc.Dial(rpcEndpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC node: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fmt.Printf("📦 Fetching block %s transactions...\n", blockNum.String())
	txHashes, err := client.FetchBlockTxHashes(ctx, blockNum)
	if err != nil {
		return fmt.Errorf("failed to fetch block transactions: %w", err)
	}

	fmt.Printf("🔍 Tracing %d transactions...\n", len(txHashes))
	txsAccesses := make(map[common.Hash][]bal.AccessEntry)

	for i, txHash := range txHashes {
		trace, err := client.FetchTxTrace(ctx, txHash)
		if err != nil {
			fmt.Printf("⚠️  Skipping tx %s: %v\n", txHash.Hex(), err)
			continue
		}

		toAddr, _ := client.FetchTxToAddress(ctx, txHash)
		contractAddr := common.Address{}
		if toAddr != nil {
			contractAddr = *toAddr
		}

		var accesses []bal.AccessEntry
		addrStack := []common.Address{contractAddr}
		var lastCallTarget common.Address
		var lastCallOp string

		for _, logEntry := range trace.StructLogs {
			op := logEntry.Op
			currDepth := logEntry.Depth

			// Update address stack based on logEntry.Depth
			if currDepth > len(addrStack) {
				// Depth increased: grow the stack
				if lastCallOp == gas.OpCALL || lastCallOp == gas.OpSTATICCALL {
					addrStack = append(addrStack, lastCallTarget)
				} else {
					// For DELEGATECALL, CALLCODE, active contract address remains the current one
					addrStack = append(addrStack, addrStack[len(addrStack)-1])
				}
			} else if currDepth < len(addrStack) {
				// Depth decreased: shrink the stack
				if currDepth > 0 {
					addrStack = addrStack[:currDepth]
				} else {
					addrStack = addrStack[:1]
				}
			}

			activeAddress := contractAddr
			if len(addrStack) > 0 {
				activeAddress = addrStack[len(addrStack)-1]
			}

			// Keep track of call opcode parameters for potential depth changes in the next step
			if op == gas.OpCALL || op == gas.OpSTATICCALL || op == gas.OpDELEGATECALL || op == gas.OpCALLCODE {
				lastCallOp = op
				if len(logEntry.Stack) >= 2 {
					targetU256 := logEntry.Stack[len(logEntry.Stack)-2]
					lastCallTarget = common.HexToAddress(targetU256.String())
				} else {
					lastCallTarget = common.Address{}
				}
			} else {
				lastCallOp = ""
			}

			if !gas.IsStorageOp(op) {
				continue
			}
			if len(logEntry.Stack) == 0 {
				continue
			}

			slotHexStr := logEntry.Stack[len(logEntry.Stack)-1].String()
			slotHash := common.HexToHash(slotHexStr)

			accesses = append(accesses, bal.AccessEntry{
				Address: activeAddress,
				Slot:    slotHash,
				IsWrite: op == gas.OpSSTORE,
				Opcode:  op,
			})
		}

		txsAccesses[txHash] = accesses
		fmt.Printf("  [%d/%d] %s — %d storage ops\n", i+1, len(txHashes), txHash.Hex()[:16], len(accesses))
	}

	fmt.Println("🧬 Diagnosing state contention...")
	contentionNodes := bal.DiagnoseContention(txsAccesses)

	collisionCount := 0
	slots := make([]pkgTypes.ContentionSlot, 0)
	for _, node := range contentionNodes {
		txHashSet := make(map[string]bool)
		for _, t := range node.TxTouches {
			txHashSet[t.TxHash.Hex()] = true
		}
		txHashList := make([]string, 0)
		for h := range txHashSet {
			txHashList = append(txHashList, h)
		}

		slot := pkgTypes.ContentionSlot{
			Address:    node.Coordinate.Address.Hex(),
			Slot:       node.Coordinate.Slot.Hex(),
			Collision:  node.Collision,
			ReadCount:  node.ReadCount,
			WriteCount: node.WriteCount,
			TxHashes:   txHashList,
		}

		if node.Collision {
			collisionCount++
		}

		slots = append(slots, slot)
	}

	fmt.Println("📦 Building block-level BAL...")
	blockBALEntries := bal.BuildBlockBAL(txsAccesses)
	rlpBytes, err := bal.SerializeBAL(blockBALEntries)
	if err != nil {
		return fmt.Errorf("failed to RLP encode block BAL: %w", err)
	}

	jsonBAL := make([]map[string]interface{}, 0)
	for _, entry := range blockBALEntries {
		entrySlots := make([]string, 0)
		for _, s := range entry.Slots {
			entrySlots = append(entrySlots, s.Hex())
		}
		jsonBAL = append(jsonBAL, map[string]interface{}{
			"address": entry.Address.Hex(),
			"slots":   entrySlots,
		})
	}

	report := pkgTypes.ContentionReport{
		BlockNumber:    blockNum.Uint64(),
		TxCount:        len(txHashes),
		TotalSlots:     len(slots),
		CollisionCount: collisionCount,
		Slots:          slots,
		BlockBAL: pkgTypes.BALDetails{
			RLPHex:     "0x" + hex.EncodeToString(rlpBytes),
			JSONFormat: jsonBAL,
		},
		Timestamp: time.Now().Unix(),
	}

	fmt.Printf("✅ Contention analysis complete.\n")
	fmt.Printf("📊 %d total slots | %d collisions across %d transactions\n", len(slots), collisionCount, len(txHashes))

	return writeReport(outputPath, report)
}

func writeReport(outputPath string, report interface{}) error {
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report JSON: %w", err)
	}

	err = os.WriteFile(outputPath, reportJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	fmt.Printf("📄 Report saved to: %s\n", outputPath)
	return nil
}
