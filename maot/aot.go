package maot

import (
	"fmt"
	"io"
	"os"

	"encoding/binary"
)

type BlockInfo struct {
	GasCost        uint32
	StackReq       int16
	StackMaxGrowth int16
}

type BlockAnalysis struct {
	GasCost         int
	StackReq        int
	StackMaxGrowth  int
	StackChange     int
	BeginBlockIndex int
}

func NewBlockAnalysis(index int) BlockAnalysis {
	return BlockAnalysis{BeginBlockIndex: index}
}

func (ba *BlockAnalysis) Close() BlockInfo {
	return BlockInfo{
		GasCost:        uint32(ba.GasCost),
		StackReq:       int16(ba.StackReq),
		StackMaxGrowth: int16(ba.StackMaxGrowth),
	}
}

type Instruction struct {
	OpCode         int
	Number         int
	PushValue      string
	SmallPushValue uint64
	Block          BlockInfo
}

func (i *Instruction) SetPushValue(bz []byte) {
	var b32 [32]byte
	copy(b32[32-len(bz):], bz)
	i.PushValue = fmt.Sprintf("0x%xull, 0x%xull, 0x%xull, 0x%xull",
		binary.BigEndian.Uint64(b32[0:8]),
		binary.BigEndian.Uint64(b32[8:16]),
		binary.BigEndian.Uint64(b32[16:24]),
		binary.BigEndian.Uint64(b32[24:32]))
}

type AdvancedCodeAnalysis struct {
	InstrList       []*Instruction
	JumpdestOffsets []int
	JumpdestTargets []int
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Analyze(rev int, codeArr []byte) (analysis AdvancedCodeAnalysis) {
	opTbl := OpTables[rev]

	analysis.InstrList = make([]*Instruction, 0, len(codeArr)+1)
	instr := &Instruction{OpCode: OPX_BEGINBLOCK}
	analysis.InstrList = append(analysis.InstrList, instr)

	block := NewBlockAnalysis(0)
	codePos := 0
	for codePos < len(codeArr) {
		opCode := codeArr[codePos]
		codePos++
		opInfo := opTbl[opCode]
		block.StackReq = max(block.StackReq, int(opInfo.StackReq)-block.StackChange)
		block.StackChange += int(opInfo.StackChange)
		block.StackMaxGrowth = max(block.StackMaxGrowth, block.StackChange)
		block.GasCost += int(opInfo.GasCost)
		if opCode == OP_JUMPDEST {
			analysis.JumpdestOffsets = append(analysis.JumpdestOffsets, codePos-1)
			analysis.JumpdestTargets = append(analysis.JumpdestTargets, len(analysis.InstrList)-1)
		} else {
			instr := &Instruction{OpCode: int(opCode)}
			analysis.InstrList = append(analysis.InstrList, instr)
		}

		instr = analysis.InstrList[len(analysis.InstrList)-1]
		isTerminator := false
		switch opCode {
		case OP_JUMP, OP_JUMPI, OP_STOP, OP_RETURN, OP_REVERT, OP_SELFDESTRUCT:
			isTerminator = true
		case OP_PUSH1, OP_PUSH2, OP_PUSH3, OP_PUSH4,
			OP_PUSH5, OP_PUSH6, OP_PUSH7, OP_PUSH8:
			pushSize := opCode - OP_PUSH1 + 1
			var data [8]byte
			copy(data[8-int(pushSize):], codeArr[codePos : codePos+int(pushSize)])
			instr.SmallPushValue = binary.BigEndian.Uint64(data[:])
			fmt.Printf("Here %d %#v %d\n", pushSize, data[:], instr.SmallPushValue)
			codePos += int(pushSize)
		case OP_PUSH9, OP_PUSH10, OP_PUSH11, OP_PUSH12,
			OP_PUSH13, OP_PUSH14, OP_PUSH15, OP_PUSH16,
			OP_PUSH17, OP_PUSH18, OP_PUSH19, OP_PUSH20,
			OP_PUSH21, OP_PUSH22, OP_PUSH23, OP_PUSH24,
			OP_PUSH25, OP_PUSH26, OP_PUSH27, OP_PUSH28,
			OP_PUSH29, OP_PUSH30, OP_PUSH31, OP_PUSH32:
			pushSize := opCode - OP_PUSH1 + 1
			instr.SetPushValue(codeArr[codePos : codePos+int(pushSize)])
			codePos += int(pushSize)
		case OP_GAS, OP_CALL, OP_CALLCODE, OP_DELEGATECALL, OP_STATICCALL,
			OP_CREATE, OP_CREATE2, OP_SSTORE:
			instr.Number = block.GasCost
		case OP_PC:
			instr.Number = codePos - 1
		}

		lastIdx := len(analysis.InstrList)-2
		if (opCode == OP_JUMP || opCode == OP_JUMPI) && codePos >= 2 {
			last := analysis.InstrList[lastIdx]
			if OP_PUSH1 <= last.OpCode && last.OpCode <= OP_PUSH3 && last.Number != 0 {
				instr.Number = last.Number
				analysis.InstrList[lastIdx].OpCode = NOP
			}
		}

		if isTerminator || (codePos < len(codeArr) && codeArr[codePos] == OP_JUMPDEST) {
			analysis.InstrList[block.BeginBlockIndex].Block = block.Close()
			instr := &Instruction{OpCode: OPX_BEGINBLOCK}
			analysis.InstrList = append(analysis.InstrList, instr)
			block = NewBlockAnalysis(len(analysis.InstrList) - 1)
		}
	}
	// Save current block.
	analysis.InstrList[block.BeginBlockIndex].Block = block.Close()

	instr = &Instruction{OpCode: OP_STOP}
	analysis.InstrList = append(analysis.InstrList, instr)
	return
}

func (analysis AdvancedCodeAnalysis) Dump(fout io.Writer) {
	wr(fout, `#include "instrexe.hpp"
evmc_result execute(evmc_vm* /*unused*/, const evmc_host_interface* host, evmc_host_context* ctx,
    evmc_revision rev, const evmc_message* msg, const uint8_t* code, size_t code_size) noexcept
{
    auto state = std::make_unique<AdvancedExecutionState>(*msg, rev, *host, ctx, code, code_size);
    instruction instr;
    instruction* next_instr = 1 + &instr;
    size_t PC = ~size_t(0);
`)
	analysis.DumpAllInstr(fout);
	analysis.DumpJumpTable(fout);
	wr(fout, "}\n")
}

func (analysis AdvancedCodeAnalysis) DumpJumpTable(fout io.Writer) {
	wr(fout, "JUMPTABLE:\n")
	wr(fout, "switch(PC){\n")
	for i, offset := range analysis.JumpdestOffsets {
		target := analysis.JumpdestTargets[i]
		wr(fout, "  case %d: goto L%05d;\n", offset, target)
	}
	wr(fout, "  default:\n")
	wr(fout, "    return state.exit(EVMC_BAD_JUMP_DESTINATION);\n")
	wr(fout, `}
ENDING:
    const auto gas_left =
        (state.status == EVMC_SUCCESS || state.status == EVMC_REVERT) ? state.gas_left : 0;

    return evmc::make_result(
        state.status, gas_left, state.memory.data() + state.output_offset, state.output_size);
}`)
}

func (analysis AdvancedCodeAnalysis) DumpAllInstr(fout io.Writer) {
	for pcPlus1, instr := range analysis.InstrList {
		pc := pcPlus1 - 1
		wr(fout, "// pc=%d op=%d (%s)\n", pc, instr.OpCode, TraitsTable[instr.OpCode].Name)
		if instr.OpCode == OP_JUMP && instr.Number != 0 { //Known target
			wr(fout, "goto L%05d;\n", instr.Number)
		}
		if instr.OpCode == OP_JUMPI && instr.Number != 0 { //Known target
			wr(fout, "if(test_jump_cond(state)) {\n")
			wr(fout, "  goto L%05d;\n", instr.Number)
			wr(fout, "}\n")
		}
		if instr.OpCode == OP_JUMP && instr.Number == 0 { //Unknown target
			wr(fout, "PC=pop_target_pc(state);\ngoto JUMPTABLE;\n")
		}
		if instr.OpCode == OP_JUMPI && instr.Number == 0 { //Unknown target
			wr(fout, "PC=(get_target_pc(state));\n")
			wr(fout, "if((~PC)!=0) goto JUMPTABLE;\n")
		}
		if instr.OpCode == OP_JUMP || instr.OpCode == OP_JUMPI {
			continue
		}
		switch instr.OpCode {
		case NOP:
			continue
		case OPX_BEGINBLOCK:
			wr(fout, "instr=instr_from_block(%d, %d, %d);\n", instr.Block.GasCost,
				instr.Block.StackReq, instr.Block.StackMaxGrowth)
		case OP_PUSH1, OP_PUSH2, OP_PUSH3, OP_PUSH4,
			OP_PUSH5, OP_PUSH6, OP_PUSH7, OP_PUSH8:
			fmt.Printf("Haha %d\n", instr.SmallPushValue)
			wr(fout, "instr=instr_from_push(%d);\n", instr.SmallPushValue)
		case OP_PUSH9, OP_PUSH10, OP_PUSH11, OP_PUSH12,
			OP_PUSH13, OP_PUSH14, OP_PUSH15, OP_PUSH16,
			OP_PUSH17, OP_PUSH18, OP_PUSH19, OP_PUSH20,
			OP_PUSH21, OP_PUSH22, OP_PUSH23, OP_PUSH24,
			OP_PUSH25, OP_PUSH26, OP_PUSH27, OP_PUSH28,
			OP_PUSH29, OP_PUSH30, OP_PUSH31, OP_PUSH32:
			wr(fout, "instr=instr_from_push(%s);\n", instr.PushValue)
		case OP_GAS, OP_CALL, OP_CALLCODE, OP_DELEGATECALL, OP_STATICCALL,
			OP_CREATE, OP_CREATE2, OP_SSTORE, OP_PC:
			wr(fout, "instr=instr_from_num(%d);\n", instr.Number)
		}
		name := TraitsTable[instr.OpCode].Name
		if t := TypeTable[instr.OpCode]; t == FullWithBreak || t == StateWithStatus {
			wr(fout, "if(next_instr!=maot%s(&instr, state)) goto ENDING;\n", name)
		} else if len(name) == 0 {
			wr(fout, "evmone::op_undefined(&instr, state);\ngoto ENDING;\n")
		} else {
			wr(fout, "maot%s(&instr, state);\n", name)
		}
	}
}

func wr(fout io.Writer, line string, a ...any) {
	s := fmt.Sprintf(line, a...)
	_, err := fout.Write([]byte(s))
	if err != nil {
		panic(err)
	}
}

func CodeToFile(rev int, codeArr []byte) {
	fout, err := os.Create("contract.cpp")
	if err != nil {
		panic(err)
	}
	analysis := Analyze(rev, codeArr)
	analysis.Dump(fout)
	err = fout.Close()
	if err != nil {
		panic(err)
	}
}

