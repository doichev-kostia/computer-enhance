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

// Common pattern
//
// |  op  | pattern |
// ------------------
// | ADD  | 000     |
// | SUB  | 101     |
// | CMP  | 111     |

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

func (d *Decoder) GetDecoded() []byte {
	return d.decoded
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
		case matchPattern("MOV: Register/memory to/from register", operation, "0b100010dw"):
			instruction, err = moveRegMemToReg(operation, d)
		case matchPattern("MOV: immediate to register/memory", operation, "0b1100011w"):
			instruction, err = moveImmediateToRegOrMem(operation, d)
		case matchPattern("MOV: immediate to register", operation, "0b1011wreg"):
			instruction, err = moveImmediateToReg(operation, d)
		case matchPattern("MOV: memory to accumulator", operation, "0b1010000w"):
			instruction, err = moveMemoryToAccumulator(operation, d)
		case matchPattern("MOV: accumulator to memory", operation, "0b1010001w"):
			instruction, err = moveAccumulatorToMemory(operation, d)
		case matchPattern("ADD: Reg/memory with register to either", operation, "0b000000dw"):
			instruction, err = addRegOrMemWithReg(operation, d)
		case matchPattern("ADD: Immediate to register/memory", operation, "0b100000sw"):
			instruction, err = addImmediateToRegOrMem(operation, d)
		case matchPattern("ADD: Immediate to accumulator", operation, "0b0000010w"):
			instruction, err = addImmediateToAccumulator(operation, d)

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

// pattern - 0b10011dwx, where any char except 0 or 1 is a wildcard
func matchPattern(name string, b byte, pattern string) bool {
	const prefix = "0b"

	if !strings.HasPrefix(pattern, prefix) {
		panic(fmt.Errorf("pattern for '%s' must start with '0b'", name))
	}

	pattern = pattern[len(prefix):] // 0b

	if len(pattern) != 8 {
		panic(fmt.Errorf("pattern for '%s' must be 8 bits long", name))
	}

	for i, ch := range pattern {
		if ch != '0' && ch != '1' {
			continue
		}

		offset := 7 - i
		bit := (b >> offset) & 0b1

		if bit == 1 && ch != '1' {
			return false
		}
		if bit == 0 && ch != '0' {
			return false
		}
	}

	return true
}

// [mod|reg|r/m]
func (d *Decoder) decodeRegOrMem(instructionName string, mod byte, reg byte, rm byte, isWord bool, dir byte) (dest string, src string, err error) {
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

// [xxx|w] [mod|000|r/m] [disp-lo] [disp-hi] [data-lo] [data-hi]
func (d *Decoder) decodeImmediateToRegOrMem(instructionName string, mod byte, reg byte, rm byte, isWord bool) (dest string, err error) {
	// must be 000 according to the "Instruction reference"
	if reg != 0 {
		return "", fmt.Errorf("expected the reg field to be 000 for the '%s' instruction", instructionName)
	}

	// mov dest, immediateValue
	// add dest, immediateValue
	dest = ""

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

		dest = fmt.Sprintf("[%s]", equation)

	case MemoryMode8DisplacementFieldEncoding:
		displacement, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to receive the displacement value for the '%s' instruction", instructionName)
		}
		equation := EffectiveAddressEquation[rm]
		signed := int8(displacement)
		if signed < 0 {
			dest = fmt.Sprintf("[%s - %d]", equation, ^signed+1) // remove the sign 1111 1011 -> 0000 0101
		} else {
			dest = fmt.Sprintf("[%s + %d]", equation, displacement)
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
			dest = fmt.Sprintf("[%s - %d]", equation, ^signed+1) // remove the sign 1111 1011 -> 0000 0101
		} else {
			dest = fmt.Sprintf("[%s + %d]", equation, displacementValue)
		}

	case RegisterModeFieldEncoding:
		rmRegisterName := ""
		if isWord {
			rmRegisterName = WordOperationRegisterFieldEncoding[rm]
		} else {
			rmRegisterName = ByteOperationRegisterFieldEncoding[rm]
		}

		dest = rmRegisterName
	default:
		panic("The mod field should only be 2 bits")
	}

	return dest, nil
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
