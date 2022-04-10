package maot

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sort"
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
	PC             int
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
	instr := &Instruction{OpCode: OPX_BEGINBLOCK, PC: -1}
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
			analysis.JumpdestTargets = append(analysis.JumpdestTargets, codePos-1)
		} else {
			instr := &Instruction{OpCode: int(opCode), PC: codePos-1}
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
			//fmt.Printf("Here %d %#v %d\n", pushSize, data[:], instr.SmallPushValue)
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
			if OP_PUSH1 <= last.OpCode && last.OpCode <= OP_PUSH3 && last.SmallPushValue != 0 {
				instr.Number = int(last.SmallPushValue)
				analysis.InstrList[lastIdx].OpCode = NOP
			}
		}

		if isTerminator || (codePos < len(codeArr) && codeArr[codePos] == OP_JUMPDEST) {
			analysis.InstrList[block.BeginBlockIndex].Block = block.Close()
			instr := &Instruction{OpCode: OPX_BEGINBLOCK, PC: codePos}
			analysis.InstrList = append(analysis.InstrList, instr)
			block = NewBlockAnalysis(len(analysis.InstrList) - 1)
		}
	}
	// Save current block.
	analysis.InstrList[block.BeginBlockIndex].Block = block.Close()

	instr = &Instruction{OpCode: OP_STOP, PC: codePos}
	analysis.InstrList = append(analysis.InstrList, instr)
	return
}

func (analysis AdvancedCodeAnalysis) Dump(name string, fout io.Writer) {
	wr(fout, fmt.Sprintf(`#include <memory>
#include "instrexe.hpp"
extern "C" {
evmc_result execute_%s(evmc_vm* /*unused*/, const evmc_host_interface* host, evmc_host_context* ctx,
    evmc_revision rev, const evmc_message* msg, const uint8_t* code, size_t code_size) noexcept;
}
evmc_result execute_%s(evmc_vm* /*unused*/, const evmc_host_interface* host, evmc_host_context* ctx,
    evmc_revision rev, const evmc_message* msg, const uint8_t* code, size_t code_size) noexcept
{
    auto state = std::make_unique<evmone::AdvancedExecutionState>(*msg, rev, *host, ctx, code, code_size);
    evmone::instruction instr(nullptr);
    evmone::instruction* next_instr = 1 + &instr;
    size_t PC = ~size_t(0);
`, name, name))
	analysis.DumpAllInstr(fout);
	analysis.DumpJumpTable(fout);
	wr(fout, "}\n")
}

func (analysis AdvancedCodeAnalysis) DumpJumpTable(fout io.Writer) {
	wr(fout, "JUMPTABLE:\n")
	wr(fout, "switch(PC){\n")
	for _, target := range analysis.JumpdestTargets {
		wr(fout, "  case %d: goto L%05d;\n", target, target)
	}
	wr(fout, "  default:\n")
	wr(fout, "    state->exit(EVMC_BAD_JUMP_DESTINATION);\n")
	wr(fout, `}
ENDING:
    const auto gas_left =
        (state->status == EVMC_SUCCESS || state->status == EVMC_REVERT) ? state->gas_left : 0;

    return evmc::make_result(
        state->status, gas_left, state->memory.data() + state->output_offset, state->output_size);
`)
}

func (analysis AdvancedCodeAnalysis) DumpAllInstr(fout io.Writer) {
	for _, instr := range analysis.InstrList {
		if instr.OpCode == OP_JUMPDEST && instr.PC > 0 {
			wr(fout, "L%05d:\n", instr.PC)
		}
		if instr.OpCode == NOP {
			wr(fout, "// pc=%d NOP\n", instr.PC)
			continue
		} else {
			wr(fout, "// pc=%d op=%d (%s)\n", instr.PC, instr.OpCode, TraitsTable[instr.OpCode].Name)
		}
		if instr.OpCode == OP_JUMP && instr.Number != 0 { //Known target
			wr(fout, "goto L%05d;\n", instr.Number)
		}
		if instr.OpCode == OP_JUMPI && instr.Number != 0 { //Known target
			wr(fout, "if(test_jump_cond(*state)) {\n")
			wr(fout, "  goto L%05d;\n", instr.Number)
			wr(fout, "}\n")
		}
		if instr.OpCode == OP_JUMP && instr.Number == 0 { //Unknown target
			wr(fout, "PC=pop_target_pc(*state);\ngoto JUMPTABLE;\n")
		}
		if instr.OpCode == OP_JUMPI && instr.Number == 0 { //Unknown target
			wr(fout, "PC=(get_target_pc(*state));\n")
			wr(fout, "if((~PC)!=0) goto JUMPTABLE;\n")
		}
		if instr.OpCode == OP_JUMP || instr.OpCode == OP_JUMPI {
			continue
		}
		switch instr.OpCode {
		case OPX_BEGINBLOCK:
			wr(fout, "instr=instr_from_block(%d, %d, %d);\n", instr.Block.GasCost,
				instr.Block.StackReq, instr.Block.StackMaxGrowth)
		case OP_PUSH1, OP_PUSH2, OP_PUSH3, OP_PUSH4,
			OP_PUSH5, OP_PUSH6, OP_PUSH7, OP_PUSH8:
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
			wr(fout, "if(next_instr!=maot%s(&instr, *state)) goto ENDING;\n", name)
		} else if len(name) == 0 {
			wr(fout, "evmone::op_undefined(&instr, *state);\ngoto ENDING;\n")
		} else {
			wr(fout, "maot%s(&instr, *state);\n", name)
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

func CodeToFile(rev int, codeArr []byte, name, fname string) {
	fout, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	analysis := Analyze(rev, codeArr)
	analysis.Dump(name, fout)
	err = fout.Close()
	if err != nil {
		panic(err)
	}
}

func readFiles(dir string) (codeMap map[string][]byte) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	codeMap = make(map[string][]byte, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := ioutil.ReadFile(path.Join(dir, entry.Name()))
		if err != nil {
			panic(err)
		}

		codeMap[entry.Name()], err = hex.DecodeString(strings.TrimSpace(string(content)))
		if err != nil {
			fmt.Printf("f %s %s '%s'\n", dir, entry.Name(), strings.TrimSpace(string(content)))
			panic(err)
		}
	}
	return
}

func getQueryExecutorSrc(nameList []string) string {
	lines := make([]string, 0, 100)
	lines = append(lines, `
#include <string>
#include <unordered_map>
#include "evmc/evmc.h"

extern "C" {
__attribute__ ((visibility ("default"))) evmc_execute_fn query_executor(const evmc_address* destination);
`)
	for _, name := range nameList {
		s := fmt.Sprintf(`evmc_result execute_%s(evmc_vm* /*unused*/, const evmc_host_interface* host, evmc_host_context* ctx,
    evmc_revision rev, const evmc_message* msg, const uint8_t* code, size_t code_size) noexcept;`, name)
		lines = append(lines, s)
	}
	lines = append(lines, `
}

evmc_execute_fn query_executor(const evmc_address* destination) {
	static std::unordered_map<std::string, evmc_execute_fn> m;
	if(m.size() == 0) { //initialized on first called`)

	s := fmt.Sprintf("\t\tm.reserve(%d);", len(nameList))
	lines = append(lines, s)
	for _, name := range nameList {
		s = fmt.Sprintf("\t\tm.insert(std::make_pair<std::string, evmc_execute_fn>(\"%s\", execute_%s));", name, name)
		lines = append(lines, s)
	}
	lines = append(lines, "\t}")
	lines = append(lines, `
	std::string key((const char*)(destination->bytes), 20);
	auto got = m.find(key);
	if(got == m.end()) return nullptr;
	return got->second;
}
`)
	return strings.Join(lines, "\n")
}

func getCompileScript(nameList []string) string {
	lines := make([]string, 0, 100)
	lines = append(lines, "#!/bin/bash")
	lines = append(lines, "export MOEINGEVM="+os.Getenv("MOEINGEVM"))
	cmd := "g++ -fPIC -std=c++17 -I $MOEINGEVM/evmwrap/evmone.release/ -I $MOEINGEVM/evmwrap/evmc/include/ -I $MOEINGEVM/evmwrap/intx/include -I $MOEINGEVM/evmwrap/keccak/include"
	fileNames := make([]string, 0, len(nameList))
	for _, name := range nameList {
		lines = append(lines, "echo === "+name+" ===")
		lines = append(lines, cmd+" -c "+name+".cpp")
		fileNames = append(fileNames, name+".o")
	}
	lines = append(lines, cmd+" -c instrexe.cpp")
	last := cmd+" -shared -fvisibility=hidden -o libevmaot.so query_executor.cpp instrexe.o "+strings.Join(fileNames, " ")
	lines = append(lines, last)
	return strings.Join(lines, "\n")
}

func AotCompile(rev int, inDir string, outDir string) {
	codeMap := readFiles(inDir)
	nameList := make([]string, 0, len(codeMap))
	for name := range codeMap {
		nameList = append(nameList, name)
	}
	sort.Strings(nameList)
	for _, name := range nameList {
		codeArr := codeMap[name]
		ofile := path.Join(outDir, name+".cpp")
		CodeToFile(rev, codeArr, name, ofile)
	}
	src := getQueryExecutorSrc(nameList)
	ofile := path.Join(outDir, "query_executor.cpp")
	err := ioutil.WriteFile(ofile, []byte(src), 0644)
	if err != nil {
		panic(err)
	}
	DumpInstrExeFiles(outDir)
	src = getCompileScript(nameList)
	ofile = path.Join(outDir, "compile.sh")
	err = ioutil.WriteFile(ofile, []byte(src), 0644)
	if err != nil {
		panic(err)
	}
}

