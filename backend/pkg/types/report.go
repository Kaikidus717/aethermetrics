package types

type OpcodeSummary struct {
	Count        uint64  `json:"count"`
	LegacyGas    uint64  `json:"legacyGas"`
	SimulatedGas uint64  `json:"simulatedGas"`
	Delta        int64   `json:"delta"`
	PctChange    float64 `json:"pctChange"`
}

type GasReport struct {
	LegacyTotal    uint64  `json:"legacyTotal"`
	SimulatedTotal uint64  `json:"simulatedTotal"`
	DeltaTotal     int64   `json:"deltaTotal"`
	PctTotalChange float64 `json:"pctTotalChange"`
}

type HeatmapSlot struct {
	Address       string `json:"address"`
	Slot          string `json:"slot"`
	ReadCount     uint64 `json:"readCount"`
	WriteCount    uint64 `json:"writeCount"`
	TotalAccesses uint64 `json:"totalAccesses"`
	IsContention  bool   `json:"isContention"`
	IsPreDeclared bool   `json:"isPreDeclared"`
}

type BALDetails struct {
	RLPHex     string                   `json:"rlpHex"`
	JSONFormat []map[string]interface{} `json:"jsonFormat"`
}

type SimulationReport struct {
	TxHash     string                   `json:"txHash"`
	GasSummary GasReport                `json:"gasSummary"`
	Opcodes    map[string]OpcodeSummary `json:"opcodes"`
	Heatmap    []HeatmapSlot            `json:"heatmap"`
	AccessList BALDetails               `json:"accessList"`
	Timestamp  int64                    `json:"timestamp"`
}
