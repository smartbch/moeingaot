package maot

import (
	"fmt"
	"os"
	"path"
	"strings"
)

var (
	OpTables      [11][256]OpTableEntry
	GasCostTable  [11][256]int = getGasCostTable()
	TraitsTable   [256]Traits  = getTraitsTable()
	FuncNameTable [256]string  = getFuncNameTable()
	TypeTable     [256]byte    = getInstrTypeTable()
)

const (
	NOP       = -2
	Undefined = -1

	StackOp         = byte(16)
	StateOnly       = byte(17)
	StateWithStatus = byte(18)
	Full            = byte(19)
	FullWithBreak   = byte(20)
	Jump            = byte(21)
	Inline          = byte(128)

	EVMC_FRONTIER          = 0
	EVMC_HOMESTEAD         = 1
	EVMC_TANGERINE_WHISTLE = 2
	EVMC_SPURIOUS_DRAGON   = 3
	EVMC_BYZANTIUM         = 4
	EVMC_CONSTANTINOPLE    = 5
	EVMC_PETERSBURG        = 6
	EVMC_ISTANBUL          = 7
	EVMC_BERLIN            = 8
	EVMC_LONDON            = 9
	EVMC_SHANGHAI          = 10

	OP_STOP       = 0x00
	OP_ADD        = 0x01
	OP_MUL        = 0x02
	OP_SUB        = 0x03
	OP_DIV        = 0x04
	OP_SDIV       = 0x05
	OP_MOD        = 0x06
	OP_SMOD       = 0x07
	OP_ADDMOD     = 0x08
	OP_MULMOD     = 0x09
	OP_EXP        = 0x0a
	OP_SIGNEXTEND = 0x0b

	OP_LT     = 0x10
	OP_GT     = 0x11
	OP_SLT    = 0x12
	OP_SGT    = 0x13
	OP_EQ     = 0x14
	OP_ISZERO = 0x15
	OP_AND    = 0x16
	OP_OR     = 0x17
	OP_XOR    = 0x18
	OP_NOT    = 0x19
	OP_BYTE   = 0x1a
	OP_SHL    = 0x1b
	OP_SHR    = 0x1c
	OP_SAR    = 0x1d

	OP_KECCAK256 = 0x20

	OP_ADDRESS        = 0x30
	OP_BALANCE        = 0x31
	OP_ORIGIN         = 0x32
	OP_CALLER         = 0x33
	OP_CALLVALUE      = 0x34
	OP_CALLDATALOAD   = 0x35
	OP_CALLDATASIZE   = 0x36
	OP_CALLDATACOPY   = 0x37
	OP_CODESIZE       = 0x38
	OP_CODECOPY       = 0x39
	OP_GASPRICE       = 0x3a
	OP_EXTCODESIZE    = 0x3b
	OP_EXTCODECOPY    = 0x3c
	OP_RETURNDATASIZE = 0x3d
	OP_RETURNDATACOPY = 0x3e
	OP_EXTCODEHASH    = 0x3f

	OP_BLOCKHASH   = 0x40
	OP_COINBASE    = 0x41
	OP_TIMESTAMP   = 0x42
	OP_NUMBER      = 0x43
	OP_DIFFICULTY  = 0x44
	OP_GASLIMIT    = 0x45
	OP_CHAINID     = 0x46
	OP_SELFBALANCE = 0x47
	OP_BASEFEE     = 0x48

	OP_POP         = 0x50
	OP_MLOAD       = 0x51
	OP_MSTORE      = 0x52
	OP_MSTORE8     = 0x53
	OP_SLOAD       = 0x54
	OP_SSTORE      = 0x55
	OP_JUMP        = 0x56
	OP_JUMPI       = 0x57
	OP_PC          = 0x58
	OP_MSIZE       = 0x59
	OP_GAS         = 0x5a
	OP_JUMPDEST    = 0x5b
	OPX_BEGINBLOCK = OP_JUMPDEST

	OP_PUSH1  = 0x60
	OP_PUSH2  = 0x61
	OP_PUSH3  = 0x62
	OP_PUSH4  = 0x63
	OP_PUSH5  = 0x64
	OP_PUSH6  = 0x65
	OP_PUSH7  = 0x66
	OP_PUSH8  = 0x67
	OP_PUSH9  = 0x68
	OP_PUSH10 = 0x69
	OP_PUSH11 = 0x6a
	OP_PUSH12 = 0x6b
	OP_PUSH13 = 0x6c
	OP_PUSH14 = 0x6d
	OP_PUSH15 = 0x6e
	OP_PUSH16 = 0x6f
	OP_PUSH17 = 0x70
	OP_PUSH18 = 0x71
	OP_PUSH19 = 0x72
	OP_PUSH20 = 0x73
	OP_PUSH21 = 0x74
	OP_PUSH22 = 0x75
	OP_PUSH23 = 0x76
	OP_PUSH24 = 0x77
	OP_PUSH25 = 0x78
	OP_PUSH26 = 0x79
	OP_PUSH27 = 0x7a
	OP_PUSH28 = 0x7b
	OP_PUSH29 = 0x7c
	OP_PUSH30 = 0x7d
	OP_PUSH31 = 0x7e
	OP_PUSH32 = 0x7f
	OP_DUP1   = 0x80
	OP_DUP2   = 0x81
	OP_DUP3   = 0x82
	OP_DUP4   = 0x83
	OP_DUP5   = 0x84
	OP_DUP6   = 0x85
	OP_DUP7   = 0x86
	OP_DUP8   = 0x87
	OP_DUP9   = 0x88
	OP_DUP10  = 0x89
	OP_DUP11  = 0x8a
	OP_DUP12  = 0x8b
	OP_DUP13  = 0x8c
	OP_DUP14  = 0x8d
	OP_DUP15  = 0x8e
	OP_DUP16  = 0x8f
	OP_SWAP1  = 0x90
	OP_SWAP2  = 0x91
	OP_SWAP3  = 0x92
	OP_SWAP4  = 0x93
	OP_SWAP5  = 0x94
	OP_SWAP6  = 0x95
	OP_SWAP7  = 0x96
	OP_SWAP8  = 0x97
	OP_SWAP9  = 0x98
	OP_SWAP10 = 0x99
	OP_SWAP11 = 0x9a
	OP_SWAP12 = 0x9b
	OP_SWAP13 = 0x9c
	OP_SWAP14 = 0x9d
	OP_SWAP15 = 0x9e
	OP_SWAP16 = 0x9f
	OP_LOG0   = 0xa0
	OP_LOG1   = 0xa1
	OP_LOG2   = 0xa2
	OP_LOG3   = 0xa3
	OP_LOG4   = 0xa4

	OP_CREATE       = 0xf0
	OP_CALL         = 0xf1
	OP_CALLCODE     = 0xf2
	OP_RETURN       = 0xf3
	OP_DELEGATECALL = 0xf4
	OP_CREATE2      = 0xf5

	OP_STATICCALL = 0xfa

	OP_REVERT       = 0xfd
	OP_INVALID      = 0xfe
	OP_SELFDESTRUCT = 0xff
)

type Traits struct {
	Name        string
	StackReq    int16
	StackChange int16
}

type OpTableEntry struct {
	FuncName    string
	GasCost     uint32
	StackReq    int16
	StackChange int16
}

func init() {
	for r := EVMC_FRONTIER; r <= EVMC_SHANGHAI; r++ {
		var table [256]OpTableEntry
		for i := 0; i < 256; i++ {
			cost := GasCostTable[r][i]
			if cost == Undefined {
				table[i].FuncName = "op_undefined"
			} else {
				table[i].FuncName = FuncNameTable[i]
				table[i].StackReq = TraitsTable[i].StackReq
				table[i].StackChange = TraitsTable[i].StackChange
			}
		}
		OpTables[r] = table
	}
}

func getGasCostTable() (table [11][256]int) {
	for i := 0; i < 256; i++ {
		table[EVMC_FRONTIER][i] = Undefined //?
	}
	table[EVMC_FRONTIER][OP_STOP] = 0
	table[EVMC_FRONTIER][OP_ADD] = 3
	table[EVMC_FRONTIER][OP_MUL] = 5
	table[EVMC_FRONTIER][OP_SUB] = 3
	table[EVMC_FRONTIER][OP_DIV] = 5
	table[EVMC_FRONTIER][OP_SDIV] = 5
	table[EVMC_FRONTIER][OP_MOD] = 5
	table[EVMC_FRONTIER][OP_SMOD] = 5
	table[EVMC_FRONTIER][OP_ADDMOD] = 8
	table[EVMC_FRONTIER][OP_MULMOD] = 8
	table[EVMC_FRONTIER][OP_EXP] = 10
	table[EVMC_FRONTIER][OP_SIGNEXTEND] = 5
	table[EVMC_FRONTIER][OP_LT] = 3
	table[EVMC_FRONTIER][OP_GT] = 3
	table[EVMC_FRONTIER][OP_SLT] = 3
	table[EVMC_FRONTIER][OP_SGT] = 3
	table[EVMC_FRONTIER][OP_EQ] = 3
	table[EVMC_FRONTIER][OP_ISZERO] = 3
	table[EVMC_FRONTIER][OP_AND] = 3
	table[EVMC_FRONTIER][OP_OR] = 3
	table[EVMC_FRONTIER][OP_XOR] = 3
	table[EVMC_FRONTIER][OP_NOT] = 3
	table[EVMC_FRONTIER][OP_BYTE] = 3
	table[EVMC_FRONTIER][OP_KECCAK256] = 30
	table[EVMC_FRONTIER][OP_ADDRESS] = 2
	table[EVMC_FRONTIER][OP_BALANCE] = 20
	table[EVMC_FRONTIER][OP_ORIGIN] = 2
	table[EVMC_FRONTIER][OP_CALLER] = 2
	table[EVMC_FRONTIER][OP_CALLVALUE] = 2
	table[EVMC_FRONTIER][OP_CALLDATALOAD] = 3
	table[EVMC_FRONTIER][OP_CALLDATASIZE] = 2
	table[EVMC_FRONTIER][OP_CALLDATACOPY] = 3
	table[EVMC_FRONTIER][OP_CODESIZE] = 2
	table[EVMC_FRONTIER][OP_CODECOPY] = 3
	table[EVMC_FRONTIER][OP_GASPRICE] = 2
	table[EVMC_FRONTIER][OP_EXTCODESIZE] = 20
	table[EVMC_FRONTIER][OP_EXTCODECOPY] = 20
	table[EVMC_FRONTIER][OP_BLOCKHASH] = 20
	table[EVMC_FRONTIER][OP_COINBASE] = 2
	table[EVMC_FRONTIER][OP_TIMESTAMP] = 2
	table[EVMC_FRONTIER][OP_NUMBER] = 2
	table[EVMC_FRONTIER][OP_DIFFICULTY] = 2
	table[EVMC_FRONTIER][OP_GASLIMIT] = 2
	table[EVMC_FRONTIER][OP_POP] = 2
	table[EVMC_FRONTIER][OP_MLOAD] = 3
	table[EVMC_FRONTIER][OP_MSTORE] = 3
	table[EVMC_FRONTIER][OP_MSTORE8] = 3
	table[EVMC_FRONTIER][OP_SLOAD] = 50
	table[EVMC_FRONTIER][OP_SSTORE] = 0
	table[EVMC_FRONTIER][OP_JUMP] = 8
	table[EVMC_FRONTIER][OP_JUMPI] = 10
	table[EVMC_FRONTIER][OP_PC] = 2
	table[EVMC_FRONTIER][OP_MSIZE] = 2
	table[EVMC_FRONTIER][OP_GAS] = 2
	table[EVMC_FRONTIER][OP_JUMPDEST] = 1
	for op := OP_PUSH1; op <= OP_PUSH32; op++ {
		table[EVMC_FRONTIER][op] = 3
	}
	for op := OP_DUP1; op <= OP_DUP16; op++ {
		table[EVMC_FRONTIER][op] = 3
	}
	for op := OP_SWAP1; op <= OP_SWAP16; op++ {
		table[EVMC_FRONTIER][op] = 3
	}
	for op := OP_LOG0; op <= OP_LOG4; op++ {
		table[EVMC_FRONTIER][op] = (op - OP_LOG0 + 1) * 375
	}
	table[EVMC_FRONTIER][OP_CREATE] = 32000
	table[EVMC_FRONTIER][OP_CALL] = 40
	table[EVMC_FRONTIER][OP_CALLCODE] = 40
	table[EVMC_FRONTIER][OP_RETURN] = 0
	table[EVMC_FRONTIER][OP_INVALID] = 0
	table[EVMC_FRONTIER][OP_SELFDESTRUCT] = 0

	table[EVMC_HOMESTEAD] = table[EVMC_FRONTIER]
	table[EVMC_HOMESTEAD][OP_DELEGATECALL] = 40

	table[EVMC_TANGERINE_WHISTLE] = table[EVMC_HOMESTEAD]
	table[EVMC_TANGERINE_WHISTLE][OP_BALANCE] = 400
	table[EVMC_TANGERINE_WHISTLE][OP_EXTCODESIZE] = 700
	table[EVMC_TANGERINE_WHISTLE][OP_EXTCODECOPY] = 700
	table[EVMC_TANGERINE_WHISTLE][OP_SLOAD] = 200
	table[EVMC_TANGERINE_WHISTLE][OP_CALL] = 700
	table[EVMC_TANGERINE_WHISTLE][OP_CALLCODE] = 700
	table[EVMC_TANGERINE_WHISTLE][OP_DELEGATECALL] = 700
	table[EVMC_TANGERINE_WHISTLE][OP_SELFDESTRUCT] = 5000

	table[EVMC_SPURIOUS_DRAGON] = table[EVMC_TANGERINE_WHISTLE]

	table[EVMC_BYZANTIUM] = table[EVMC_SPURIOUS_DRAGON]
	table[EVMC_BYZANTIUM][OP_RETURNDATASIZE] = 2
	table[EVMC_BYZANTIUM][OP_RETURNDATACOPY] = 3
	table[EVMC_BYZANTIUM][OP_STATICCALL] = 700
	table[EVMC_BYZANTIUM][OP_REVERT] = 0

	table[EVMC_CONSTANTINOPLE] = table[EVMC_BYZANTIUM]
	table[EVMC_CONSTANTINOPLE][OP_SHL] = 3
	table[EVMC_CONSTANTINOPLE][OP_SHR] = 3
	table[EVMC_CONSTANTINOPLE][OP_SAR] = 3
	table[EVMC_CONSTANTINOPLE][OP_EXTCODEHASH] = 400
	table[EVMC_CONSTANTINOPLE][OP_CREATE2] = 32000

	table[EVMC_PETERSBURG] = table[EVMC_CONSTANTINOPLE]

	table[EVMC_ISTANBUL] = table[EVMC_PETERSBURG]
	table[EVMC_ISTANBUL][OP_BALANCE] = 700
	table[EVMC_ISTANBUL][OP_CHAINID] = 2
	table[EVMC_ISTANBUL][OP_EXTCODEHASH] = 700
	table[EVMC_ISTANBUL][OP_SELFBALANCE] = 5
	table[EVMC_ISTANBUL][OP_SLOAD] = 800

	table[EVMC_BERLIN] = table[EVMC_ISTANBUL]
	table[EVMC_BERLIN][OP_EXTCODESIZE] = 100
	table[EVMC_BERLIN][OP_EXTCODECOPY] = 100
	table[EVMC_BERLIN][OP_EXTCODEHASH] = 100
	table[EVMC_BERLIN][OP_BALANCE] = 100
	table[EVMC_BERLIN][OP_CALL] = 100
	table[EVMC_BERLIN][OP_CALLCODE] = 100
	table[EVMC_BERLIN][OP_DELEGATECALL] = 100
	table[EVMC_BERLIN][OP_STATICCALL] = 100
	table[EVMC_BERLIN][OP_SLOAD] = 100

	table[EVMC_LONDON] = table[EVMC_BERLIN]
	table[EVMC_LONDON][OP_BASEFEE] = 2

	table[EVMC_SHANGHAI] = table[EVMC_LONDON]
	return
}

func getTraitsTable() (table [256]Traits) {
	table[OP_STOP] = Traits{"STOP", 0, 0}
	table[OP_ADD] = Traits{"ADD", 2, -1}
	table[OP_MUL] = Traits{"MUL", 2, -1}
	table[OP_SUB] = Traits{"SUB", 2, -1}
	table[OP_DIV] = Traits{"DIV", 2, -1}
	table[OP_SDIV] = Traits{"SDIV", 2, -1}
	table[OP_MOD] = Traits{"MOD", 2, -1}
	table[OP_SMOD] = Traits{"SMOD", 2, -1}
	table[OP_ADDMOD] = Traits{"ADDMOD", 3, -2}
	table[OP_MULMOD] = Traits{"MULMOD", 3, -2}
	table[OP_EXP] = Traits{"EXP", 2, -1}
	table[OP_SIGNEXTEND] = Traits{"SIGNEXTEND", 2, -1}

	table[OP_LT] = Traits{"LT", 2, -1}
	table[OP_GT] = Traits{"GT", 2, -1}
	table[OP_SLT] = Traits{"SLT", 2, -1}
	table[OP_SGT] = Traits{"SGT", 2, -1}
	table[OP_EQ] = Traits{"EQ", 2, -1}
	table[OP_ISZERO] = Traits{"ISZERO", 1, 0}
	table[OP_AND] = Traits{"AND", 2, -1}
	table[OP_OR] = Traits{"OR", 2, -1}
	table[OP_XOR] = Traits{"XOR", 2, -1}
	table[OP_NOT] = Traits{"NOT", 1, 0}
	table[OP_BYTE] = Traits{"BYTE", 2, -1}
	table[OP_SHL] = Traits{"SHL", 2, -1}
	table[OP_SHR] = Traits{"SHR", 2, -1}
	table[OP_SAR] = Traits{"SAR", 2, -1}

	table[OP_KECCAK256] = Traits{"KECCAK256", 2, -1}

	table[OP_ADDRESS] = Traits{"ADDRESS", 0, 1}
	table[OP_BALANCE] = Traits{"BALANCE", 1, 0}
	table[OP_ORIGIN] = Traits{"ORIGIN", 0, 1}
	table[OP_CALLER] = Traits{"CALLER", 0, 1}
	table[OP_CALLVALUE] = Traits{"CALLVALUE", 0, 1}
	table[OP_CALLDATALOAD] = Traits{"CALLDATALOAD", 1, 0}
	table[OP_CALLDATASIZE] = Traits{"CALLDATASIZE", 0, 1}
	table[OP_CALLDATACOPY] = Traits{"CALLDATACOPY", 3, -3}
	table[OP_CODESIZE] = Traits{"CODESIZE", 0, 1}
	table[OP_CODECOPY] = Traits{"CODECOPY", 3, -3}
	table[OP_GASPRICE] = Traits{"GASPRICE", 0, 1}
	table[OP_EXTCODESIZE] = Traits{"EXTCODESIZE", 1, 0}
	table[OP_EXTCODECOPY] = Traits{"EXTCODECOPY", 4, -4}
	table[OP_RETURNDATASIZE] = Traits{"RETURNDATASIZE", 0, 1}
	table[OP_RETURNDATACOPY] = Traits{"RETURNDATACOPY", 3, -3}
	table[OP_EXTCODEHASH] = Traits{"EXTCODEHASH", 1, 0}

	table[OP_BLOCKHASH] = Traits{"BLOCKHASH", 1, 0}
	table[OP_COINBASE] = Traits{"COINBASE", 0, 1}
	table[OP_TIMESTAMP] = Traits{"TIMESTAMP", 0, 1}
	table[OP_NUMBER] = Traits{"NUMBER", 0, 1}
	table[OP_DIFFICULTY] = Traits{"DIFFICULTY", 0, 1}
	table[OP_GASLIMIT] = Traits{"GASLIMIT", 0, 1}
	table[OP_CHAINID] = Traits{"CHAINID", 0, 1}
	table[OP_SELFBALANCE] = Traits{"SELFBALANCE", 0, 1}
	table[OP_BASEFEE] = Traits{"BASEFEE", 0, 1}

	table[OP_POP] = Traits{"POP", 1, -1}
	table[OP_MLOAD] = Traits{"MLOAD", 1, 0}
	table[OP_MSTORE] = Traits{"MSTORE", 2, -2}
	table[OP_MSTORE8] = Traits{"MSTORE8", 2, -2}
	table[OP_SLOAD] = Traits{"SLOAD", 1, 0}
	table[OP_SSTORE] = Traits{"SSTORE", 2, -2}
	table[OP_JUMP] = Traits{"JUMP", 1, -1}
	table[OP_JUMPI] = Traits{"JUMPI", 2, -2}
	table[OP_PC] = Traits{"PC", 0, 1}
	table[OP_MSIZE] = Traits{"MSIZE", 0, 1}
	table[OP_GAS] = Traits{"GAS", 0, 1}
	table[OP_JUMPDEST] = Traits{"BEGINBLOCK", 0, 0}

	table[OP_PUSH1] = Traits{"PUSH1", 0, 1}
	table[OP_PUSH2] = Traits{"PUSH2", 0, 1}
	table[OP_PUSH3] = Traits{"PUSH3", 0, 1}
	table[OP_PUSH4] = Traits{"PUSH4", 0, 1}
	table[OP_PUSH5] = Traits{"PUSH5", 0, 1}
	table[OP_PUSH6] = Traits{"PUSH6", 0, 1}
	table[OP_PUSH7] = Traits{"PUSH7", 0, 1}
	table[OP_PUSH8] = Traits{"PUSH8", 0, 1}
	table[OP_PUSH9] = Traits{"PUSH9", 0, 1}
	table[OP_PUSH10] = Traits{"PUSH10", 0, 1}
	table[OP_PUSH11] = Traits{"PUSH11", 0, 1}
	table[OP_PUSH12] = Traits{"PUSH12", 0, 1}
	table[OP_PUSH13] = Traits{"PUSH13", 0, 1}
	table[OP_PUSH14] = Traits{"PUSH14", 0, 1}
	table[OP_PUSH15] = Traits{"PUSH15", 0, 1}
	table[OP_PUSH16] = Traits{"PUSH16", 0, 1}
	table[OP_PUSH17] = Traits{"PUSH17", 0, 1}
	table[OP_PUSH18] = Traits{"PUSH18", 0, 1}
	table[OP_PUSH19] = Traits{"PUSH19", 0, 1}
	table[OP_PUSH20] = Traits{"PUSH20", 0, 1}
	table[OP_PUSH21] = Traits{"PUSH21", 0, 1}
	table[OP_PUSH22] = Traits{"PUSH22", 0, 1}
	table[OP_PUSH23] = Traits{"PUSH23", 0, 1}
	table[OP_PUSH24] = Traits{"PUSH24", 0, 1}
	table[OP_PUSH25] = Traits{"PUSH25", 0, 1}
	table[OP_PUSH26] = Traits{"PUSH26", 0, 1}
	table[OP_PUSH27] = Traits{"PUSH27", 0, 1}
	table[OP_PUSH28] = Traits{"PUSH28", 0, 1}
	table[OP_PUSH29] = Traits{"PUSH29", 0, 1}
	table[OP_PUSH30] = Traits{"PUSH30", 0, 1}
	table[OP_PUSH31] = Traits{"PUSH31", 0, 1}
	table[OP_PUSH32] = Traits{"PUSH32", 0, 1}

	table[OP_DUP1] = Traits{"DUP1", 1, 1}
	table[OP_DUP2] = Traits{"DUP2", 2, 1}
	table[OP_DUP3] = Traits{"DUP3", 3, 1}
	table[OP_DUP4] = Traits{"DUP4", 4, 1}
	table[OP_DUP5] = Traits{"DUP5", 5, 1}
	table[OP_DUP6] = Traits{"DUP6", 6, 1}
	table[OP_DUP7] = Traits{"DUP7", 7, 1}
	table[OP_DUP8] = Traits{"DUP8", 8, 1}
	table[OP_DUP9] = Traits{"DUP9", 9, 1}
	table[OP_DUP10] = Traits{"DUP10", 10, 1}
	table[OP_DUP11] = Traits{"DUP11", 11, 1}
	table[OP_DUP12] = Traits{"DUP12", 12, 1}
	table[OP_DUP13] = Traits{"DUP13", 13, 1}
	table[OP_DUP14] = Traits{"DUP14", 14, 1}
	table[OP_DUP15] = Traits{"DUP15", 15, 1}
	table[OP_DUP16] = Traits{"DUP16", 16, 1}

	table[OP_SWAP1] = Traits{"SWAP1", 2, 0}
	table[OP_SWAP2] = Traits{"SWAP2", 3, 0}
	table[OP_SWAP3] = Traits{"SWAP3", 4, 0}
	table[OP_SWAP4] = Traits{"SWAP4", 5, 0}
	table[OP_SWAP5] = Traits{"SWAP5", 6, 0}
	table[OP_SWAP6] = Traits{"SWAP6", 7, 0}
	table[OP_SWAP7] = Traits{"SWAP7", 8, 0}
	table[OP_SWAP8] = Traits{"SWAP8", 9, 0}
	table[OP_SWAP9] = Traits{"SWAP9", 10, 0}
	table[OP_SWAP10] = Traits{"SWAP10", 11, 0}
	table[OP_SWAP11] = Traits{"SWAP11", 12, 0}
	table[OP_SWAP12] = Traits{"SWAP12", 13, 0}
	table[OP_SWAP13] = Traits{"SWAP13", 14, 0}
	table[OP_SWAP14] = Traits{"SWAP14", 15, 0}
	table[OP_SWAP15] = Traits{"SWAP15", 16, 0}
	table[OP_SWAP16] = Traits{"SWAP16", 17, 0}

	table[OP_LOG0] = Traits{"LOG0", 2, -2}
	table[OP_LOG1] = Traits{"LOG1", 3, -3}
	table[OP_LOG2] = Traits{"LOG2", 4, -4}
	table[OP_LOG3] = Traits{"LOG3", 5, -5}
	table[OP_LOG4] = Traits{"LOG4", 6, -6}

	table[OP_CREATE] = Traits{"CREATE", 3, -2}
	table[OP_CALL] = Traits{"CALL", 7, -6}
	table[OP_CALLCODE] = Traits{"CALLCODE", 7, -6}
	table[OP_RETURN] = Traits{"RETURN", 2, -2}
	table[OP_DELEGATECALL] = Traits{"DELEGATECALL", 6, -5}
	table[OP_CREATE2] = Traits{"CREATE2", 4, -3}
	table[OP_STATICCALL] = Traits{"STATICCALL", 6, -5}
	table[OP_REVERT] = Traits{"REVERT", 2, -2}
	table[OP_INVALID] = Traits{"INVALID", 0, 0}
	table[OP_SELFDESTRUCT] = Traits{"SELFDESTRUCT", 1, -1}
	return
}

func getFuncNameTable() (table [256]string) {
	table[OP_STOP] = "op_stop"
	table[OP_ADD] = "op<evmone::add>"
	table[OP_MUL] = "op<evmone::mul>"
	table[OP_SUB] = "op<evmone::sub>"
	table[OP_DIV] = "op<evmone::div>"
	table[OP_SDIV] = "op<evmone::sdiv>"
	table[OP_MOD] = "op<evmone::mod>"
	table[OP_SMOD] = "op<evmone::smod>"
	table[OP_ADDMOD] = "op<evmone::addmod>"
	table[OP_MULMOD] = "op<evmone::mulmod>"
	table[OP_EXP] = "op<evmone::exp>"
	table[OP_SIGNEXTEND] = "op<evmone::signextend>"
	table[OP_LT] = "op<evmone::lt>"
	table[OP_GT] = "op<evmone::gt>"
	table[OP_SLT] = "op<evmone::slt>"
	table[OP_SGT] = "op<evmone::sgt>"
	table[OP_EQ] = "op<evmone::eq>"
	table[OP_ISZERO] = "op<evmone::iszero>"
	table[OP_AND] = "op<evmone::and_>"
	table[OP_OR] = "op<evmone::or_>"
	table[OP_XOR] = "op<evmone::xor_>"
	table[OP_NOT] = "op<evmone::not_>"
	table[OP_BYTE] = "op<evmone::byte>"
	table[OP_SHL] = "op<evmone::shl>"
	table[OP_SHR] = "op<evmone::shr>"
	table[OP_SAR] = "op<evmone::sar>"

	table[OP_KECCAK256] = "op<evmone::keccak256>"

	table[OP_ADDRESS] = "op<evmone::address>"
	table[OP_BALANCE] = "op<evmone::balance>"
	table[OP_ORIGIN] = "op<evmone::origin>"
	table[OP_CALLER] = "op<evmone::caller>"
	table[OP_CALLVALUE] = "op<evmone::callvalue>"
	table[OP_CALLDATALOAD] = "op<evmone::calldataload>"
	table[OP_CALLDATASIZE] = "op<evmone::calldatasize>"
	table[OP_CALLDATACOPY] = "op<evmone::calldatacopy>"
	table[OP_CODESIZE] = "op<evmone::codesize>"
	table[OP_CODECOPY] = "op<evmone::codecopy>"
	table[OP_GASPRICE] = "op<evmone::gasprice>"
	table[OP_EXTCODESIZE] = "op<evmone::extcodesize>"
	table[OP_EXTCODECOPY] = "op<evmone::extcodecopy>"
	table[OP_RETURNDATASIZE] = "op<evmone::returndatasize>"
	table[OP_RETURNDATACOPY] = "op<evmone::returndatacopy>"
	table[OP_EXTCODEHASH] = "op<evmone::extcodehash>"
	table[OP_BLOCKHASH] = "op<evmone::blockhash>"
	table[OP_COINBASE] = "op<evmone::coinbase>"
	table[OP_TIMESTAMP] = "op<evmone::timestamp>"
	table[OP_NUMBER] = "op<evmone::number>"
	table[OP_DIFFICULTY] = "op<evmone::difficulty>"
	table[OP_GASLIMIT] = "op<evmone::gaslimit>"
	table[OP_CHAINID] = "op<evmone::chainid>"
	table[OP_SELFBALANCE] = "op<evmone::selfbalance>"
	table[OP_BASEFEE] = "op<evmone::basefee>"

	table[OP_POP] = "op<evmone::pop>"
	table[OP_MLOAD] = "op<evmone::mload>"
	table[OP_MSTORE] = "op<evmone::mstore>"
	table[OP_MSTORE8] = "op<evmone::mstore8>"
	table[OP_SLOAD] = "op<evmone::sload>"
	table[OP_SSTORE] = "op_sstore"
	table[OP_JUMP] = "op_jump"
	table[OP_JUMPI] = "op_jumpi"
	table[OP_PC] = "op_pc"
	table[OP_MSIZE] = "op<evmone::msize>"
	table[OP_GAS] = "op_gas"
	table[OPX_BEGINBLOCK] = "opx_beginblock"

	for op := OP_PUSH1; op <= OP_PUSH8; op++ {
		table[op] = "op_push_small"
	}
	for op := OP_PUSH9; op <= OP_PUSH32; op++ {
		table[op] = "op_push_full"
	}

	table[OP_DUP1] = "op<evmone::dup<1>>"
	table[OP_DUP2] = "op<evmone::dup<2>>"
	table[OP_DUP3] = "op<evmone::dup<3>>"
	table[OP_DUP4] = "op<evmone::dup<4>>"
	table[OP_DUP5] = "op<evmone::dup<5>>"
	table[OP_DUP6] = "op<evmone::dup<6>>"
	table[OP_DUP7] = "op<evmone::dup<7>>"
	table[OP_DUP8] = "op<evmone::dup<8>>"
	table[OP_DUP9] = "op<evmone::dup<9>>"
	table[OP_DUP10] = "op<evmone::dup<10>>"
	table[OP_DUP11] = "op<evmone::dup<11>>"
	table[OP_DUP12] = "op<evmone::dup<12>>"
	table[OP_DUP13] = "op<evmone::dup<13>>"
	table[OP_DUP14] = "op<evmone::dup<14>>"
	table[OP_DUP15] = "op<evmone::dup<15>>"
	table[OP_DUP16] = "op<evmone::dup<16>>"

	table[OP_SWAP1] = "op<evmone::swap<1>>"
	table[OP_SWAP2] = "op<evmone::swap<2>>"
	table[OP_SWAP3] = "op<evmone::swap<3>>"
	table[OP_SWAP4] = "op<evmone::swap<4>>"
	table[OP_SWAP5] = "op<evmone::swap<5>>"
	table[OP_SWAP6] = "op<evmone::swap<6>>"
	table[OP_SWAP7] = "op<evmone::swap<7>>"
	table[OP_SWAP8] = "op<evmone::swap<8>>"
	table[OP_SWAP9] = "op<evmone::swap<9>>"
	table[OP_SWAP10] = "op<evmone::swap<10>>"
	table[OP_SWAP11] = "op<evmone::swap<11>>"
	table[OP_SWAP12] = "op<evmone::swap<12>>"
	table[OP_SWAP13] = "op<evmone::swap<13>>"
	table[OP_SWAP14] = "op<evmone::swap<14>>"
	table[OP_SWAP15] = "op<evmone::swap<15>>"
	table[OP_SWAP16] = "op<evmone::swap<16>>"

	table[OP_LOG0] = "op<evmone::log<0>>"
	table[OP_LOG1] = "op<evmone::log<1>>"
	table[OP_LOG2] = "op<evmone::log<2>>"
	table[OP_LOG3] = "op<evmone::log<3>>"
	table[OP_LOG4] = "op<evmone::log<4>>"

	table[OP_CREATE] = "op_create<EVMC_CREATE>"
	table[OP_CALL] = "op_call<EVMC_CALL>"
	table[OP_CALLCODE] = "op_call<EVMC_CALLCODE>"
	table[OP_RETURN] = "op_return<EVMC_SUCCESS>"
	table[OP_DELEGATECALL] = "op_call<EVMC_DELEGATECALL>"
	table[OP_CREATE2] = "op_create<EVMC_CREATE2>"
	table[OP_STATICCALL] = "op_call<EVMC_CALL, true>"
	table[OP_REVERT] = "op_return<EVMC_REVERT>"
	table[OP_INVALID] = "op_invalid"
	table[OP_SELFDESTRUCT] = "op_selfdestruct"
	return
}

func getInstrTypeTable() (table [256]byte) {
	table[OP_STOP] = FullWithBreak
	table[OP_ADD] = StackOp | Inline
	table[OP_MUL] = StackOp | Inline
	table[OP_SUB] = StackOp | Inline
	table[OP_DIV] = StackOp
	table[OP_SDIV] = StackOp
	table[OP_MOD] = StackOp
	table[OP_SMOD] = StackOp
	table[OP_ADDMOD] = StackOp
	table[OP_MULMOD] = StackOp
	table[OP_EXP] = StackOp
	table[OP_SIGNEXTEND] = StackOp | Inline
	table[OP_LT] = StackOp | Inline
	table[OP_GT] = StackOp | Inline
	table[OP_SLT] = StackOp | Inline
	table[OP_SGT] = StackOp | Inline
	table[OP_EQ] = StackOp | Inline
	table[OP_ISZERO] = StackOp | Inline
	table[OP_AND] = StackOp | Inline
	table[OP_OR] = StackOp | Inline
	table[OP_XOR] = StackOp | Inline
	table[OP_NOT] = StackOp | Inline
	table[OP_BYTE] = StackOp | Inline
	table[OP_SHL] = StackOp | Inline
	table[OP_SHR] = StackOp | Inline
	table[OP_SAR] = StackOp | Inline
	table[OP_KECCAK256] = StateWithStatus
	table[OP_ADDRESS] = StateOnly | Inline
	table[OP_BALANCE] = StateWithStatus
	table[OP_ORIGIN] = StateOnly | Inline
	table[OP_CALLER] = StateOnly | Inline
	table[OP_CALLVALUE] = StateOnly | Inline
	table[OP_CALLDATALOAD] = StateOnly | Inline
	table[OP_CALLDATASIZE] = StateOnly | Inline
	table[OP_CALLDATACOPY] = StateWithStatus | Inline
	table[OP_CODESIZE] = StateOnly | Inline
	table[OP_CODECOPY] = StateWithStatus | Inline
	table[OP_GASPRICE] = StateOnly | Inline
	table[OP_EXTCODESIZE] = StateWithStatus
	table[OP_EXTCODECOPY] = StateWithStatus
	table[OP_RETURNDATASIZE] = StateOnly | Inline
	table[OP_RETURNDATACOPY] = StateWithStatus | Inline
	table[OP_EXTCODEHASH] = StateWithStatus
	table[OP_BLOCKHASH] = StateOnly
	table[OP_COINBASE] = StateOnly | Inline
	table[OP_TIMESTAMP] = StateOnly | Inline
	table[OP_NUMBER] = StateOnly | Inline
	table[OP_DIFFICULTY] = StateOnly | Inline
	table[OP_GASLIMIT] = StateOnly | Inline
	table[OP_CHAINID] = StateOnly | Inline
	table[OP_SELFBALANCE] = StateOnly
	table[OP_BASEFEE] = StateOnly
	table[OP_POP] = StackOp | Inline
	table[OP_MLOAD] = StateWithStatus | Inline
	table[OP_MSTORE] = StateWithStatus | Inline
	table[OP_MSTORE8] = StateWithStatus | Inline
	table[OP_SLOAD] = StateWithStatus
	table[OP_SSTORE] = FullWithBreak
	table[OP_JUMP] = Jump
	table[OP_JUMPI] = Jump
	table[OP_PC] = Full | Inline
	table[OP_MSIZE] = StateOnly | Inline
	table[OP_GAS] = Full
	table[OP_JUMPDEST] = FullWithBreak
	table[OP_LOG0] = StateWithStatus
	table[OP_LOG1] = StateWithStatus
	table[OP_LOG2] = StateWithStatus
	table[OP_LOG3] = StateWithStatus
	table[OP_LOG4] = StateWithStatus
	table[OP_CREATE] = Full
	table[OP_CALL] = Full
	table[OP_CALLCODE] = Full
	table[OP_RETURN] = Full
	table[OP_DELEGATECALL] = Full
	table[OP_CREATE2] = Full
	table[OP_STATICCALL] = Full
	table[OP_REVERT] = Full
	table[OP_INVALID] = Full
	table[OP_SELFDESTRUCT] = Full
	return
}



func DumpInstrExeFiles(dir string) {
	opTbl := OpTables[EVMC_ISTANBUL]
	hF := []string{`#pragma once
#include "analysis.hpp"
#include "instructions.hpp"

namespace evmone
{
template <void InstrFn(Stack&)>
inline const instruction* op(const instruction* instr, AdvancedExecutionState& state) noexcept
{
    InstrFn(state.stack);
    return ++instr;
}

template <void InstrFn(ExecutionState&)>
inline const instruction* op(const instruction* instr, AdvancedExecutionState& state) noexcept
{
    InstrFn(state);
    return ++instr;
}

template <evmc_status_code InstrFn(ExecutionState&)>
inline const instruction* op(const instruction* instr, AdvancedExecutionState& state) noexcept
{
    const auto status_code = InstrFn(state);
    if (status_code != EVMC_SUCCESS)
        return state.exit(status_code);
    return ++instr;
}

inline const instruction* op_pc(const instruction* instr, AdvancedExecutionState& state) noexcept
{
    state.stack.push(instr->arg.number);
    return ++instr;
}

inline const instruction* op_push_small(const instruction* instr, AdvancedExecutionState& state) noexcept
{
    state.stack.push(instr->arg.small_push_value);
    return ++instr;
}

inline const instruction* op_push_full(const instruction* instr, AdvancedExecutionState& state) noexcept
{
    state.stack.push(*instr->arg.push_value);
    return ++instr;
}

inline bool test_jump_cond(AdvancedExecutionState& state) noexcept {
	const auto top = state.stack.pop();
	return top != 0;
}
inline size_t pop_target_pc(AdvancedExecutionState& state) noexcept {
	const auto pc = state.stack.pop();
	return static_cast<size_t>(pc);
}
inline size_t get_target_pc(AdvancedExecutionState& state) noexcept {
	const auto pc = state.stack.pop();
	const auto cond = state.stack.pop();
	if(cond != 0) return ~size_t(0);
	return static_cast<size_t>(pc);
}

const instruction* op_stop(const instruction*, AdvancedExecutionState& state) noexcept;
const instruction* op_invalid(const instruction*, AdvancedExecutionState& state) noexcept;
const instruction* op_sstore(const instruction* instr, AdvancedExecutionState& state) noexcept;
const instruction* op_gas(const instruction* instr, AdvancedExecutionState& state) noexcept;
template <evmc_status_code status_code>
const instruction* op_return(const instruction*, AdvancedExecutionState& state) noexcept;
template <evmc_call_kind Kind, bool Static = false>
const instruction* op_call(const instruction* instr, AdvancedExecutionState& state) noexcept;
template <evmc_call_kind Kind>
const instruction* op_create(const instruction* instr, AdvancedExecutionState& state) noexcept;
const instruction* op_undefined(const instruction*, AdvancedExecutionState& state) noexcept;
const instruction* op_selfdestruct(const instruction*, AdvancedExecutionState& state) noexcept;
const instruction* opx_beginblock(const instruction* instr, AdvancedExecutionState& state) noexcept;
}

inline evmone::instruction instr_from_block(uint32_t gas_cost, int16_t stack_req, int16_t stack_max_growth) {
	evmone::instruction instr(nullptr);
	instr.arg.block.gas_cost = gas_cost;
	instr.arg.block.stack_req = stack_req;
	instr.arg.block.stack_max_growth = stack_max_growth;
	return instr;
}
inline evmone::instruction instr_from_push(uint64_t v) {
	evmone::instruction instr(nullptr);
	instr.arg.small_push_value = v;
	return instr;
}
inline evmone::instruction instr_from_push(uint64_t n3, uint64_t n2, uint64_t n1, uint64_t n0) {
	evmone::instruction instr(nullptr);
	instr.arg.push_value = new intx::uint256;
	return instr;
}
inline evmone::instruction instr_from_num(uint64_t n) {
	evmone::instruction instr(nullptr);
	instr.arg.number = n;
	return instr;
}
`}
	cF := []string{`
#include "instrexe.hpp"
namespace evmone
{
const instruction* op_stop(const instruction*, AdvancedExecutionState& state) noexcept
{
    return state.exit(EVMC_SUCCESS);
}

const instruction* op_invalid(const instruction*, AdvancedExecutionState& state) noexcept
{
    return state.exit(EVMC_INVALID_INSTRUCTION);
}
const instruction* op_sstore(const instruction* instr, AdvancedExecutionState& state) noexcept
{
    const auto gas_left_correction = state.current_block_cost - instr->arg.number;
    state.gas_left += gas_left_correction;

    const auto status = sstore(state);
    if (status != EVMC_SUCCESS)
        return state.exit(status);

    if ((state.gas_left -= gas_left_correction) < 0)
        return state.exit(EVMC_OUT_OF_GAS);

    return ++instr;
}

const instruction* op_gas(const instruction* instr, AdvancedExecutionState& state) noexcept
{
    const auto correction = state.current_block_cost - instr->arg.number;
    const auto gas = static_cast<uint64_t>(state.gas_left + correction);
    state.stack.push(gas);
    return ++instr;
}

template <evmc_status_code status_code>
const instruction* op_return(const instruction*, AdvancedExecutionState& state) noexcept
{
    const auto offset = state.stack[0];
    const auto size = state.stack[1];

    if (!check_memory(state, offset, size))
        return state.exit(EVMC_OUT_OF_GAS);

    state.output_size = static_cast<size_t>(size);
    if (state.output_size != 0)
        state.output_offset = static_cast<size_t>(offset);
    return state.exit(status_code);
}

template <evmc_call_kind Kind, bool Static = false>
const instruction* op_call(const instruction* instr, AdvancedExecutionState& state) noexcept
{
    const auto gas_left_correction = state.current_block_cost - instr->arg.number;
    state.gas_left += gas_left_correction;

    const auto status = call<Kind, Static>(state);
    if (status != EVMC_SUCCESS)
        return state.exit(status);

    if ((state.gas_left -= gas_left_correction) < 0)
        return state.exit(EVMC_OUT_OF_GAS);

    return ++instr;
}

template <evmc_call_kind Kind>
const instruction* op_create(const instruction* instr, AdvancedExecutionState& state) noexcept
{
    const auto gas_left_correction = state.current_block_cost - instr->arg.number;
    state.gas_left += gas_left_correction;

    const auto status = create<Kind>(state);
    if (status != EVMC_SUCCESS)
        return state.exit(status);

    if ((state.gas_left -= gas_left_correction) < 0)
        return state.exit(EVMC_OUT_OF_GAS);

    return ++instr;
}

const instruction* op_undefined(const instruction*, AdvancedExecutionState& state) noexcept
{
    return state.exit(EVMC_UNDEFINED_INSTRUCTION);
}

const instruction* op_selfdestruct(const instruction*, AdvancedExecutionState& state) noexcept
{
    return state.exit(selfdestruct(state));
}

const instruction* opx_beginblock(const instruction* instr, AdvancedExecutionState& state) noexcept
{
    auto& block = instr->arg.block;

    if ((state.gas_left -= block.gas_cost) < 0)
        return state.exit(EVMC_OUT_OF_GAS);

    if (static_cast<int>(state.stack.size()) < block.stack_req)
        return state.exit(EVMC_STACK_UNDERFLOW);

    if (static_cast<int>(state.stack.size()) + block.stack_max_growth > Stack::limit)
        return state.exit(EVMC_STACK_OVERFLOW);

    state.current_block_cost = block.gas_cost;
    return ++instr;
}
}
`}
	fFmt := "const evmone::instruction* maot%s(const evmone::instruction* instr, evmone::AdvancedExecutionState& state) noexcept"
	for op := 0; op < 256; op++ {
		if len(TraitsTable[op].Name) == 0 || op == OP_JUMP || op == OP_JUMPI {
			continue // ignore such op code
		}
		sec := fmt.Sprintf("// %d %s\n", op, TraitsTable[op].Name)
		hF = append(hF, sec)
		cF = append(cF, sec)
		fStr := fmt.Sprintf(fFmt, TraitsTable[op].Name)
		content := fStr+" {\nevmone::"+
		           opTbl[op].FuncName+"(instr, state);\n"+
		           "return nullptr;\n}\n"
		if (TypeTable[op] & Inline) == 0 {
			hF = append(hF, fStr+";\n")
			cF = append(cF, content)
		} else {
			hF = append(hF, "inline "+content+";")
		}
	}
	err := os.WriteFile(path.Join(dir, "instrexe.hpp"), []byte(strings.Join(hF, "")), 0644)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(path.Join(dir, "instrexe.cpp"), []byte(strings.Join(cF, "")), 0644)
	if err != nil {
		panic(err)
	}
}
