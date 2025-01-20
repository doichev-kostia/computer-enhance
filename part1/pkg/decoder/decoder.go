package decoder

import (
	"fmt"
	"math"
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

// S field
const (
	NoSignExtension = byte(0)
	SignExtension   = byte(1)
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

type Decoder struct {
	bytes   []byte
	pos     int
	decoded []byte
}

func NewDecoder(bytes []byte) *Decoder {
	return &Decoder{
		bytes:   bytes,
		pos:     0,
		decoded: make([]byte, 0),
	}
}

func (d *Decoder) Decode() ([]byte, error) {
	d.pos = 0
	for {
		instruction := ""
		var err error
		operation, ok := d.next()
		if ok == false {
			break
		}

		// Opcode
		switch {
		// MOV: Register/memory to/from register
		case pattern(operation, 0b100010):
			instruction, err = moveRegMemToReg(operation, d)
		// MOV: immediate to register/memory
		case pattern(operation, 0b1100011):
			instruction, err = moveImmediateToRegOrMem(operation, d)
		// MOV: immediate to register
		case pattern(operation, 0b1011):
			instruction, err = moveImmediateToReg(operation, d)
		// MOV: memory to accumulator
		case pattern(operation, 0b1010000):
			instruction, err = moveMemoryToAccumulator(operation, d)
		// MOV: accumulator to memory
		case pattern(operation, 0b1010001):
			instruction, err = moveAccumulatorToMemory(operation, d)

		default:
			panic(fmt.Sprintf("AssertionError: unexpected operation %b", int(operation)))
		}

		if err != nil {
			return nil, err
		}

		d.decoded = append(d.decoded, []byte(instruction)...)
	}

	return d.decoded, nil
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

func pattern(b byte, pattern byte) bool {
	// 0b100010 -> 6 bits
	// 0b1011 -> 4 bits
	bits := int(math.Trunc(math.Log2(float64(pattern))) + 1)
	remainder := 8 - bits
	v := b >> remainder

	return v == pattern
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
