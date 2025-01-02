package main

import (
	"errors"
	"fmt"
	"math"
	"os"
)

// Instruction reference for 8086 CPU (https://edge.edx.org/c4x/BITSPilani/EEE231/asset/8086_family_Users_Manual_1_.pdf | page 161(pdf))
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

// MOD
const (
	RegisterModeFieldEncoding = 0b11
)

func printHead(filename string) string {
	return fmt.Sprintf("; %s\nbits 16\n\n", filename)
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

	i := 0
	decoded := make([]byte, 0)
	for len(bytes) > i {
		instruction := ""
		operation := bytes[i]
		i += 1

		operationType := ByteOperation
		dir := byte(0)

		// Opcode
		switch {
		// MOV: Register/memory to/from register
		case pattern(operation, 0b100010):
			// direction is the 2nd bit
			// the & 0b00 is to discard all the other bits and leave the ones we care about
			dir = (operation >> 1) & 0b00000001
			// the & 0b00 is to discard all the other bits and leave the ones we care about
			operationType = operation & 0b00000001
			isWord := operationType == WordOperation

			operand := bytes[i]
			i += 1

			// mod is the 2 high bits
			mod := operand >> 6
			if mod != RegisterModeFieldEncoding {
				exit(fmt.Errorf("Expected to only have operations between registers; mod = 11"))
			}

			// REG
			leftReg := (operand >> 3) & 0b00000111
			// r/m, but we only handle registers
			rightReg := operand & 0b00000111

			left := ""
			right := ""
			if isWord {
				left = WordOperationRegisterFieldEncoding[leftReg]
				right = WordOperationRegisterFieldEncoding[rightReg]
			} else {
				left = ByteOperationRegisterFieldEncoding[leftReg]
				right = ByteOperationRegisterFieldEncoding[rightReg]
			}

			dest := ""
			src := ""

			if dir == RegIsDestination {
				dest = left
				src = right
			} else if dir == RegIsSource {
				dest = right
				src = left
			} else {
				panic("Assertion Error: The destination (D) is a boolean value")
			}
			instruction = fmt.Sprintf("mov %s, %s\n", dest, src)
		// MOV: immediate to register/memory
		case pattern(operation, 0b1100011):
			panic("todo")
		// MOV: immediate to register
		case pattern(operation, 0b1011):
			panic("todo")

		default:
			panic(fmt.Sprintf("AssertionError: unexpected operation %b", int(operation)))
		}

		decoded = append(decoded, []byte(instruction)...)
	}

	contents := printHead(filename) + string(decoded)

	fmt.Print(contents)
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
