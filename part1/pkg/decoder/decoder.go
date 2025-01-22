package decoder

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// Instruction reference for 8086 CPU (https://edge.edx.org/c4x/BITSPilani/EEE231/asset/8086_family_Users_Manual_1_.pdf | page 161(pdf))
// The "Instruction reference"👆
// [opcode|d|m] [mod|reg|r/m] [displacement-low] [displacement-high] [data-low] [data-high]
//    6    1 1    2   3   3
// The intel x86 processors use Little Endian, so the low byte comes first
// Disp-lo (Displacement low) - Low-order byte of optional 8- or 16-bit __unsigned__ displacement; MOD indicates if present.
// Disp-hi (Displacement High) - High-order byte of optional 16-bit __unsigned__ displacement; MOD indicates if present.
// Data-lo (Data low) - Low-order byte of 16-bit immediate constant.
// Data-hi (Data high) - High-order byte of 16-bit immediate constant.

// D bit - Direction of the operation
const (
	RegIsSource      = 0
	RegIsDestination = 1
)

// W bit
const (
	ByteOperation = byte(0)
	WordOperation = byte(1)
)

// S bit
const (
	NoSignExtension = byte(0)
	SignExtension   = byte(1) // Sign extend 8-bit immediate data to 16 bits if W=1
)

// MOD field
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

// REG (Register) field encoding - ByteOperationRegisterFieldEncoding & WordOperationRegisterFieldEncoding
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
	0b000: "es", // extra segment
	0b001: "cs", // code segment
	0b010: "ss", // stack segment
	0b011: "ds", // data segment
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
	next  *instructionNode
}

type Decoder struct {
	bytes         []byte
	pos           int
	head          *instructionNode
	tail          *instructionNode
	numberOfNodes int
	labels        map[int]string // pos:label
	cacheKey      string
	decoded       []byte
}

func NewDecoder(bytes []byte) *Decoder {
	return &Decoder{
		bytes:         bytes,
		pos:           0,
		head:          nil,
		tail:          nil,
		numberOfNodes: 0,
		labels:        make(map[int]string),
		cacheKey:      "",
		decoded:       make([]byte, 0),
	}
}

func (d *Decoder) appendInstruction(pos int, value string) {
	n := instructionNode{
		value: value,
		pos:   pos,
		next:  nil,
	}

	if d.head == nil && d.tail == nil {
		d.head = &n
		d.tail = &n
	} else {
		if d.head == nil || d.tail == nil {
			panic("Assertion error: head and tail should be either both nil or both not nil")
		}

		d.tail.next = &n
		d.tail = &n
	}
	d.numberOfNodes += 1
}

func (d *Decoder) computeCacheKey() string {
	return fmt.Sprintf("n=%d;l=%d", d.numberOfNodes, len(d.labels))
}

func (d *Decoder) GetDecoded() []byte {
	cacheKey := d.computeCacheKey()
	if cacheKey == d.cacheKey {
		return d.decoded
	}

	n := d.head
	iter := 0
	d.decoded = d.decoded[:0] // reuse the same array
	for n != nil {
		iter += 1
		if iter > 15_000_000_000 {
			panic("[GetDecoded] Iterated over 15 billion nodes. This is probably a mistake")
		}

		instruction := ""
		label, ok := d.labels[n.pos-1] // as we start counting instructions from 1, instead of 0
		if ok {
			instruction += fmt.Sprintf("%s:\n", label)
		}

		instruction += n.value
		d.decoded = append(d.decoded, []byte(instruction)...)

		n = n.next
	}

	d.cacheKey = cacheKey
	return d.decoded
}

func (d *Decoder) Decode() ([]byte, error) {
	d.pos = 0
	for {
		// Section 2.7 Instruction set. p. 2-30
		instruction := ""
		var err error
		operation, ok := d.next()
		if ok == false {
			break
		}

		instructionPointer := d.pos

		// Table 4-12. 8086 Instruction Encoding
		switch {
		// MOV
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

		// XCHG
		case d.matchPattern("XCHG: Register/memory with register", operation, "0b1000011w"):
			panic("TODO: XCHG: Register/memory with register")
		case d.matchPattern("XCHG: register with accumulator", operation, "0b10010reg"):
			panic("TODO: XCHG: register with accumulator")

		// IN
		case d.matchPattern("IN: Fixed port", operation, "0b1110010w"):
			panic("TODO: IN: Fixed port")
		case d.matchPattern("IN: Variable port", operation, "0b1110110w"):
			panic("TODO: IN: Variable port")

		// OUT
		case d.matchPattern("OUT: Fixed port", operation, "0b1110011w"):
			panic("TODO: OUT: Fixed port")
		case d.matchPattern("OUT: Variable port", operation, "0b1110111w"):
			panic("TODO: OUT: Variable port")
		case d.matchPattern("OUT: XLAT - Translate byte to AL", operation, "0b11010111"):
			panic("TODO: OUT: XLAT - Translate byte to AL")
		case d.matchPattern("OUT: LEA - Load effective address to register", operation, "0b10001101"):
			panic("TODO: OUT: LEA - Load effective address to register")
		case d.matchPattern("OUT: LDS - Load pointer to DS", operation, "0b11000101"):
			panic("TODO: OUT: LDS - Load pointer to DS")
		case d.matchPattern("OUT: LES - Load pointer to ES", operation, "0b11000100"):
			panic("TODO: OUT: LES - Load pointer to ES")
		case d.matchPattern("OUT: LAHF - Load AH with flags", operation, "0b10011111"):
			panic("TODO: OUT: LAHF - Load AH with flags")
		case d.matchPattern("OUT: SAHF - Store AH into flags", operation, "0b10011110"):
			panic("TODO: OUT: SAHF - Store AH into flags")
		case d.matchPattern("OUT: PUSHF - Push flags", operation, "0b10011100"):
			panic("TODO: OUT: PUSHF - Push flags")
		case d.matchPattern("OUT: POPF - Pop flags", operation, "0b10011101"):
			panic("TODO: OUT: POPF - Pop flags")

		// ADD
		case d.matchPattern("ADD: Reg/memory with register to either", operation, "0b000000dw"):
			instruction, err = addRegOrMemToReg(operation, d)
		case d.matchPattern("ADD: Immediate to register/memory", operation, "0b100000sw|0b__000___"):
			instruction, err = addImmediateToRegOrMem(operation, d)
		case d.matchPattern("ADD: Immediate to accumulator", operation, "0b0000010w"):
			instruction, err = addImmediateToAccumulator(operation, d)

		// SUB
		case d.matchPattern("SUB: Reg/memory and register to either", operation, "0b001010dw"):
			instruction, err = subRegOrMemFromReg(operation, d)
		case d.matchPattern("SUB: Immediate to register/memory", operation, "0b100000sw|0b__101___"):
			instruction, err = subImmediateFromRegOrMem(operation, d)
		case d.matchPattern("SUB: Immediate from accumulator", operation, "0b0010110w"):
			instruction, err = subImmediateFromAccumulator(operation, d)

		// CMP
		case d.matchPattern("CMP: Reg/memory and register", operation, "0b001110dw"):
			instruction, err = cmpRegOrMemWithReg(operation, d)
		case d.matchPattern("CMP: Immediate with register/memory", operation, "0b100000sw|0b__111___"):
			instruction, err = cmpImmediateWithRegOrMem(operation, d)
		case d.matchPattern("CMP: Immediate from accumulator", operation, "0b0011110w"):
			instruction, err = cmpImmediateWithAccumulator(operation, d)

		// JMP (unconditional)
		case d.matchPattern("JMP: Direct within segment", operation, "0b11101001"):
			panic("TODO: JMP: Direct within segment")
		case d.matchPattern("JMP: Direct within segment-short", operation, "0b11101011"):
			panic("TODO: JMP: Direct within segment-short")
		case d.matchPattern("JMP: Indirect within segment", operation, "0b11111111|0b__100___"):
			panic("TODO: JMP: Indirect within segment")
		case d.matchPattern("JMP: Direct intersegment", operation, "0b11101010"):
			panic("TODO: JMP: Direct intersegment")
		case d.matchPattern("JMP: Indirect intersegment", operation, "0b11111111|0b__101___"):
			panic("TODO: JMP: Indirect intersegment")

		// RET
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
		default:
			panic(fmt.Sprintf("AssertionError: unexpected operation %b", int(operation)))
		}

		if err != nil {
			return nil, err
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
		equation := ""
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
			displacementValue := binary.LittleEndian.Uint16([]byte{displacementLow, displacementHigh})
			equation = strconv.Itoa(int(displacementValue))
		} else {
			equation = EffectiveAddressEquation[rm]
		}

		effectiveAddress := fmt.Sprintf("[%s]", equation)

		if dir == RegIsDestination {
			dest = regName
			src = effectiveAddress
		} else {
			dest = effectiveAddress
			src = regName
		}

	case MemoryMode8DisplacementFieldEncoding:
		displacement, ok := d.next()
		if ok == false {
			return dest, src, fmt.Errorf("expected to receive the displacement value for the '%s' instruction", instructionName)
		}
		equation := EffectiveAddressEquation[rm]
		signed := int8(displacement)
		effectiveAddress := ""
		if signed < 0 {
			effectiveAddress = fmt.Sprintf("[%s - %d]", equation, ^signed+1) // remove the sign 1111 1011 -> 0000 0101
		} else {
			effectiveAddress = fmt.Sprintf("[%s + %d]", equation, displacement)
		}

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

		equation := EffectiveAddressEquation[rm]
		displacementValue := binary.LittleEndian.Uint16([]byte{displacementLow, displacementHigh})
		effectiveAddress := ""
		signed := int16(displacementValue)
		if signed < 0 {
			effectiveAddress = fmt.Sprintf("[%s - %d]", equation, ^signed+1) // remove the sign 1111 1011 -> 0000 0101
		} else {
			effectiveAddress = fmt.Sprintf("[%s + %d]", equation, displacementValue)
		}

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
		equation := ""
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
			displacementValue := binary.LittleEndian.Uint16([]byte{displacementLow, displacementHigh})
			equation = strconv.Itoa(int(displacementValue))
		} else {
			equation = EffectiveAddressEquation[rm]
		}

		regOrMem = fmt.Sprintf("[%s]", equation)

	case MemoryMode8DisplacementFieldEncoding:
		displacement, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to receive the displacement value for the '%s' instruction", instructionName)
		}
		equation := EffectiveAddressEquation[rm]
		signed := int8(displacement)
		if signed < 0 {
			regOrMem = fmt.Sprintf("[%s - %d]", equation, ^signed+1) // remove the sign 1111 1011 -> 0000 0101
		} else {
			regOrMem = fmt.Sprintf("[%s + %d]", equation, displacement)
		}

	case MemoryMode16DisplacementFieldEncoding:
		displacementLow, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to receive the Low displacement value for the '%s' instruction", instructionName)
		}
		displacementHigh, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to receive the High displacement value for the '%s' instruction", instructionName)
		}

		equation := EffectiveAddressEquation[rm]
		displacementValue := binary.LittleEndian.Uint16([]byte{displacementLow, displacementHigh})
		signed := int16(displacementValue)
		if signed < 0 {
			regOrMem = fmt.Sprintf("[%s - %d]", equation, ^signed+1) // remove the sign 1111 1011 -> 0000 0101
		} else {
			regOrMem = fmt.Sprintf("[%s + %d]", equation, displacementValue)
		}

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
