package decoder

import (
	"fmt"
	"strings"
)

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

	dest, err := d.decodeImmediateToRegOrMem("immediate to register/memory", mod, reg, rm, isWord)
	if err != nil {
		return "", err
	}

	immediateValue, err := d.decodeImmediate("immediate to register/memory", isWord)
	if err != nil {
		return "", err
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

	immediateValue, err := d.decodeImmediate("MOV: immediate to register", isWord)
	if err != nil {
		return "", err
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

	dest, src, err := d.decodeRegOrMem("Register/memory to/from register", mod, reg, rm, isWord, dir)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("mov %s, %s\n", dest, src), nil
}

// [1010000w] [addr-lo] [addr-hi]
func moveMemoryToAccumulator(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	regName := ""
	if isWord {
		regName = "ax"
	} else {
		regName = "al"
	}

	address, err := d.decodeAddress("MOV: memory to accumulator", isWord)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("mov %s, [%d]\n", regName, address), nil
}

// [1010001w] [addr-lo] [addr-hi]
func moveAccumulatorToMemory(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	address, err := d.decodeAddress("MOV: accumulator to address", isWord)
	if err != nil {
		return "", err
	}

	regName := ""
	if isWord {
		regName = "ax"
	} else {
		regName = "al"
	}

	return fmt.Sprintf("mov [%d], %s\n", address, regName), nil
}
