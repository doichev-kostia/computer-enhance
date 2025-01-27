package decoder

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// definitions.D_FIELD
const (
	RegIsSource      = 0
	RegIsDestination = 1
)

// definitions.W_FIELD
const (
	ByteOperation = byte(0)
	WordOperation = byte(1)
)

// definitions.S_FIELD
const (
	NoSignExtension = byte(0)
	SignExtension   = byte(1) // Sign extend 8-bit immediate data to 16 bits if W=1
)

// definitions.V_FIELD
const (
	CountByOne = 0 // Shift/rotate count is one
	CountByCL  = 1 // Shift/rotate count is specified in CL register
)

// definitions.Z_FIELD
const (
	LoopWhileNotZero = 0
	LoopWhileZero    = 1
)

// definitions.MOD field
//
// The MOD field indicates how many displacement bytes are present.
// Following Intel convention, if the displacement is two bytes,
// the most-significant byte is stored second in the instruction. (Little Endian)
// If the displacement is only a single byte, the 8086 or 8088 __automatically sign-extends__ (section 2.8, page 2-68)
// this quantity to 16-bits before using the information in further address calculations.
// Immediate values __always__ follow any displacement values that __may__ be present. (data-low, data-high)
// The second byte of a two-byte immediate value is the most significant. (Little Endian)
const (
	MemoryModeNoDisplacementFieldEncoding = 0b00
	MemoryMode8DisplacementFieldEncoding  = 0b01
	MemoryMode16DisplacementFieldEncoding = 0b10
	RegisterModeFieldEncoding             = 0b11
)

// definitions.REG (Register) field encoding - ByteOperationRegisterFieldEncoding & WordOperationRegisterFieldEncoding
// | REG | W = 0 | W = 1|
// ---------------------
// | 000 | AL    | AX   |
// | 001 | CL    | CX   |
// | 010 | DL    | DX   |
// | 011 | BL    | BX   |
// | 100 | AH    | SP   |
// | 101 | CH    | BP   |
// | 110 | DH    | SI   |
// | 111 | BH    | DI   |
var ByteOperationRegisterFieldEncoding = map[byte]string{
	0b000: "al",
	0b001: "cl",
	0b10:  "dl",
	0b11:  "bl",
	0b100: "ah",
	0b101: "ch",
	0b110: "dh",
	0b111: "bh",
}

var WordOperationRegisterFieldEncoding = map[byte]string{
	0b000: "ax",
	0b001: "cx",
	0b10:  "dx",
	0b11:  "bx",
	0b100: "sp",
	0b101: "bp",
	0b110: "si",
	0b111: "di",
}

var SegmentRegisterFieldEncoding = map[byte]string{
	0b00: "es", // extra segment
	0b01: "cs", // code segment
	0b10: "ss", // stack segment
	0b11: "ds", // data segment
}

// EffectiveAddressEquation based on the r/m (Register/Memory) field encoding
// Table 4-10 in "Instruction reference"
// r/m: equation
var EffectiveAddressEquation = map[byte]string{
	0b000: "bx + si",
	0b001: "bx + di",
	0b010: "bp + si",
	0b011: "bp + di",
	0b100: "si",
	0b101: "di",
	0b110: "bp", // If MOD = 00, then it's a Direct Address
	0b111: "bx",
}

var JumpNames = map[byte]string{
	0b01110100: "JZ",
	0b01111100: "JL",
	0b01111110: "JLE",
	0b01110010: "JB",
	0b01110110: "JBE",
	0b01111010: "JP",
	0b01110000: "JO",
	0b01111000: "JS",
	0b01110101: "JNZ",
	0b01111101: "JGE",
	0b01111111: "JG",
	0b01110011: "JAE",
	0b01110111: "JA",
	0b01111011: "JNP",
	0b01110001: "JNO",
	0b01111001: "JNS",
	0b11100011: "JCXZ",

	// Loops
	0b11100010: "LOOP",
	0b11100001: "LOOPZ",
	0b11100000: "LOOPNZ",
}

var JumpAlternativeNames = map[byte]string{
	0b01110100: "JE",
	0b01111100: "JNGE",
	0b01111110: "JNG",
	0b01110010: "JNAE",
	0b01110110: "JNA",
	0b01111010: "JPE",
	0b01110101: "JNE",
	0b01111101: "JNL",
	0b01111111: "JNLE",
	0b01110011: "JNB",
	0b01110111: "JNBE",
	0b01111011: "JPO",

	// Loops
	0b11100001: "LOOPE",
	0b11100000: "LOOPNE",
}

type instructionNode struct {
	value string
	pos   int
}

type Decoder struct {
	bytes    []byte
	pos      int
	segment  string // for the effective address segment override
	nodes    []instructionNode
	labels   map[int]string // pos:label
	cacheKey string
	decoded  []byte
}

func NewDecoder(bytes []byte) *Decoder {
	return &Decoder{
		bytes:    bytes,
		pos:      0,
		segment:  "",
		nodes:    make([]instructionNode, 0),
		labels:   make(map[int]string),
		cacheKey: "",
		decoded:  make([]byte, 0),
	}
}

func (d *Decoder) appendInstruction(pos int, value string) {
	n := instructionNode{
		value: value,
		pos:   pos,
	}

	d.nodes = append(d.nodes, n)
}

func (d *Decoder) computeCacheKey() string {
	return fmt.Sprintf("n=%d;l=%d", len(d.nodes), len(d.labels))
}

func (d *Decoder) GetDecoded() []byte {
	cacheKey := d.computeCacheKey()
	if cacheKey == d.cacheKey {
		return d.decoded
	}

	d.decoded = d.decoded[:0] // reuse the same array
	for _, node := range d.nodes {
		instruction := ""
		label, ok := d.labels[node.pos-1] // as we start counting instructions from 1, instead of 0
		if ok {
			instruction += fmt.Sprintf("%s:\n", label)
		}

		instruction += node.value
		d.decoded = append(d.decoded, []byte(instruction)...)

	}

	d.cacheKey = cacheKey
	return d.decoded
}

func (d *Decoder) Decode() ([]byte, error) {
	d.pos = 0
	for {
		// Section 2.7 Instruction set. p. 2-30
		instruction := ""
		prefix := ""

		var err error
		operation, ok := d.next()
		if ok == false {
			// TODO: return EOF
			break
		}
		instructionPointer := d.pos

		// Prefix
		switch {
		case d.matchPattern("LOCK: Bus lock prefix", operation, "0b11110000"):
			prefix = "lock "
		case d.matchPattern("REP: Repeat", operation, "0b1111001z"):
			prefix = repeatPrefix(operation, d) + " "
		}

		if prefix != "" {
			operation, ok = d.next()
			if ok == false {
				// TODO: return EOF
				break
			}
			if instructionPointer != instructionPointer {
				panic("Assertion Failed: The instruction pointer must not be updated when handling prefixes")
			}
		}

		if d.matchPattern("SEGMENT: override prefix", operation, "0b001__110") {
			d.segment = segmentPrefix(operation, d)
			operation, ok = d.next()
			if ok == false {
				// TODO: return EOF
				break
			}
		} else {
			d.segment = ""
		}

		// Table 4-12. 8086 Instruction Encoding
		switch {
		// MOV = Move
		case d.matchPattern("MOV: Register/memory to/from register", operation, "0b100010dw"):
			instruction, err = moveRegMemToReg(operation, d)
		case d.matchPattern("MOV: Immediate to register/memory", operation, "0b1100011w"):
			instruction, err = moveImmediateToRegOrMem(operation, d)
		case d.matchPattern("MOV: Immediate to register", operation, "0b1011wreg"):
			instruction, err = moveImmediateToReg(operation, d)
		case d.matchPattern("MOV: Memory to accumulator", operation, "0b1010000w"):
			instruction, err = moveMemoryToAccumulator(operation, d)
		case d.matchPattern("MOV: Accumulator to memory", operation, "0b1010001w"):
			instruction, err = moveAccumulatorToMemory(operation, d)

		// PUSH
		case d.matchPattern("PUSH: Register/memory", operation, "0b11111111|0b__110___"):
			instruction, err = pushRegOrMem(operation, d)
		case d.matchPattern("PUSH: Register", operation, "0b01010reg"):
			instruction, err = pushReg(operation, d)
		case d.matchPattern("PUSH: segment register", operation, "0b000__110"):
			instruction, err = pushSegmentReg(operation, d)

		// POP
		case d.matchPattern("POP: Register/memory", operation, "0b10001111|0b__000___"):
			instruction, err = popRegOrMem(operation, d)
		case d.matchPattern("POP: Register", operation, "0b01011reg"):
			instruction, err = popReg(operation, d)
		case d.matchPattern("POP: segment register", operation, "0b000__111"):
			instruction, err = popSegmentReg(operation, d)

		// XCHG = Exchange
		case d.matchPattern("XCHG: Register/memory with register", operation, "0b1000011w"):
			instruction, err = exchangeRegOrMemWithReg(operation, d)
		case d.matchPattern("XCHG: register with accumulator", operation, "0b10010reg"):
			instruction, err = exchangeRegWithAccumulator(operation, d)

		// IN = Input from
		case d.matchPattern("IN: Fixed port", operation, "0b1110010w"):
			instruction, err = inputFromFixedPort(operation, d)
		case d.matchPattern("IN: Variable port", operation, "0b1110110w"):
			instruction, err = inputFromVariablePort(operation, d)

		// OUT = Output to
		case d.matchPattern("OUT: Fixed port", operation, "0b1110011w"):
			instruction, err = outputToFixedPort(operation, d)
		case d.matchPattern("OUT: Variable port", operation, "0b1110111w"):
			instruction, err = outputToVariablePort(operation, d)

		case d.matchPattern("XLAT - Translate byte to AL", operation, "0b11010111"):
			instruction, err = xlat(operation, d)

		// Address Object Transfers
		case d.matchPattern("LEA - Load effective address to register", operation, "0b10001101"):
			instruction, err = lea(operation, d)
		case d.matchPattern("LDS - Load pointer to DS", operation, "0b11000101"):
			instruction, err = lds(operation, d)
		case d.matchPattern("LES - Load pointer to ES", operation, "0b11000100"):
			instruction, err = les(operation, d)

		// Flag Transfers
		case d.matchPattern("LAHF - Load AH with flags", operation, "0b10011111"):
			instruction, err = lahf(operation, d)
		case d.matchPattern("SAHF - Store AH into flags", operation, "0b10011110"):
			instruction, err = sahf(operation, d)
		case d.matchPattern("PUSHF - Push flags", operation, "0b10011100"):
			instruction, err = pushf(operation, d)
		case d.matchPattern("POPF - Pop flags", operation, "0b10011101"):
			instruction, err = popf(operation, d)

		// ADD
		case d.matchPattern("ADD: Reg/memory with register to either", operation, "0b000000dw"):
			instruction, err = addRegOrMemToReg(operation, d)
		case d.matchPattern("ADD: Immediate to register/memory", operation, "0b100000sw|0b__000___"):
			instruction, err = addImmediateToRegOrMem(operation, d)
		case d.matchPattern("ADD: Immediate to accumulator", operation, "0b0000010w"):
			instruction, err = addImmediateToAccumulator(operation, d)

		// ADC = Add with carry
		case d.matchPattern("ADC: Reg/memory with register to either", operation, "0b000100dw"):
			instruction, err = adcRegOrMemToReg(operation, d)
		case d.matchPattern("ADC: Immediate to register/memory", operation, "0b100000sw|0b__010___"):
			instruction, err = adcImmediateToRegOrMem(operation, d)
		case d.matchPattern("ADC: Immediate to accumulator", operation, "0b0001010w"):
			instruction, err = adcImmediateToAccumulator(operation, d)

		// INC = Increment
		case d.matchPattern("INC: Register/memory", operation, "0b1111111w|0b__000___"):
			instruction, err = incRegOrMem(operation, d)
		case d.matchPattern("INC: Register", operation, "0b01000reg"):
			instruction, err = incReg(operation, d)

		case d.matchPattern("AAA: ASCII adjust for add", operation, "0b00110111"):
			instruction, err = aaa(operation, d)
		case d.matchPattern("DAA: Decimal adjust for add", operation, "0b00100111"):
			instruction, err = daa(operation, d)

		// SUB = Subtract
		case d.matchPattern("SUB: Reg/memory and register to either", operation, "0b001010dw"):
			instruction, err = subRegOrMemFromReg(operation, d)
		case d.matchPattern("SUB: Immediate to register/memory", operation, "0b100000sw|0b__101___"):
			instruction, err = subImmediateFromRegOrMem(operation, d)
		case d.matchPattern("SUB: Immediate from accumulator", operation, "0b0010110w"):
			instruction, err = subImmediateFromAccumulator(operation, d)

		// SBB = Subtract with borrow
		case d.matchPattern("SBB: Reg/memory and register to either", operation, "0b000110dw"):
			instruction, err = sbbRegOrMemFromReg(operation, d)
		case d.matchPattern("SBB: Immediate to register/memory", operation, "0b100000sw|0b__011___"):
			instruction, err = sbbImmediateFromRegOrMem(operation, d)
		case d.matchPattern("SBB: Immediate from accumulator", operation, "0b0001110w"):
			instruction, err = sbbImmediateFromAccumulator(operation, d)

		// DEC = Decrement
		case d.matchPattern("DEC: Register/memory", operation, "0b1111111w|0b__001___"):
			instruction, err = decRegOrMem(operation, d)
		case d.matchPattern("DEC: Register", operation, "0b01001reg"):
			instruction, err = decReg(operation, d)

		case d.matchPattern("NEG: Change sign", operation, "0b1111011w|0b__011___"):
			instruction, err = neg(operation, d)

		// CMP = Compare
		case d.matchPattern("CMP: Reg/memory and register", operation, "0b001110dw"):
			instruction, err = cmpRegOrMemWithReg(operation, d)
		case d.matchPattern("CMP: Immediate with register/memory", operation, "0b100000sw|0b__111___"):
			instruction, err = cmpImmediateWithRegOrMem(operation, d)
		case d.matchPattern("CMP: Immediate from accumulator", operation, "0b0011110w"):
			instruction, err = cmpImmediateWithAccumulator(operation, d)

		case d.matchPattern("AAS: ASCII adjust for subtract", operation, "0b00111111"):
			instruction, err = aas(operation, d)
		case d.matchPattern("DAS: decimal adjust for subtract", operation, "0b00101111"):
			instruction, err = das(operation, d)

		case d.matchPattern("MUL: Unsigned multiply", operation, "0b1111011w|0b__100___"):
			instruction, err = mul(operation, d)
		case d.matchPattern("IMUL: Signed multiply", operation, "0b1111011w|0b__101___"):
			instruction, err = imul(operation, d)
		case d.matchPattern("AAM: ASCII adjust for multiply", operation, "0b11010100|0b00001010"):
			instruction, err = aam(operation, d)

		case d.matchPattern("DIV: Unsigned divide", operation, "0b1111011w|0b__110___"):
			instruction, err = div(operation, d)
		case d.matchPattern("IDIV: Signed divide", operation, "0b1111011w|0b__111___"):
			instruction, err = idiv(operation, d)
		case d.matchPattern("AAD: ASCII adjust for divide", operation, "0b11010101|0b00001010"):
			instruction, err = aad(operation, d)
		case d.matchPattern("CBW: convert byte to word", operation, "0b10011000"):
			instruction, err = cbw(operation, d)
		case d.matchPattern("CWD: convert word to double word", operation, "0b10011001"):
			instruction, err = cwd(operation, d)

		// LOGIC
		case d.matchPattern("NOT: Invert", operation, "0b1111011w|0b__010___"):
			instruction, err = not(operation, d)
		case d.matchPattern("SHL/SAL: Shift logical/arithmetic left", operation, "0b110100vw|0b__100___"):
			instruction, err = shl(operation, d)
		case d.matchPattern("SHR: Shift logical right", operation, "0b110100vw|0b__101___"):
			instruction, err = shr(operation, d)
		case d.matchPattern("SAR: Shift arithmetic right", operation, "0b110100vw|0b__111___"):
			instruction, err = sar(operation, d)
		case d.matchPattern("ROL: Rotate left", operation, "0b110100vw|0b__000___"):
			instruction, err = rol(operation, d)
		case d.matchPattern("ROR: Rotate right", operation, "0b110100vw|0b__001___"):
			instruction, err = ror(operation, d)
		case d.matchPattern("RCL: Rotate through carry left", operation, "0b110100vw|0b__010___"):
			instruction, err = rcl(operation, d)
		case d.matchPattern("RCR: Rotate through carry right", operation, "0b110100vw|0b__011___"):
			instruction, err = rcr(operation, d)

		// AND
		case d.matchPattern("AND: Logical AND reg/mem with reg", operation, "0b001000dw"):
			instruction, err = andRegOrMemWithReg(operation, d)
		case d.matchPattern("AND: Logical AND immediate with reg/mem", operation, "0b1000000w|0b__100___"):
			instruction, err = andImmediateWithRegOrMem(operation, d)
		case d.matchPattern("AND: Logical AND immediate with accumulator", operation, "0b0010010w"):
			instruction, err = andImmediateWithAccumulator(operation, d)

		// TEST
		case d.matchPattern("TEST: Logical compare reg/mem with reg", operation, "0b100001dw"): // NOTE(Kostia): for some reason, the "Instruction reference" says that test is [000100|d|w], but when using nasm v2.16.03, the opcode is different. Moreover, the table 4-13 aligns with the nasm, but 4-12 doesn't
			instruction, err = testRegOrMemWithReg(operation, d)
		case d.matchPattern("TEST: Logical compare immediate with reg/mem", operation, "0b1111011w|0b__000___"):
			instruction, err = testImmediateWithRegOrMem(operation, d)
		case d.matchPattern("TEST: Logical compare immediate with accumulator", operation, "0b1010100w"):
			instruction, err = testImmediateWithAccumulator(operation, d)

		// OR
		case d.matchPattern("OR: Logical OR reg/mem with reg", operation, "0b000010dw"):
			instruction, err = orRegOrMemWithReg(operation, d)
		case d.matchPattern("OR: Logical OR immediate with reg/mem", operation, "0b1000000w|0b__001___"):
			instruction, err = orImmediateWithRegOrMem(operation, d)
		case d.matchPattern("OR: Logical OR immediate with accumulator", operation, "0b0000110w"):
			instruction, err = orImmediateWithAccumulator(operation, d)

		// XOR
		case d.matchPattern("XOR: Logical XOR reg/mem with reg", operation, "0b001100dw"):
			instruction, err = xorRegOrMemWithReg(operation, d)
		case d.matchPattern("XOR: Logical XOR immediate with reg/mem", operation, "0b1000000w|0b__110___"): // NOTE(Kostia): for some reason, the "Instruction reference" says that xor is [0011010|w] [data] [disp-lo?] [disp-hi?] [data] [data if w=1], but when using nasm v2.16.03, the opcode is different and the [data] seems to be wrong. Moreover, the table 4-13 aligns with the nasm, but 4-12 doesn't
			instruction, err = xorImmediateWithRegOrMem(operation, d)
		case d.matchPattern("XOR: Logical XOR immediate with accumulator", operation, "0b0011010w"):
			instruction, err = xorImmediateWithAccumulator(operation, d)

		// STRING
		case d.matchPattern("MOVS: move byte/word", operation, "0b1010010w"):
			instruction, err = movs(operation, d)
		case d.matchPattern("CMPS: compare byte/word", operation, "0b1010011w"):
			instruction, err = cmps(operation, d)
		case d.matchPattern("SCAS: scan byte/word", operation, "0b1010111w"):
			instruction, err = scas(operation, d)
		case d.matchPattern("LODS: load byte/word", operation, "0b1010110w"):
			instruction, err = lods(operation, d)
		case d.matchPattern("STOS: store byte/word", operation, "0b1010101w"):
			instruction, err = stos(operation, d)

		// CALL
		case d.matchPattern("CALL: Direct within segment", operation, "0b11101000"):
			instruction, err = callDirectWithinSegment(operation, d)
		case d.matchPattern("CALL: Indirect within segment", operation, "0b11111111|0b__010___"):
			instruction, err = callIndirectWithinSegment(operation, d)
		case d.matchPattern("CALL: Direct intersegment", operation, "0b10011010"):
			instruction, err = callDirectIntersegment(operation, d)
		case d.matchPattern("CALL: Indirect intersegment", operation, "0b11111111|0b__011___"):
			instruction, err = callIndirectIntersegment(operation, d)

		// JMP = Unconditional jump
		case d.matchPattern("JMP: Direct within segment", operation, "0b11101001"):
			instruction, err = jumpDirectWithinSegment(operation, d)
		case d.matchPattern("JMP: Direct within segment-short", operation, "0b11101011"):
			instruction, err = jumpDirectWithinSegmentShort(operation, d)
		case d.matchPattern("JMP: Indirect within segment", operation, "0b11111111|0b__100___"):
			instruction, err = jumpIndirectWithinSegment(operation, d)
		case d.matchPattern("JMP: Direct intersegment", operation, "0b11101010"):
			instruction, err = jumpDirectIntersegment(operation, d)
		case d.matchPattern("JMP: Indirect intersegment", operation, "0b11111111|0b__101___"):
			instruction, err = jumpIndirectIntersegment(operation, d)

		// RET = Return from CALL
		case d.matchPattern("RET: Within segment", operation, "0b11000011"):
			panic("TODO: RET: Within segment")
		case d.matchPattern("RET: Within seg adding immed to SP", operation, "0b11000010"):
			panic("TODO: RET: Within seg adding immed to SP")
		case d.matchPattern("RET: Intersegment", operation, "0b11001011"):
			panic("TODO: RET: Intersegment")
		case d.matchPattern("RET: Intersegment adding immediate to SP", operation, "0b11001010"):
			panic("TODO: RET: Intersegment adding immediate to SP")

		// Jumps
		case d.matchPattern("JE/JZ: Jump on equal/zero", operation, "0b01110100"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JL/JNGE: Jump on less/not greater or equal", operation, "0b01111100"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JLE/JNG: Jump on less or equal/not greater", operation, "0b01111110"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JB/JNAE: Jump on below/not above or equal", operation, "0b01110010"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JBE/JNA: Jump on below or equal/not above", operation, "0b01110110"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JP/JPE: Jump on parity/even", operation, "0b01111010"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JO: Jump on overflow", operation, "0b01110000"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JS: Jump on sign", operation, "0b01111000"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JNE/JNZ: Jump on not equal/not zero", operation, "0b01110101"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JNL/JGE: Jump on not less/greater or equal", operation, "0b01111101"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JNLE/JG: Jump on not less nor equal/greater", operation, "0b01111111"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JNB/JAE: Jump on not below/above or equal", operation, "0b01110011"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JNBE/JA: Jump on not below nor equal/above", operation, "0b01110111"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JNP/JPO: Jump on not parity/odd", operation, "0b01111011"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JNO: Jump on not overflow", operation, "0b01110001"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JNS: Jump on not sign", operation, "0b01111001"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("JCXZ: Jump if CX register is zero", operation, "0b11100011"):
			instruction, err = jumpConditionally(operation, d)

		// Loops
		case d.matchPattern("LOOP: Loop CX times", operation, "0b11100010"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("LOOPZ/LOOPE: Loop while zero/equal", operation, "0b11100001"):
			instruction, err = jumpConditionally(operation, d)
		case d.matchPattern("LOOPNZ/LOOPNE: Loop while not zero/not equal", operation, "0b11100000"):
			instruction, err = jumpConditionally(operation, d)

		// Interrupts
		case d.matchPattern("INT: Type specified", operation, "0b11001101"):
			instruction, err = interruptWithType(operation, d)
		case d.matchPattern("INT: type 3", operation, "0b11001100"):
			instruction, err = interruptType3(operation, d) // Breakpoint
		case d.matchPattern("INTO: interrupt on overflow", operation, "0b11001110"):
			instruction, err = interruptOnOverflow(operation, d)
		case d.matchPattern("IRET: Interrupt return", operation, "0b11001111"):
			instruction, err = interruptReturn(operation, d)

		// Processor control
		case d.matchPattern("CLC: Clear carry", operation, "0b11111000"):
			instruction, err = clc(operation, d)
		case d.matchPattern("CMC: Complement carry", operation, "0b11110101"):
			instruction, err = cmc(operation, d)
		case d.matchPattern("STC: Set carry", operation, "0b11111001"):
			instruction, err = stc(operation, d)
		case d.matchPattern("CLD: Clear direction", operation, "0b11111100"):
			instruction, err = cld(operation, d)
		case d.matchPattern("STD: set direction", operation, "0b11111101"):
			instruction, err = std(operation, d)
		case d.matchPattern("CLI: Clear interrupt", operation, "0b11111010"):
			instruction, err = cli(operation, d)
		case d.matchPattern("STI: Set interrupt", operation, "0b11111011"):
			instruction, err = sti(operation, d)
		case d.matchPattern("HLT: Halt", operation, "0b11110100"):
			instruction, err = hlt(operation, d)
		case d.matchPattern("WAIT: Wait", operation, "0b10011011"):
			instruction, err = wait(operation, d)

		default:
			panic(fmt.Sprintf("AssertionError: unexpected operation %b", int(operation)))
		}

		if err != nil {
			return nil, err
		}

		if prefix != "" {
			instruction = prefix + instruction
		}

		d.appendInstruction(instructionPointer, instruction)
	}

	return d.GetDecoded(), nil
}

func (d *Decoder) next() (byte, bool) {
	if len(d.bytes) > d.pos {
		b := d.bytes[d.pos]
		d.pos += 1
		return b, true
	} else {
		return 0, false
	}
}

func (d *Decoder) peekNext() (byte, bool) {
	if len(d.bytes) > d.pos {
		return d.bytes[d.pos], true
	} else {
		return 0, false
	}
}

// 1 - next char
func (d *Decoder) peekForward(offset int) (byte, bool) {
	pos := 0
	if offset == 1 {
		pos = d.pos
	} else {
		pos = d.pos + offset
	}

	if len(d.bytes) > pos {
		return d.bytes[pos], true
	} else {
		return 0, false
	}
}

// pattern - 0b10011dwx, where any char except 0 or 1 is a wildcard
// can contain several bytes 0b10011dwx|0b__111___
func (d *Decoder) matchPattern(name string, candidate byte, pattern string) bool {
	const prefix = "0b"
	const separator = "|"

	bytePatterns := strings.Split(pattern, separator)

	for i, p := range bytePatterns {
		if !strings.HasPrefix(p, prefix) {
			panic(fmt.Errorf("pattern for '%s' must start with '0b'", name))
		}

		p = p[len(prefix):] // 0b

		if len(p) != 8 {
			panic(fmt.Errorf("pattern for '%s' must be 8 bits long", name))
		}

		b := candidate
		if i > 0 {
			var ok bool
			b, ok = d.peekForward(i)
			if ok == false {
				return false
			}
		}

		for offset, ch := range p {
			if ch != '0' && ch != '1' {
				continue
			}

			shift := 7 - offset
			bit := (b >> shift) & 0b1

			if bit == 1 && ch != '1' {
				return false
			}
			if bit == 0 && ch != '0' {
				return false
			}
		}
	}

	return true
}

// [mod|reg|r/m]
func (d *Decoder) decodeBinaryRegOrMem(instructionName string, mod byte, reg byte, rm byte, isWord bool, dir byte) (dest string, src string, err error) {
	verifyDirection(dir)
	regName := ""
	if isWord {
		regName = WordOperationRegisterFieldEncoding[reg]
	} else {
		regName = ByteOperationRegisterFieldEncoding[reg]
	}

	// MOV dest, src
	// ADD dest, src
	dest = ""
	src = ""

	switch mod {
	case MemoryModeNoDisplacementFieldEncoding:
		displacementValue := uint16(0)
		// the exception for the direct address - 16-bit displacement for the direct address
		if rm == 0b110 {
			displacementLow, ok := d.next()
			if ok == false {
				return dest, src, fmt.Errorf("expected to receive the Low displacement value for direct address in the '%s' instruction", instructionName)
			}
			displacementHigh, ok := d.next()
			if ok == false {
				return dest, src, fmt.Errorf("expected to receive the High displacement value for direct address in the '%s' instruction", instructionName)
			}
			displacementValue = binary.LittleEndian.Uint16([]byte{displacementLow, displacementHigh})
		}

		effectiveAddress := d.calculateEffectiveAddress(rm, displacementValue, MemoryModeNoDisplacementFieldEncoding)

		if dir == RegIsDestination {
			dest = regName
			src = effectiveAddress
		} else {
			dest = effectiveAddress
			src = regName
		}

	case MemoryMode8DisplacementFieldEncoding:
		displacementValue, ok := d.next()
		if ok == false {
			return dest, src, fmt.Errorf("expected to receive the displacement value for the '%s' instruction", instructionName)
		}
		effectiveAddress := d.calculateEffectiveAddress(rm, uint16(displacementValue), MemoryMode8DisplacementFieldEncoding)

		if dir == RegIsDestination {
			dest = regName
			src = effectiveAddress
		} else {
			dest = effectiveAddress
			src = regName
		}

	case MemoryMode16DisplacementFieldEncoding:
		displacementLow, ok := d.next()
		if ok == false {
			return dest, src, fmt.Errorf("expected to receive the Low displacement value for the '%s' instruction", instructionName)
		}
		displacementHigh, ok := d.next()
		if ok == false {
			return dest, src, fmt.Errorf("expected to receive the High displacement value for the '%s' instruction", instructionName)
		}

		displacementValue := binary.LittleEndian.Uint16([]byte{displacementLow, displacementHigh})
		effectiveAddress := d.calculateEffectiveAddress(rm, displacementValue, MemoryMode16DisplacementFieldEncoding)

		if dir == RegIsDestination {
			dest = regName
			src = effectiveAddress
		} else {
			dest = effectiveAddress
			src = regName
		}

	case RegisterModeFieldEncoding:
		rmRegisterName := ""
		if isWord {
			rmRegisterName = WordOperationRegisterFieldEncoding[rm]
		} else {
			rmRegisterName = ByteOperationRegisterFieldEncoding[rm]
		}

		if dir == RegIsDestination {
			dest = regName
			src = rmRegisterName
		} else {
			dest = rmRegisterName
			src = regName
		}
	default:
		panic("The mod field should only be 2 bits")
	}

	return dest, src, nil
}

// [xxx|w] [mod|xxx|r/m] [disp-lo] [disp-hi]
func (d *Decoder) decodeUnaryRegOrMem(instructionName string, mod byte, rm byte, isWord bool) (string, error) {
	regOrMem := ""

	switch mod {
	case MemoryModeNoDisplacementFieldEncoding:
		displacementValue := uint16(0)
		// the exception for the direct address - 16-bit displacement for the direct address
		if rm == 0b110 {
			displacementLow, ok := d.next()
			if ok == false {
				return "", fmt.Errorf("expected to receive the Low displacement value for direct address in the '%s' instruction", instructionName)
			}
			displacementHigh, ok := d.next()
			if ok == false {
				return "", fmt.Errorf("expected to receive the High displacement value for direct address in the '%s' instruction", instructionName)
			}
			displacementValue = binary.LittleEndian.Uint16([]byte{displacementLow, displacementHigh})
		}

		regOrMem = d.calculateEffectiveAddress(rm, displacementValue, MemoryModeNoDisplacementFieldEncoding)

	case MemoryMode8DisplacementFieldEncoding:
		displacementValue, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to receive the displacement value for the '%s' instruction", instructionName)
		}
		regOrMem = d.calculateEffectiveAddress(rm, uint16(displacementValue), MemoryMode8DisplacementFieldEncoding)

	case MemoryMode16DisplacementFieldEncoding:
		displacementLow, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to receive the Low displacement value for the '%s' instruction", instructionName)
		}
		displacementHigh, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to receive the High displacement value for the '%s' instruction", instructionName)
		}

		displacementValue := binary.LittleEndian.Uint16([]byte{displacementLow, displacementHigh})
		regOrMem = d.calculateEffectiveAddress(rm, displacementValue, MemoryMode16DisplacementFieldEncoding)

	case RegisterModeFieldEncoding:
		rmRegisterName := ""
		if isWord {
			rmRegisterName = WordOperationRegisterFieldEncoding[rm]
		} else {
			rmRegisterName = ByteOperationRegisterFieldEncoding[rm]
		}

		regOrMem = rmRegisterName
	default:
		panic("The mod field should only be 2 bits")
	}

	return regOrMem, nil
}

// [xxx|w] [data] [data if isWord]
// decodeImmediate decodes a constant byte or word
func (d *Decoder) decodeImmediate(instructionName string, isWord bool) (immediateValue uint16, err error) {
	immediateValue = uint16(0)

	if isWord {
		low, ok := d.next()
		if ok == false {
			return 0, fmt.Errorf("expected to get the immediate value (low) for the '%s' instruction", instructionName)
		}
		high, ok := d.next()
		if ok == false {
			return 0, fmt.Errorf("expected to get the immediate value (high) for the '%s' instruction", instructionName)
		}

		immediateValue = binary.LittleEndian.Uint16([]byte{low, high})
	} else {
		v, ok := d.next()
		if ok == false {
			return 0, fmt.Errorf("expected to get the immediate value for the '%s' instruction", instructionName)
		}
		immediateValue = uint16(v)
	}

	return immediateValue, nil
}

// [xxx|w] [addr-lo] [addr-hi]
func (d *Decoder) decodeAddress(instructionName string, isWord bool) (address uint16, err error) {
	address = uint16(0)

	if isWord {
		low, ok := d.next()
		if ok == false {
			return 0, fmt.Errorf("expected to get the address (low) for the '%s' instruction", instructionName)
		}
		high, ok := d.next()
		if ok == false {
			return 0, fmt.Errorf("expected to get the address (high) for the '%s' instruction", instructionName)
		}

		address = binary.LittleEndian.Uint16([]byte{low, high})
	} else {
		v, ok := d.next()
		if ok == false {
			return 0, fmt.Errorf("expected to get the address for the '%s' instruction", instructionName)
		}
		address = uint16(v)
	}

	return address, nil
}

// [xxxxxxx|w] [data] [data if w = 1]
func (d *Decoder) immediateWithAccumulator(instructionName string, operation byte) (regName string, immediateValue uint16, err error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	immediateValue, err = d.decodeImmediate(instructionName, isWord)
	if err != nil {
		return "", 0, err
	}

	regName = ""
	if isWord {
		regName = "ax"
	} else {
		regName = "al"
	}

	return regName, immediateValue, nil
}

// [xxxxxx|d|w] [mod|reg|r/m] [disp-lo?] [disp-hi?]
func (d *Decoder) regOrMemWithReg(instructionName string, operation byte) (dest string, src string, err error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	// direction is the 2nd bit
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	dir := (operation >> 1) & 0b00000001
	verifyDirection(dir)

	operand, ok := d.next()
	if ok == false {
		return "", "", fmt.Errorf("expected to get an operand for the '%s' instruction", instructionName)
	}

	// mod is the 2 high bits
	mod, reg, rm := decodeOperand(operand)

	return d.decodeBinaryRegOrMem(instructionName, mod, reg, rm, isWord, dir)
}

// [xxxxxxx|w] [mod|<regPattern>|r/m] [disp-lo?] [disp-hi?] [data] [data if s|w = 0|1]
func (d *Decoder) buildImmediateWithRegOrMemInstruction(mnemonic string, regPattern byte, instructionName string, operation byte) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the '%s' instruction", instructionName)
	}

	mod, reg, rm := decodeOperand(operand)

	if reg != regPattern {
		return "", fmt.Errorf("expected the reg field to be %.3b for the '%s' instruction", regPattern, instructionName)
	}

	dest, err := d.decodeUnaryRegOrMem(instructionName, mod, rm, isWord)
	if err != nil {
		return "", err
	}

	immediateValue, err := d.decodeImmediate(instructionName, isWord)
	if err != nil {
		return "", err
	}

	size := ""
	if isWord {
		size = "word"
	} else {
		size = "byte"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "%s %s, ", mnemonic, dest)

	// we need to specify the size of the value
	if mod != RegisterModeFieldEncoding {
		// mov [bp + 75], byte 42
		// add [bp + 75], byte 12
		// sub [bp + 75], word 512
		builder.WriteString(size + " ")
	}

	fmt.Fprintf(&builder, "%d", immediateValue)

	signedValue := int16(immediateValue)
	if signedValue < 0 {
		fmt.Fprintf(&builder, " ; or %d", signedValue)
	}

	builder.WriteString("\n")

	return builder.String(), nil
}

func (d *Decoder) calculateEffectiveAddress(rm byte, displacementValue uint16, mod byte) string {
	address := ""
	if mod == MemoryModeNoDisplacementFieldEncoding {
		equation := ""
		// the exception for the direct address - 16-bit displacement for the direct address
		if rm == 0b110 {
			equation = strconv.Itoa(int(displacementValue))
		} else {
			equation = EffectiveAddressEquation[rm]
		}

		address = fmt.Sprintf("[%s]", equation)
	} else if mod == MemoryMode8DisplacementFieldEncoding {
		equation := EffectiveAddressEquation[rm]
		signed := int8(uint8(displacementValue))
		if signed < 0 {
			address = fmt.Sprintf("[%s - %d]", equation, ^signed+1) // remove the sign 1111 1011 -> 0000 0101
		} else {
			address = fmt.Sprintf("[%s + %d]", equation, displacementValue)
		}
	} else if mod == MemoryMode16DisplacementFieldEncoding {
		equation := EffectiveAddressEquation[rm]
		signed := int16(displacementValue)
		if signed < 0 {
			address = fmt.Sprintf("[%s - %d]", equation, ^signed+1) // remove the sign 1111 1011 -> 0000 0101
		} else {
			address = fmt.Sprintf("[%s + %d]", equation, displacementValue)
		}
	} else {
		panic(fmt.Errorf("AssertionError: Unknown mod for effective address calculation. %.3b", mod))
	}

	if d.segment != "" {
		return fmt.Sprintf("%s:%s", d.segment, address)
	} else {
		return address
	}
}

// [mod|reg|r/m]
func decodeOperand(operand byte) (mod byte, reg byte, rm byte) {
	mod = operand >> 6
	reg = (operand >> 3) & 0b00000111
	rm = operand & 0b00000111
	return
}

func verifyOperationType(t byte) {
	if t != WordOperation && t != ByteOperation {
		panic(fmt.Sprintf("The operation type should be a binary value (word or byte). Got %d instead", t))
	}
}

func verifyDirection(dir byte) {
	if dir != RegIsDestination && dir != RegIsSource {
		panic(fmt.Sprintf("The direction should be a binary value (dest or src). Got %d instead", dir))
	}
}

func verifySign(sign byte) {
	if sign != SignExtension && sign != NoSignExtension {
		panic(fmt.Sprintf("The sign should be a binary value (sign or no sign). Got %d instead", sign))
	}
}

func verifyCount(count byte) {
	if count != CountByOne && count != CountByCL {
		panic(fmt.Sprintf("The count should be a binary value (1 or CL). Got %d instead", count))
	}
}

func verifyLoop(z byte) {
	if z != LoopWhileNotZero && z != LoopWhileZero {
		panic(fmt.Sprintf("The z should be a binary value (zero or not zero). Got %d instead", z))
	}
}
