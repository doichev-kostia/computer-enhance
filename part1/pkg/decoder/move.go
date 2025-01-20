package decoder

import (
	"encoding/binary"
	"fmt"
	"strconv"
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
