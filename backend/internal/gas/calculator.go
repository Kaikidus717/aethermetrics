package gas

const (
	OpSLOAD        = "SLOAD"
	OpSSTORE       = "SSTORE"
	OpKECCAK256    = "KECCAK256"
	OpSHA3         = "SHA3"
	OpEXP          = "EXP"
	OpADD          = "ADD"
	OpMUL          = "MUL"
	OpSUB          = "SUB"
	OpDIV          = "DIV"
	OpSDIV         = "SDIV"
	OpMOD          = "MOD"
	OpSMOD         = "SMOD"
	OpADDMOD       = "ADDMOD"
	OpMULMOD       = "MULMOD"
	OpSIGNEXTEND   = "SIGNEXTEND"
	OpLT           = "LT"
	OpGT           = "GT"
	OpSLT          = "SLT"
	OpSGT          = "SGT"
	OpEQ           = "EQ"
	OpISZERO       = "ISZERO"
	OpAND          = "AND"
	OpOR           = "OR"
	OpXOR          = "XOR"
	OpNOT          = "NOT"
	OpBYTE         = "BYTE"
	OpSHL          = "SHL"
	OpSHR          = "SHR"
	OpSAR          = "SAR"
	OpBALANCE      = "BALANCE"
	OpEXTCODESIZE  = "EXTCODESIZE"
	OpEXTCODEHASH  = "EXTCODEHASH"
	OpCALL         = "CALL"
	OpCALLCODE     = "CALLCODE"
	OpDELEGATECALL = "DELEGATECALL"
	OpSTATICCALL   = "STATICCALL"
	OpLOG0         = "LOG0"
	OpLOG1         = "LOG1"
	OpLOG2         = "LOG2"
	OpLOG3         = "LOG3"
	OpLOG4         = "LOG4"
	OpCREATE       = "CREATE"
	OpCREATE2      = "CREATE2"
	OpSELFDESTRUCT = "SELFDESTRUCT"
)

type OpcodeCategory string

const (
	CategoryArithmetic OpcodeCategory = "arithmetic"
	CategoryComparison OpcodeCategory = "comparison"
	CategoryBitwise    OpcodeCategory = "bitwise"
	CategoryCrypto     OpcodeCategory = "crypto"
	CategoryStorage    OpcodeCategory = "storage"
	CategoryCall       OpcodeCategory = "call"
	CategoryLog        OpcodeCategory = "log"
	CategoryAccount    OpcodeCategory = "account"
	CategorySystem     OpcodeCategory = "system"
	CategoryUnknown    OpcodeCategory = "unknown"
)

type OpcodeRepricer struct {
	LegacyCost      uint64
	GlamsterdamCost uint64
	Category        OpcodeCategory
}

var standardRepricings = map[string]OpcodeRepricer{
	OpADD:          {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryArithmetic},
	OpMUL:          {LegacyCost: 5, GlamsterdamCost: 3, Category: CategoryArithmetic},
	OpSUB:          {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryArithmetic},
	OpDIV:          {LegacyCost: 5, GlamsterdamCost: 4, Category: CategoryArithmetic},
	OpSDIV:         {LegacyCost: 5, GlamsterdamCost: 4, Category: CategoryArithmetic},
	OpMOD:          {LegacyCost: 5, GlamsterdamCost: 4, Category: CategoryArithmetic},
	OpSMOD:         {LegacyCost: 5, GlamsterdamCost: 4, Category: CategoryArithmetic},
	OpADDMOD:       {LegacyCost: 8, GlamsterdamCost: 5, Category: CategoryArithmetic},
	OpMULMOD:       {LegacyCost: 8, GlamsterdamCost: 5, Category: CategoryArithmetic},
	OpSIGNEXTEND:   {LegacyCost: 5, GlamsterdamCost: 3, Category: CategoryArithmetic},
	OpLT:           {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryComparison},
	OpGT:           {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryComparison},
	OpSLT:          {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryComparison},
	OpSGT:          {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryComparison},
	OpEQ:           {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryComparison},
	OpISZERO:       {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryComparison},
	OpAND:          {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryBitwise},
	OpOR:           {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryBitwise},
	OpXOR:          {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryBitwise},
	OpNOT:          {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryBitwise},
	OpBYTE:         {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryBitwise},
	OpSHL:          {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryBitwise},
	OpSHR:          {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryBitwise},
	OpSAR:          {LegacyCost: 3, GlamsterdamCost: 2, Category: CategoryBitwise},
	OpKECCAK256:    {LegacyCost: 30, GlamsterdamCost: 65, Category: CategoryCrypto},
	OpSHA3:         {LegacyCost: 30, GlamsterdamCost: 65, Category: CategoryCrypto},
	OpEXP:          {LegacyCost: 10, GlamsterdamCost: 25, Category: CategoryCrypto},
	OpBALANCE:      {LegacyCost: 100, GlamsterdamCost: 80, Category: CategoryAccount},
	OpEXTCODESIZE:  {LegacyCost: 100, GlamsterdamCost: 80, Category: CategoryAccount},
	OpEXTCODEHASH:  {LegacyCost: 100, GlamsterdamCost: 80, Category: CategoryAccount},
	OpCALL:         {LegacyCost: 100, GlamsterdamCost: 80, Category: CategoryCall},
	OpCALLCODE:     {LegacyCost: 100, GlamsterdamCost: 80, Category: CategoryCall},
	OpDELEGATECALL: {LegacyCost: 100, GlamsterdamCost: 80, Category: CategoryCall},
	OpSTATICCALL:   {LegacyCost: 100, GlamsterdamCost: 80, Category: CategoryCall},
	OpLOG0:         {LegacyCost: 375, GlamsterdamCost: 400, Category: CategoryLog},
	OpLOG1:         {LegacyCost: 750, GlamsterdamCost: 800, Category: CategoryLog},
	OpLOG2:         {LegacyCost: 1125, GlamsterdamCost: 1200, Category: CategoryLog},
	OpLOG3:         {LegacyCost: 1500, GlamsterdamCost: 1600, Category: CategoryLog},
	OpLOG4:         {LegacyCost: 1875, GlamsterdamCost: 2000, Category: CategoryLog},
	OpCREATE:       {LegacyCost: 32000, GlamsterdamCost: 34000, Category: CategorySystem},
	OpCREATE2:      {LegacyCost: 32000, GlamsterdamCost: 34000, Category: CategorySystem},
	OpSELFDESTRUCT: {LegacyCost: 5000, GlamsterdamCost: 7500, Category: CategorySystem},
}

type GasSimulationResult struct {
	LegacyCost      uint64  `json:"legacyCost"`
	GlamsterdamCost uint64  `json:"glamsterdamCost"`
	Delta           int64   `json:"delta"`
	PercentChange   float64 `json:"percentChange"`
}

func GetOpcodeCategory(op string) OpcodeCategory {
	if repricing, exists := standardRepricings[op]; exists {
		return repricing.Category
	}
	if op == OpSLOAD || op == OpSSTORE {
		return CategoryStorage
	}
	return CategoryUnknown
}

func IsStorageOp(op string) bool {
	return op == OpSLOAD || op == OpSSTORE
}

func IsCallOp(op string) bool {
	return op == OpCALL || op == OpCALLCODE || op == OpDELEGATECALL || op == OpSTATICCALL
}

func SimulateOpcodeGas(op string, baseCost uint64, isWarm bool, isPreDeclared bool) GasSimulationResult {
	var legacy, glamsterdam uint64

	switch op {
	case OpSLOAD:
		legacy = baseCost
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
		if repricing, exists := standardRepricings[op]; exists {
			legacy = repricing.LegacyCost
			glamsterdam = repricing.GlamsterdamCost
		} else {
			legacy = baseCost
			glamsterdam = baseCost
		}
	}

	return buildResult(legacy, glamsterdam)
}

func CalculateTotalGasDelta(legacyGas uint64, simulatedGas uint64) GasSimulationResult {
	return buildResult(legacyGas, simulatedGas)
}

func buildResult(legacy, glamsterdam uint64) GasSimulationResult {
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
