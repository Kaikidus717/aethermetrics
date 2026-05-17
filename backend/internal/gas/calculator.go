package gas

const (
	OpSLOAD      = "SLOAD"
	OpSSTORE     = "SSTORE"
	OpKECCAK256  = "KECCAK256"
	OpEXP        = "EXP"
	OpECRECOVER  = "ECRECOVER"
	OpADD        = "ADD"
	OpMUL        = "MUL"
	OpDIV        = "DIV"
)

type OpcodeRepricer struct {
	LegacyCost      uint64
	GlamsterdamCost uint64
}

var standardComputeRepricings = map[string]OpcodeRepricer{
	OpKECCAK256: {LegacyCost: 30, GlamsterdamCost: 65},
	OpEXP:       {LegacyCost: 10, GlamsterdamCost: 25},
	OpADD:       {LegacyCost: 3, GlamsterdamCost: 2},
	OpMUL:       {LegacyCost: 5, GlamsterdamCost: 3},
	OpDIV:       {LegacyCost: 5, GlamsterdamCost: 4},
}

type GasSimulationResult struct {
	LegacyCost      uint64  `json:"legacyCost"`
	GlamsterdamCost uint64  `json:"glamsterdamCost"`
	Delta           int64   `json:"delta"`
	PercentChange   float64 `json:"percentChange"`
}

func SimulateOpcodeGas(op string, baseCost uint64, isWarm bool, isPreDeclared bool) GasSimulationResult {
	var legacy, glamsterdam uint64

	switch op {
	case OpSLOAD:
		if isWarm {
			legacy = 100
		} else {
			legacy = 2100
		}

		if isPreDeclared {
			glamsterdam = 45
		} else {
			glamsterdam = 2800
		}

	case OpSSTORE:
		legacy = baseCost
		if isPreDeclared {
			glamsterdam = uint64(float64(baseCost) * 0.75)
			if glamsterdam < 100 {
				glamsterdam = 100
			}
		} else {
			glamsterdam = baseCost + 3500
		}

	default:
		if repricing, exists := standardComputeRepricings[op]; exists {
			legacy = repricing.LegacyCost
			glamsterdam = repricing.GlamsterdamCost
		} else {
			legacy = baseCost
			glamsterdam = baseCost
		}
	}

	delta := int64(glamsterdam) - int64(legacy)
	var pct float64
	if legacy > 0 {
		pct = (float64(delta) / float64(legacy)) * 100
	}

	return GasSimulationResult{
		LegacyCost:      legacy,
		GlamsterdamCost: glamsterdam,
		Delta:           delta,
		PercentChange:   pct,
	}
}

func CalculateTotalGasDelta(legacyGas uint64, simulatedGas uint64) GasSimulationResult {
	delta := int64(simulatedGas) - int64(legacyGas)
	var pct float64
	if legacyGas > 0 {
		pct = (float64(delta) / float64(legacyGas)) * 100
	}

	return GasSimulationResult{
		LegacyCost:      legacyGas,
		GlamsterdamCost: simulatedGas,
		Delta:           delta,
		PercentChange:   pct,
	}
}
