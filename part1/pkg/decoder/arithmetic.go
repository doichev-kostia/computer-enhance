package decoder

import (
	"fmt"
	"strings"
)

// Common pattern
//
// |  op  | pattern |
// ------------------
// | ADD  | 000     |
// | SUB  | 101     |
// | CMP  | 111     |

// [000000|d|w] [mod|reg|r/m] [disp-lo] [disp-hi]
func addRegOrMemToReg(operation byte, d *Decoder) (string, error) {
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
		return "", fmt.Errorf("expected to get an operand for the 'ADD: Reg/memory with register to either' instruction")
	}

	// mod is the 2 high bits
	mod := operand >> 6
	reg := (operand >> 3) & 0b00000111
	rm := operand & 0b00000111

	dest, src, err := d.decodeRegOrMem("ADD: Reg/memory with register to either", mod, reg, rm, isWord, dir)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("add %s, %s\n", dest, src), nil
}

// [100000|s|w] [mod|000|r/m] [disp-lo] [disp-hi] [data] [data if s|w = 0|1]
func addImmediateToRegOrMem(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	sign := (operation >> 1) & 0b00000001
	verifySign(sign)
	isSigned := sign == SignExtension

	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the 'ADD: immediate to register/memory' instruction")
	}

	mod := operand >> 6
	reg := (operand >> 3) & 0b00000111
	rm := operand & 0b00000111

	// must be 000 according to the "Instruction reference"
	if reg != 0 {
		return "", fmt.Errorf("expected the reg field to be 000 for the 'ADD: immediate to register/memory' instruction")
	}

	dest, err := d.decodeImmediateToRegOrMem("ADD: immediate to register/memory", mod, rm, isWord)
	if err != nil {
		return "", err
	}

	// the 8086 uses optimization technique - instead of using two bytes to represent a 16-bit immediate value, it can use one byte and sign-extend it, saving a byte in the instruction encoding when the immediate value is small enough to fit in a signed byte.
	immediateValue, err := d.decodeImmediate("ADD: immediate to register/memory", isWord && !isSigned)
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
	fmt.Fprintf(&builder, "add %s, ", dest)

	// we need to specify the size of the value
	if mod != RegisterModeFieldEncoding {
		// add [bp + 75], byte 12
		// add [bp + 75], word 512
		builder.WriteString(size + " ")
	}

	if isSigned {
		truncated := uint8(immediateValue)
		fmt.Fprintf(&builder, "%d", int8(truncated))
	} else {
		fmt.Fprintf(&builder, "%d", immediateValue)
	}

	builder.WriteString("\n")
	return builder.String(), nil
}

// [0000010|w] [data] [data if w = 1]
func addImmediateToAccumulator(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	immediateValue, err := d.decodeImmediate("ADD: immediate to accumulator", isWord)
	if err != nil {
		return "", err
	}

	regName := ""
	if isWord {
		regName = "ax"
	} else {
		regName = "al"
	}

	return fmt.Sprintf("add %s, %d\n", regName, immediateValue), nil
}

// [001010|d|w] [mod|reg|r/m] [disp-lo] [disp-hi]
func subRegOrMemFromReg(operation byte, d *Decoder) (string, error) {
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
		return "", fmt.Errorf("expected to get an operand for the 'SUB: Reg/memory and register to either' instruction")
	}

	// mod is the 2 high bits
	mod := operand >> 6
	reg := (operand >> 3) & 0b00000111
	rm := operand & 0b00000111

	dest, src, err := d.decodeRegOrMem("SUB: Reg/memory and register to either", mod, reg, rm, isWord, dir)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("sub %s, %s\n", dest, src), nil
}

// [100000|s|w] [mod|101|r/m] [disp-lo] [disp-hi] [data] [data if s|w = 0|1]
func subImmediateFromRegOrMem(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	sign := (operation >> 1) & 0b00000001
	verifySign(sign)
	isSigned := sign == SignExtension

	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the 'SUB: immediate to register/memory' instruction")
	}

	mod := operand >> 6
	reg := (operand >> 3) & 0b00000111
	rm := operand & 0b00000111

	// must be 101 according to the "Instruction reference"
	if reg != 0b101 {
		return "", fmt.Errorf("expected the reg field to be 101 for the 'SUB: immediate to register/memory' instruction")
	}

	dest, err := d.decodeImmediateToRegOrMem("SUB: immediate to register/memory", mod, rm, isWord)
	if err != nil {
		return "", err
	}

	// the 8086 uses optimization technique - instead of using two bytes to represent a 16-bit immediate value, it can use one byte and sign-extend it, saving a byte in the instruction encoding when the immediate value is small enough to fit in a signed byte.
	immediateValue, err := d.decodeImmediate("SUB: immediate to register/memory", isWord && !isSigned)
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
	fmt.Fprintf(&builder, "sub %s, ", dest)

	// we need to specify the size of the value
	if mod != RegisterModeFieldEncoding {
		// sub [bp + 75], byte 12
		// sub [bp + 75], word 512
		builder.WriteString(size + " ")
	}

	if isSigned {
		truncated := uint8(immediateValue)
		fmt.Fprintf(&builder, "%d", int8(truncated))
	} else {
		fmt.Fprintf(&builder, "%d", immediateValue)
	}

	builder.WriteString("\n")
	return builder.String(), nil
}

// [0010110|w] [data] [data if w = 1]
func subImmediateFromAccumulator(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	immediateValue, err := d.decodeImmediate("SUB: immediate to accumulator", isWord)
	if err != nil {
		return "", err
	}

	regName := ""
	if isWord {
		regName = "ax"
	} else {
		regName = "al"
	}

	return fmt.Sprintf("sub %s, %d\n", regName, immediateValue), nil
}

// [001110|d|w] [mod|reg|r/m] [disp-lo] [disp-hi]
func cmpRegOrMemWithReg(operation byte, d *Decoder) (string, error) {
	return "", nil
}

// [100000|s|w] [mod|111|r/m] [disp-lo] [disp-hi] [data] [data if s|w = 0|1]
func cmpImmediateWithRegOrMem(operation byte, d *Decoder) (string, error) {
	return "", nil
}

// [0011110|w] [data] [data if w = 1]
func cmpImmediateWithAccumulator(operation byte, d *Decoder) (string, error) {
	return "", nil
}
