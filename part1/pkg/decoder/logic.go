package decoder

import "fmt"

// [1111011|w] [mod|010|r/m] [disp-lo?] [disp-hi?]
func not(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the 'NOT: Invert' instruction")
	}

	mod, reg, rm := parseOperand(operand)

	pattern := byte(0b010)
	if reg != pattern {
		return "", fmt.Errorf("expected the reg field to be %.3b for the 'NOT: Invert' instruction", pattern)
	}

	dest, err := d.decodeUnaryRegOrMem("NOT: Invert", mod, rm, isWord)
	if err != nil {
		return "", err
	}

	size := ""
	if isWord {
		size = "word"
	} else {
		size = "byte"
	}

	if mod != RegisterModeFieldEncoding {
		return fmt.Sprintf("not %s %s\n", size, dest), nil
	} else {
		return fmt.Sprintf("not %s\n", dest), nil
	}
}

// [110100|v|w] [mod|100|r/m] [disp-lo?] [disp-hi?]
func shl(operation byte, d *Decoder) (string, error) {
	return bitShift("shl", 0b100, "SHL/SAL: Shift logical/arithmetic left", operation, d)
}

// [110100|v|w] [mod|101|r/m] [disp-lo?] [disp-hi?]
func shr(operation byte, d *Decoder) (string, error) {
	return bitShift("shr", 0b101, "SHR: Shift logical right", operation, d)
}

// [110100|v|w] [mod|111|r/m] [disp-lo?] [disp-hi?]
func sar(operation byte, d *Decoder) (string, error) {
	return bitShift("sar", 0b111, "SAR: Shift arithmetic right", operation, d)
}

// [110100|v|w] [mod|000|r/m] [disp-lo?] [disp-hi?]
func rol(operation byte, d *Decoder) (string, error) {
	return bitShift("rol", 0b000, "ROL: Rotate left", operation, d)
}

// [110100|v|w] [mod|001|r/m] [disp-lo?] [disp-hi?]
func ror(operation byte, d *Decoder) (string, error) {
	return bitShift("ror", 0b001, "ROR: Rotate right", operation, d)
}

// [110100|v|w] [mod|010|r/m] [disp-lo?] [disp-hi?]
func rcl(operation byte, d *Decoder) (string, error) {
	return bitShift("rcl", 0b010, "RCL: Rotate through carry flag left", operation, d)
}

// [110100|v|w] [mod|011|r/m] [disp-lo?] [disp-hi?]
func rcr(operation byte, d *Decoder) (string, error) {
	return bitShift("rcr", 0b011, "RCR: Rotate through carry flag right", operation, d)
}

// [001000|d|w] [mod|reg|r/m] [disp-lo?] [disp-hi?]
func andRegOrMemWithReg(operation byte, d *Decoder) (string, error) {
	dest, src, err := d.regOrMemWithReg("AND: Reg/memory with register to either", operation)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("and %s, %s\n", dest, src), nil
}

// [1000000|w] [mod|100|r/m] [disp-lo?] [disp-hi?] [data] [data if w = 1]
func andImmediateWithRegOrMem(operation byte, d *Decoder) (string, error) {
	return d.buildImmediateWithRegOrMemInstruction("and", 0b100, "AND: Immediate with register/memory", operation)
}

// [0010010|w] [data] [data if w = 1]
func andImmediateWithAccumulator(operation byte, d *Decoder) (string, error) {
	regName, immediateValue, err := d.immediateWithAccumulator("AND: immediate with accumulator", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("and %s, %d\n", regName, immediateValue), nil
}

// [100001|d|w] [mod|reg|r/m] [disp-lo?] [disp-hi?]
func testRegOrMemWithReg(operation byte, d *Decoder) (string, error) {
	dest, src, err := d.regOrMemWithReg("TEST: Reg/memory with register to either", operation)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("test %s, %s\n", dest, src), nil
}

// [1111011|w] [mod|000|r/m] [disp-lo?] [disp-hi?] [data] [data if w = 1]
func testImmediateWithRegOrMem(operation byte, d *Decoder) (string, error) {
	return d.buildImmediateWithRegOrMemInstruction("test", 0b000, "TEST: Immediate with register/memory", operation)
}

// [1010100|w] [data] [data if w = 1]
func testImmediateWithAccumulator(operation byte, d *Decoder) (string, error) {
	regName, immediateValue, err := d.immediateWithAccumulator("TEST: immediate with accumulator", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("test %s, %d\n", regName, immediateValue), nil
}

// [000010|d|w] [mod|reg|r/m] [disp-lo?] [disp-hi?]
func orRegOrMemWithReg(operation byte, d *Decoder) (string, error) {
	dest, src, err := d.regOrMemWithReg("OR: Reg/memory with register to either", operation)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("or %s, %s\n", dest, src), nil
}

// [1000000|w] [mod|001|r/m] [disp-lo?] [disp-hi?] [data] [data if w = 1]
func orImmediateWithRegOrMem(operation byte, d *Decoder) (string, error) {
	return d.buildImmediateWithRegOrMemInstruction("or", 0b001, "OR: Immediate with register/memory", operation)
}

// [0000110|w] [data] [data if w = 1]
func orImmediateWithAccumulator(operation byte, d *Decoder) (string, error) {
	regName, immediateValue, err := d.immediateWithAccumulator("OR: immediate with accumulator", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("or %s, %d\n", regName, immediateValue), nil
}

// [001100|d|w] [mod|reg|r/m] [disp-lo?] [disp-hi?]
func xorRegOrMemWithReg(operation byte, d *Decoder) (string, error) {
	dest, src, err := d.regOrMemWithReg("XOR: Reg/memory with register to either", operation)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("xor %s, %s\n", dest, src), nil
}

// [1000000|w] [mod|110|r/m] [disp-lo?] [disp-hi?] [data] [data if w = 1]
func xorImmediateWithRegOrMem(operation byte, d *Decoder) (string, error) {
	return d.buildImmediateWithRegOrMemInstruction("xor", 0b110, "XOR: Immediate with register/memory", operation)
}

// [0011010|w] [data] [data if w = 1]
func xorImmediateWithAccumulator(operation byte, d *Decoder) (string, error) {
	regName, immediateValue, err := d.immediateWithAccumulator("XOR: immediate with accumulator", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("xor %s, %d\n", regName, immediateValue), nil
}

// [110100|v|w] [mod|<regPattern>|r/m] [disp-lo?] [disp-hi?]
func bitShift(mnemonic string, regPattern byte, instructionName string, operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	count := (operation >> 1) & 0b00000001
	verifyCount(count)

	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the '%s' instruction", instructionName)
	}

	mod, reg, rm := parseOperand(operand)

	if reg != regPattern {
		return "", fmt.Errorf("expected the reg field to be %.3b for the '%s' instruction", regPattern, instructionName)
	}

	dest, err := d.decodeUnaryRegOrMem(instructionName, mod, rm, isWord)
	if err != nil {
		return "", err
	}

	size := ""
	if isWord {
		size = "word"
	} else {
		size = "byte"
	}

	displayCount := ""
	if count == CountByCL {
		displayCount = "cl"
	} else {
		displayCount = "1"
	}

	if mod != RegisterModeFieldEncoding {
		return fmt.Sprintf("%s %s %s, %s\n", mnemonic, size, dest, displayCount), nil
	} else {
		return fmt.Sprintf("%s %s, %s\n", mnemonic, dest, displayCount), nil
	}
}
