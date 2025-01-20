package decoder

import "fmt"

// [000000|d|w] [mod|reg|r/m] [disp-lo] [disp-hi]
func addRegOrMemWithReg(operation byte, d *Decoder) (string, error) {
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
		return "", fmt.Errorf("expected to get an operand for the 'Reg/memory with register to either' instruction")
	}

	// mod is the 2 high bits
	mod := operand >> 6
	reg := (operand >> 3) & 0b00000111
	rm := operand & 0b00000111

	dest, src, err := d.decodeRegOrMem("Reg/memory with register to either", mod, reg, rm, isWord, dir)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("add %s, %s\n", dest, src), nil
}

// [100000|s|w] [mod|000|r/m]
func addImmediateToRegOrMem(operation byte, d *Decoder) (string, error) {
	return "", nil
}

// [0000010|w] [data-lo] [data-hi]
func addImmediateToAccumulator(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := (operation >> 3) & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	immediateValue, err := d.decodeImmediate("ADD: immediate to accumulator", isWord)
	if err != nil {
		return "", err
	}
	signedValue := int16(immediateValue)

	regName := ""
	if isWord {
		regName = "ax"
	} else {
		regName = "al"
	}

	if signedValue < 0 {
		return fmt.Sprintf("add %s, %d ; or %d\n", regName, immediateValue, signedValue), nil
	} else {
		return fmt.Sprintf("add %s, %d\n", regName, immediateValue), nil
	}
}
