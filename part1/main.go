package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
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

func newDecoder(bytes []byte) *Decoder {
	return &Decoder{
		bytes:   bytes,
		pos:     0,
		decoded: make([]byte, 0),
	}
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

func (d *Decoder) decode() ([]byte, error) {
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

// [1100011|w] [mod|000|r/m] [disp-lo] [disp-hi] [data] [data if w=1]
func moveImmediateToRegOrMem(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the 'immediate to register/memory' instruction")
	}

	mod := operand >> 6
	reg := (operand >> 3) & 0b00000111
	rm := operand & 0b00000111

	// must be 000 according to the "Instruction reference"
	if reg != 0 {
		return "", fmt.Errorf("expected the reg field to be 000 for the 'immediate to register/memory' instruction")
	}

	// mov dest, immediateValue
	dest := ""
	immediateValue := uint16(0)

	// dest
	switch mod {
	case MemoryModeNoDisplacementFieldEncoding:
		equation := ""
		// the exception for the direct address - 16-bit displacement for the direct address
		if rm == 0b110 {
			displacementLow, ok := d.next()
			if ok == false {
				return "", fmt.Errorf("expected to receive the Low displacement value for direct address in the 'immediate to register/memory' instruction")
			}
			displacementHigh, ok := d.next()
			if ok == false {
				return "", fmt.Errorf("expected to receive the High displacement value for direct address in the 'immediate to register/memory' instruction")
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
			return "", fmt.Errorf("expected to receive the displacement value for the 'immediate to register/memory' instruction")
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
			return "", fmt.Errorf("expected to receive the Low displacement value for the 'immediate to register/memory' instruction")
		}
		displacementHigh, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to receive the High displacement value for the 'immediate to register/memory' instruction")
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

	// immediateValue
	if isWord {
		low, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the immediate value (low) for the 'immediate to register/memory' instruction")
		}
		high, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the immediate value (high) for the 'immediate to register/memory' instruction")
		}

		immediateValue = binary.LittleEndian.Uint16([]byte{low, high})
	} else {
		v, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the immediate value for the 'immediate to register/memory' instruction")
		}
		immediateValue = uint16(v)
	}

	size := ""
	if isWord {
		size = "word"
	} else {
		size = "byte"
	}

	signedValue := int16(immediateValue)
	var builder strings.Builder
	fmt.Fprintf(&builder, "mov %s, ", dest)

	// we need to specify the size of the value
	if mod != RegisterModeFieldEncoding {
		// mov [bp + 75], byte 12
		// mov [bp + 75], word 512
		builder.WriteString(size + " ")
	}

	fmt.Fprintf(&builder, "%d", immediateValue)

	if signedValue < 0 {
		fmt.Fprintf(&builder, " ; or %d", signedValue)
	}

	builder.WriteString("\n")
	return builder.String(), nil
}

// [1011|w|reg]  [data]  [data if w = 1]
func moveImmediateToReg(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := (operation >> 3) & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	reg := operation & 0b00000111
	regName := ""
	if isWord {
		regName = WordOperationRegisterFieldEncoding[reg]
	} else {
		regName = ByteOperationRegisterFieldEncoding[reg]
	}

	immediateValue := uint16(0)

	if isWord {
		low, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the immediate value (low) for the 'immediate to register' instruction")
		}
		high, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the immediate value (high) for the 'immediate to register' instruction")
		}

		immediateValue = binary.LittleEndian.Uint16([]byte{low, high})
	} else {
		v, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the immediate value for the 'immediate to register' instruction")
		}
		immediateValue = uint16(v)
	}

	signedValue := int16(immediateValue)

	if signedValue < 0 {
		return fmt.Sprintf("mov %s, %d ; or %d\n", regName, immediateValue, signedValue), nil
	} else {
		return fmt.Sprintf("mov %s, %d\n", regName, immediateValue), nil
	}
}

// [100010|d|w] [mod|reg|r/m] [disp-lo] [disp-hi]
func moveRegMemToReg(operation byte, d *Decoder) (string, error) {
	// direction is the 2nd bit
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	dir := (operation >> 1) & 0b00000001
	verifyDirection(dir)

	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the 'Register/memory to/from register' instruction")
	}

	// mod is the 2 high bits
	mod := operand >> 6
	reg := (operand >> 3) & 0b00000111
	rm := operand & 0b00000111

	regName := ""
	if isWord {
		regName = WordOperationRegisterFieldEncoding[reg]
	} else {
		regName = ByteOperationRegisterFieldEncoding[reg]
	}

	// MOV dest, src
	dest := ""
	src := ""

	switch mod {
	case MemoryModeNoDisplacementFieldEncoding:
		equation := ""
		// the exception for the direct address - 16-bit displacement for the direct address
		if rm == 0b110 {
			displacementLow, ok := d.next()
			if ok == false {
				return "", fmt.Errorf("expected to receive the Low displacement value for direct address in the 'Register/memory to/from register' instruction")
			}
			displacementHigh, ok := d.next()
			if ok == false {
				return "", fmt.Errorf("expected to receive the High displacement value for direct address in the 'Register/memory to/from register' instruction")
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
			return "", fmt.Errorf("expected to receive the displacement value for the 'Register/memory to/from register' instruction")
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
			return "", fmt.Errorf("expected to receive the Low displacement value for the 'Register/memory to/from register' instruction")
		}
		displacementHigh, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to receive the High displacement value for the 'Register/memory to/from register' instruction")
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

	return fmt.Sprintf("mov %s, %s\n", dest, src), nil
}

// [1010000w] [addr-lo] [addr-hi]
func moveMemoryToAccumulator(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	address := uint16(0)

	if isWord {
		low, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the address value (low) for the 'memory to accumulator' instruction")
		}
		high, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the address value (high) for the 'memory to accumulator' instruction")
		}

		address = binary.LittleEndian.Uint16([]byte{low, high})
	} else {
		v, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the address value for the 'memory to accumulator' instruction")
		}
		address = uint16(v)
	}

	return fmt.Sprintf("mov ax, [%d]\n", address), nil
}

// [1010001w] [addr-lo] [addr-hi]
func moveAccumulatorToMemory(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	address := uint16(0)

	if isWord {
		low, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the address value (low) for the 'accumulator to address' instruction")
		}
		high, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the address value (high) for the 'accumulator to address' instruction")
		}

		address = binary.LittleEndian.Uint16([]byte{low, high})
	} else {
		v, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the address value for the 'accumulator to address' instruction")
		}
		address = uint16(v)
	}

	return fmt.Sprintf("mov [%d], ax\n", address), nil
}

func main() {
	// 1 - program name, 2 - filename
	if len(os.Args) < 2 {
		exit(fmt.Errorf("invalid number of arguments, expected at least one for the filename\n"))
	}

	filename := os.Args[1]
	if !fileExists(filename) {
		exit(fmt.Errorf("The specified file %s doesn't exist\n", filename))
	}

	bytes, err := os.ReadFile(filename)
	if err != nil {
		exit(fmt.Errorf("Failed to read the file %s. Error = %w\n", filename, err))
	}

	decoder := newDecoder(bytes)
	decoded, err := decoder.decode()

	if err != nil {
		exit(err)
	}

	contents := printHead(filename) + string(decoded)

	fmt.Print(contents)
}

func printHead(filename string) string {
	return fmt.Sprintf("; %s\nbits 16\n\n", filename)
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

func pattern(b byte, pattern byte) bool {
	// 0b100010 -> 6 bits
	// 0b1011 -> 4 bits
	bits := int(math.Trunc(math.Log2(float64(pattern))) + 1)
	remainder := 8 - bits
	v := b >> remainder

	return v == pattern
}

func exit(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)

	return !errors.Is(err, os.ErrNotExist)
}
