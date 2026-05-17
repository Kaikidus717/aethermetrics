package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
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
				Usage:   "Ethereum JSON-RPC HTTP endpoint (Geth, Anvil, Hardhat, etc.)",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "report.json",
				Usage:   "Path to output the simulation visualizer JSON report",
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "trace",
				Usage:     "Trace a historical transaction hash and simulate Glamsterdam changes",
				ArgsUsage: "<tx_hash>",
				Action:    runTraceCommand,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "fuzz-predeclared",
						Value: true,
						Usage: "Assume storage slots were successfully pre-declared in EIP-7928 BAL",
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runTraceCommand(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("missing required argument: transaction hash")
	}

	txHashStr := c.Args().First()
	txHash := common.HexToHash(txHashStr)
	rpcEndpoint := c.String("rpc")
	outputPath := c.String("output")
	fuzzPredeclared := c.Bool("fuzz-predeclared")

	fmt.Printf("⚡ Connecting to RPC Node: %s...\n", rpcEndpoint)
	client, err := rpc.Dial(rpcEndpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC node: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("🔍 Fetching Transaction details for %s...\n", txHash.Hex())
	originalGasLimit, originalValue, err := client.FetchTxGasLimit(ctx, txHash)
	if err != nil {
		return fmt.Errorf("failed to fetch tx parameters: %w", err)
	}

	fmt.Println("📊 Running execution tracing (debug_traceTransaction)...")
	trace, err := client.FetchTxTrace(ctx, txHash)
	if err != nil {
		return fmt.Errorf("failed to fetch tx execution logs: %w", err)
	}

	fmt.Println("📈 Simulating Glamsterdam Hard Fork gas recalculations...")

	var accesses []struct {
		Address common.Address
		Slot    common.Hash
	}

	opcodeStats := make(map[string]pkgTypes.OpcodeSummary)
	heatmapMap := make(map[string]*pkgTypes.HeatmapSlot)

	var legacyGasAccumulated uint64
	var simulatedGasAccumulated uint64

	for _, logEntry := range trace.StructLogs {
		op := logEntry.Op
		opCost := logEntry.GasCost

		isStorageOp := (op == "SLOAD" || op == "SSTORE")
		isWarm := false
		isPreDeclared := fuzzPredeclared

		var coordStr string
		var slotHash common.Hash
		contractAddress := common.HexToAddress("0x0000000000000000000000000000000000000000")

		if isStorageOp && len(logEntry.Stack) > 0 {
			slotHexStr := logEntry.Stack[len(logEntry.Stack)-1].String()
			slotHash = common.HexToHash(slotHexStr)
			coordStr = fmt.Sprintf("%s:%s", contractAddress.Hex(), slotHash.Hex())

			accesses = append(accesses, struct {
				Address common.Address
				Slot    common.Hash
			}{Address: contractAddress, Slot: slotHash})

			if _, exists := heatmapMap[coordStr]; exists {
				isWarm = true
			}
		}

		sim := gas.SimulateOpcodeGas(op, opCost, isWarm, isPreDeclared)
		legacyGasAccumulated += sim.LegacyCost
		simulatedGasAccumulated += sim.GlamsterdamCost

		summary, exists := opcodeStats[op]
		if !exists {
			summary = pkgTypes.OpcodeSummary{}
		}
		summary.Count++
		summary.LegacyGas += sim.LegacyCost
		summary.SimulatedGas += sim.GlamsterdamCost
		summary.Delta += sim.Delta
		opcodeStats[op] = summary

		if isStorageOp {
			node, exists := heatmapMap[coordStr]
			if !exists {
				node = &pkgTypes.HeatmapSlot{
					Address:       contractAddress.Hex(),
					Slot:          slotHash.Hex(),
					IsPreDeclared: isPreDeclared,
				}
				heatmapMap[coordStr] = node
			}
			node.TotalAccesses++
			if op == "SSTORE" {
				node.WriteCount++
			} else {
				node.ReadCount++
			}
			if node.WriteCount > 0 {
				node.IsContention = true
			}
		}
	}

	fmt.Println("📦 Packing EIP-7928 Block-Level Access List...")
	balEntries := bal.BuildDeterministicBAL(accesses)
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

	gasRep := gas.CalculateTotalGasDelta(legacyGasAccumulated, simulatedGasAccumulated)
	if simulatedGasAccumulated == 0 {
		gasRep = gas.CalculateTotalGasDelta(trace.Gas, trace.Gas)
	}

	_ = originalGasLimit
	_ = originalValue

	report := pkgTypes.SimulationReport{
		TxHash: txHash.Hex(),
		GasSummary: pkgTypes.GasReport{
			LegacyTotal:    gasRep.LegacyCost,
			SimulatedTotal: gasRep.GlamsterdamCost,
			DeltaTotal:     gasRep.Delta,
			PctTotalChange: gasRep.PercentChange,
		},
		Opcodes: opcodeStats,
		Heatmap: heatmapSlice,
		AccessList: pkgTypes.BALDetails{
			RLPHex:     "0x" + hex.EncodeToString(rlpBytes),
			JSONFormat: jsonBAL,
		},
		Timestamp: time.Now().Unix(),
	}

	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal simulation JSON: %w", err)
	}

	err = os.WriteFile(outputPath, reportJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write simulation report: %w", err)
	}

	fmt.Printf("✅ Success! Simulation Complete.\n")
	fmt.Printf("📄 Comparative analysis saved to: %s\n", outputPath)
	fmt.Printf("📉 Simulated Gas Shift: %d gas (%+.2f%%)\n", gasRep.Delta, gasRep.PercentChange)

	return nil
}
