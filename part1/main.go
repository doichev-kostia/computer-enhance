package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
)

// Instruction reference for 8086 CPU (https://edge.edx.org/c4x/BITSPilani/EEE231/asset/8086_family_Users_Manual_1_.pdf | page 161(pdf))
// The "Instruction reference"👆
// [opcode|d|m] [mod|reg|r/m]
//    6    1 1    2   3   3

// Direction of the operation (the d bit)
const (
	RegIsSource      = 0
	RegIsDestination = 1
)

// W bit
const (
	ByteOperation = byte(0)
	WordOperation = byte(1)
)

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
var EffectiveAddressEquation = map[byte]string{
	0b000: "bx + si",
	0b001: "bx + di",
	0b010: "bp + si",
	0b011: "bp + di",
	0b100: "si",
	0b101: "di",
	0b110: "bp",
	0b111: "bx",
}

// MOD
const (
	MemoryModeNoDisplacementFieldEncoding = 0b00
	MemoryMode8DisplacementFieldEncoding  = 0b01
	MemoryMode16DisplacementFieldEncoding = 0b10
	RegisterModeFieldEncoding             = 0b11
)

func printHead(filename string) string {
	return fmt.Sprintf("; %s\nbits 16\n\n", filename)
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
			panic("todo")
		// MOV: immediate to register
		case pattern(operation, 0b1011):
			instruction, err = immediateToReg(operation, d)
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

// func immediateToMemOrReg(operation byte, d *Decoder) (string, error) {
// 	// the & 0b00 is to discard all the other bits and leave the ones we care about
// 	operationType := operation & 0b00000001
// 	isWord := operationType == WordOperation
//
// 	operand, ok := d.next()
// 	if ok == false {
// 		return "", fmt.Errorf("Expected to get an operand for the 'immediate to register/memory' instruction")
// 	}
//
// 	mod := operand >> 6
// }

// 1011|w|reg  data  data if w = 1
func immediateToReg(operation byte, d *Decoder) (string, error) {
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

	immediateValue := 0

	if isWord {
		low, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the immediate value (low) for the 'immediate to register' instruction")
		}
		high, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the immediate value (high) for the 'immediate to register' instruction")
		}

		v := binary.LittleEndian.Uint16([]byte{low, high})
		immediateValue = int(v)
	} else {
		v, ok := d.next()
		if ok == false {
			return "", fmt.Errorf("expected to get the immediate value for the 'immediate to register' instruction")
		}
		immediateValue = int(v)
	}

	return fmt.Sprintf("mov %s, %d\n", regName, immediateValue), nil
}

// 100010dw mod reg r/m disp-lo disp-hi
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
		effectiveAddress := fmt.Sprintf("[%s + %d]", equation, displacement)

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
		effectiveAddress := fmt.Sprintf("[%s + %d]", equation, displacementValue)

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
