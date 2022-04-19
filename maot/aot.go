package maot

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
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
	StackChange     int // the accumuluated StackChange of seen instructions
	BeginBlockIndex int // the starting position
}

func NewBlockAnalysis(index int) BlockAnalysis {
	return BlockAnalysis{BeginBlockIndex: index}
}

// Stop analyzing and get a compact BlockInfo
func (ba *BlockAnalysis) Close() BlockInfo {
	return BlockInfo{
		GasCost:        uint32(ba.GasCost),
		StackReq:       int16(ba.StackReq),
		StackMaxGrowth: int16(ba.StackMaxGrowth),
	}
}

// Some miscellaneous data using during an instruction's execution
// C++ code can use union to reduce the space requirement. In golang we just list them all
type Instruction struct {
	PC             int
	OpCode         int
	Number         int
	PushValue      string
	SmallPushValue uint64
	Block          BlockInfo
}

// For PUSH9~PUSH32
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
	// the following two fields contain the same targets
	JumpdestTargets []int
	TargetsSet      map[int]struct{}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Analyze(rev int, codeArr []byte) (analysis AdvancedCodeAnalysis) {
	opTbl := OpTables[rev]

	analysis.TargetsSet = make(map[int]struct{})
	analysis.InstrList = make([]*Instruction, 0, len(codeArr)+1)
	instr := &Instruction{OpCode: OPX_BEGINBLOCK, PC: -1} // for the first basic block
	analysis.InstrList = append(analysis.InstrList, instr)

	block := NewBlockAnalysis(0)
	codePos := 0
	for codePos < len(codeArr) {
		opCode := codeArr[codePos]
		codePos++
		opInfo := opTbl[opCode]

		block.StackReq = max(block.StackReq, int(opInfo.StackReq)-block.StackChange/*negative for pop*/)
		block.StackChange += int(opInfo.StackChange)
		//StackMaxGrowth is the peak value of StackChange
		block.StackMaxGrowth = max(block.StackMaxGrowth, block.StackChange)
		block.GasCost += int(opInfo.GasCost)

		//fmt.Printf("Now pos %d GasCost %d %d\n", codePos-1, block.GasCost, opInfo.GasCost)
		if opCode == OP_JUMPDEST {
			analysis.JumpdestTargets = append(analysis.JumpdestTargets, codePos-1)
			analysis.TargetsSet[codePos-1] = struct{}{}
		} else {
			instr := &Instruction{OpCode: int(opCode), PC: codePos - 1}
			analysis.InstrList = append(analysis.InstrList, instr)
		}

		instr = analysis.InstrList[len(analysis.InstrList)-1]
		isTerminator := false // does it terminate a basic block?
		switch opCode {
		case OP_JUMP, OP_JUMPI, OP_STOP, OP_RETURN, OP_REVERT, OP_SELFDESTRUCT:
			isTerminator = true
		case OP_PUSH1, OP_PUSH2, OP_PUSH3, OP_PUSH4,
			OP_PUSH5, OP_PUSH6, OP_PUSH7, OP_PUSH8:
			pushSize := opCode - OP_PUSH1 + 1
			var data [8]byte
			copy(data[8-int(pushSize):], codeArr[codePos:codePos+int(pushSize)])
			instr.SmallPushValue = binary.BigEndian.Uint64(data[:]) // param used during execution
			//fmt.Printf("Here %d %#v %d\n", pushSize, data[:], instr.SmallPushValue)
			codePos += int(pushSize)
		case OP_PUSH9, OP_PUSH10, OP_PUSH11, OP_PUSH12,
			OP_PUSH13, OP_PUSH14, OP_PUSH15, OP_PUSH16,
			OP_PUSH17, OP_PUSH18, OP_PUSH19, OP_PUSH20,
			OP_PUSH21, OP_PUSH22, OP_PUSH23, OP_PUSH24,
			OP_PUSH25, OP_PUSH26, OP_PUSH27, OP_PUSH28,
			OP_PUSH29, OP_PUSH30, OP_PUSH31, OP_PUSH32:
			pushSize := opCode - OP_PUSH1 + 1
			instr.SetPushValue(codeArr[codePos : codePos+int(pushSize)]) // param used during execution
			codePos += int(pushSize)
		case OP_GAS, OP_CALL, OP_CALLCODE, OP_DELEGATECALL, OP_STATICCALL,
			OP_CREATE, OP_CREATE2, OP_SSTORE:
			instr.Number = block.GasCost // param used during execution
		case OP_PC:
			instr.Number = codePos - 1 // param used during execution
		}

		// Fuse a PUSH1~3 instruction and a JUMP/JUMPI instruction, such that we know the target during compilation
		lastIdx := len(analysis.InstrList) - 2
		if (opCode == OP_JUMP || opCode == OP_JUMPI) && codePos >= 2 {
			last := analysis.InstrList[lastIdx]
			if (OP_PUSH1 <= last.OpCode && last.OpCode <= OP_PUSH3) && last.SmallPushValue != 0 {
				instr.Number = int(last.SmallPushValue)
				analysis.InstrList[lastIdx].OpCode = NOP
			}
		}

		if isTerminator || (codePos < len(codeArr) && codeArr[codePos] == OP_JUMPDEST) {
			analysis.InstrList[block.BeginBlockIndex].Block = block.Close() //close the last basic block
			//fmt.Printf("At Close %d %#v\n", block.BeginBlockIndex, analysis.InstrList[block.BeginBlockIndex].Block)
			instr := &Instruction{OpCode: OPX_BEGINBLOCK, PC: codePos}
			analysis.InstrList = append(analysis.InstrList, instr)
			block = NewBlockAnalysis(len(analysis.InstrList) - 1) // open a new block
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
#include <iostream>
#include "instrexe.hpp"
extern "C" { // declare the execute function with C linkage
evmc_result execute_%s(evmc_vm* /*unused*/, const evmc_host_interface* host, evmc_host_context* ctx,
    evmc_revision rev, const evmc_message* msg, const uint8_t* code, size_t code_size) noexcept;
}

void show_stack(evmone::AdvancedExecutionState& state) {
    for(int i = state.stack.size() - 1; i >= 0; i--) {
        std::cout<<"0x"<<intx::hex(state.stack[i])<<std::endl;
    }
}

evmc_result execute_%s(evmc_vm* /*unused*/, const evmc_host_interface* host, evmc_host_context* ctx,
    evmc_revision rev, const evmc_message* msg, const uint8_t* code, size_t code_size) noexcept
{
    auto state = std::make_unique<evmone::AdvancedExecutionState>(*msg, rev, *host, ctx, code, code_size);
    evmone::instruction instr(nullptr);
    evmone::instruction* next_instr = 1 + &instr;
    size_t PC = ~size_t(0);
`, name, name))
	analysis.DumpAllInstr(fout)
	analysis.DumpJumpTable(fout)
	wr(fout, "}\n")
}

// The JumpTable is a PC-to-label table implemented with "switch"
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
			wr(fout, "L%05d:\n", instr.PC) // a label at the beginning of a basic block
		}
		if instr.OpCode == NOP {
			wr(fout, "// pc=%d NOP\n", instr.PC)
			continue
		} else {
			wr(fout, "// pc=%d op=%d (%s)\n", instr.PC, instr.OpCode, TraitsTable[instr.OpCode].Name)
			//wr(fout, "std::cout<<\"pc=%d %s gas 0x\"<<std::hex<<state->gas_left<<std::endl;\n",
			//	instr.PC, TraitsTable[instr.OpCode].Name)
			//wr(fout, "show_stack(*state);\n")
		}
		if instr.OpCode == OP_JUMP && instr.Number != 0 { //Known target, for an unconditional jump
			if _, ok := analysis.TargetsSet[instr.Number]; ok {
				wr(fout, "goto L%05d;\n", instr.Number)
			} else {
				wr(fout, "state->exit(EVMC_BAD_JUMP_DESTINATION); goto ENDING;//%05d", instr.Number)
			}
		}
		if instr.OpCode == OP_JUMPI && instr.Number != 0 { //Known target, for a conditional jump
			wr(fout, "if(test_jump_cond(*state)) {\n")
			if _, ok := analysis.TargetsSet[instr.Number]; ok {
				wr(fout, "  goto L%05d;\n", instr.Number)
			} else {
				wr(fout, "  state->exit(EVMC_BAD_JUMP_DESTINATION); goto ENDING;//%05d", instr.Number)
			}
			wr(fout, "}\n")
		}
		if instr.OpCode == OP_JUMP && instr.Number == 0 { //Unknown target, for an unconditional jump
			wr(fout, "PC=pop_target_pc(*state);\ngoto JUMPTABLE;\n")
		}
		if instr.OpCode == OP_JUMPI && instr.Number == 0 { //Unknown target, for a conditional jump
			wr(fout, "PC=(get_target_pc(*state));\n")
			wr(fout, "if((~PC)!=0) goto JUMPTABLE;\n") // an all-ones PC means "don't jump"
		}
		if instr.OpCode == OP_JUMP || instr.OpCode == OP_JUMPI {
			continue
		}
		// prepare some miscellaneous information for the instruction's execution
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
			// an instruction which may not return instr++
			wr(fout, "if(next_instr!=maot%s(&instr, *state)) goto ENDING;\n", name)
		} else if len(name) == 0 { //undefined instruction
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

// read files in "dir" and returns a "address-to-bytecode" map
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

// hex address to a string literal which presents it
func addr2str(addr string) string {
	res := ""
	for i := 0; i < len(addr); i += 2 {
		res += "\\x" + addr[i:i+2]
	}
	return res
}

// generate the query_executor function, which maps <addr> to an execute_<addr> function
func getQueryExecutorSrc(addrList []string) string {
	lines := make([]string, 0, 100)
	lines = append(lines, `
#include <string>
#include <unordered_map>
#include "evmc/evmc.h"

extern "C" {
__attribute__ ((visibility ("default"))) evmc_execute_fn query_executor(const evmc_address* destination);
`)
	for _, addr := range addrList {
		s := fmt.Sprintf(`evmc_result execute_%s(evmc_vm* /*unused*/, const evmc_host_interface* host, evmc_host_context* ctx,
    evmc_revision rev, const evmc_message* msg, const uint8_t* code, size_t code_size) noexcept;`, addr)
		lines = append(lines, s)
	}
	lines = append(lines, `
}

evmc_execute_fn query_executor(const evmc_address* destination) {
	static std::unordered_map<std::string, evmc_execute_fn> m;
	if(m.size() == 0) { //initialized on first called`)

	s := fmt.Sprintf("\t\tm.reserve(%d);", len(addrList))
	lines = append(lines, s)
	for _, addr := range addrList {
		s = fmt.Sprintf("\t\tm.insert(std::make_pair<std::string, evmc_execute_fn>(\"%s\", execute_%s));",
			addr2str(addr), addr)
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

func getCompileScript(addrList []string) string {
	lines := make([]string, 0, 100)
	lines = append(lines, "#!/bin/bash")
	lines = append(lines, "export MOEINGEVM="+os.Getenv("MOEINGEVM"))
	cmd := "g++ -O3 -fPIC -std=c++17 -I $MOEINGEVM/evmwrap/evmone.release/ -I $MOEINGEVM/evmwrap/evmc/include/ -I $MOEINGEVM/evmwrap/intx/include -I $MOEINGEVM/evmwrap/keccak/include"
	fileNames := make([]string, 0, len(addrList))
	for _, addr := range addrList { // compile the files generated from bytecodes
		lines = append(lines, "echo === "+addr+" ===")
		lines = append(lines, cmd+" -c "+addr+".cpp")
		fileNames = append(fileNames, addr+".o")
	}
	lines = append(lines, cmd+" -c instrexe.cpp")
	last := cmd + " -shared -fvisibility=hidden -o libevmaot.so query_executor.cpp instrexe.o " + strings.Join(fileNames, " ") // we use -fvisibility=hidden to hide unnecessary functions
	lines = append(lines, last)
	return strings.Join(lines, "\n")
}

func AotCompile(rev int, inDir string, outDir string) {
	codeMap := readFiles(inDir)
	addrList := make([]string, 0, len(codeMap))
	for addr := range codeMap {
		addrList = append(addrList, addr)
	}
	sort.Strings(addrList)
	for _, addr := range addrList {
		codeArr := codeMap[addr]
		ofile := path.Join(outDir, addr+".cpp")
		CodeToFile(rev, codeArr, addr, ofile)
	}
	src := getQueryExecutorSrc(addrList)
	ofile := path.Join(outDir, "query_executor.cpp")
	err := ioutil.WriteFile(ofile, []byte(src), 0644)
	if err != nil {
		panic(err)
	}
	DumpInstrExeFiles(outDir)
	src = getCompileScript(addrList)
	ofile = path.Join(outDir, "compile.sh")
	err = ioutil.WriteFile(ofile, []byte(src), 0644)
	if err != nil {
		panic(err)
	}
}
