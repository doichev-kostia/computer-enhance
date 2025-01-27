package decoder

import (
	"encoding/binary"
	"fmt"
)

// [11101000] [ip-inc-lo] [ip-inc-hi]
// definitions.IP_INC_LO definitions.IP_INC_HI
// Example: call 11804
func callDirectWithinSegment(operation byte, d *Decoder) (string, error) {
	low, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a lower instruction pointer byte in 'CALL: Direct within segment'")
	}
	high, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a higher instruction pointer byte in 'CALL: Direct within segment'")
	}

	pointerIncrement := binary.LittleEndian.Uint16([]byte{low, high})
	pointer := uint32(pointerIncrement) + uint32(d.pos)
	return fmt.Sprintf("call %d\n", pointer), nil
}

// [11111111] [mod|010|r/m] [disp-lo?] [disp-hi?]
// Example: call ax or call [bp - 100] or call near [bp+si-0x3a]
func callIndirectWithinSegment(operation byte, d *Decoder) (string, error) {
	const isWord = true
	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand in 'CALL: Indirect within segment'")
	}
	mod, reg, rm := decodeOperand(operand)
	if reg != 0b010 {
		return "", fmt.Errorf("expected to get a register value of 010 in 'CALL: Indirect within segment'")
	}
	procedureAddress, err := d.decodeUnaryRegOrMem("CALL: Indirect within segment", mod, rm, isWord)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("call %s\n", procedureAddress), nil
}

// [10011010] [ip-lo] [ip-hi] [cs-lo] [cs-hi]
// Example: call 123:456; 10011010 (11001000 00000001 = 456 le) (01111011 00000000 = 123 le)
// definitions.IP_LO definitions.IP_HI
func callDirectIntersegment(operation byte, d *Decoder) (string, error) {
	ipLow, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a lower instruction pointer byte in 'CALL: Direct intersegment'")
	}
	ipHigh, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a higher instruction pointer byte in 'CALL: Direct intersegment'")
	}

	codeSegmentLow, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a lower code segment byte in 'CALL: Direct intersegment'")
	}

	codeSegmentHigh, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a higher code segment byte in 'CALL: Direct intersegment'")
	}

	instructionPointer := binary.LittleEndian.Uint16([]byte{ipLow, ipHigh})
	codeSegment := binary.LittleEndian.Uint16([]byte{codeSegmentLow, codeSegmentHigh})

	return fmt.Sprintf("call %d:%d\n", codeSegment, instructionPointer), nil
}

// [11111111] [mod|011|r/m] [disp-lo?] [disp-hi?]
// Example: call far [bp+si-0x3a]
func callIndirectIntersegment(operation byte, d *Decoder) (string, error) {
	const isWord = true
	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand in 'CALL: Indirect intersegment'")
	}
	mod, reg, rm := decodeOperand(operand)
	if reg != 0b011 {
		return "", fmt.Errorf("expected to get a register value of 011 in 'CALL: Indirect intersegment'")
	}
	procedureAddress, err := d.decodeUnaryRegOrMem("CALL: Indirect intersegment", mod, rm, isWord)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("call far %s\n", procedureAddress), nil
}

// [11101001] [ip-inc-lo] [ip-inc-hi]
// definitions.IP_INC_LO definitions.IP_INC_HI
func jumpDirectWithinSegment(operation byte, d *Decoder) (string, error) {
	low, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a lower instruction pointer byte in 'JMP: Direct within segment'")
	}
	high, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a higher instruction pointer byte in 'JMP: Direct within segment'")
	}

	pointerIncrement := binary.LittleEndian.Uint16([]byte{low, high})
	pointer := uint32(pointerIncrement) + uint32(d.pos)
	return fmt.Sprintf("jmp %d\n", pointer), nil
}

// [11101011] [inc-inc8]
// definitions.IP_INC8
// Example: jmp test_label where label is within 127 bytes
func jumpDirectWithinSegmentShort(operation byte, d *Decoder) (string, error) {
	pointerIncrement, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an 8-bit instruction pointer increment in 'JMP: Direct within segment-short'")
	}

	offset := int8(pointerIncrement)
	address := d.pos + int(offset)
	labelName := createLabelName(address)
	d.labels[address] = labelName
	return fmt.Sprintf("jmp %s\n", labelName), nil
}

// [11111111] [mod|100|r/m] [disp-lo?] [disp-hi?]
func jumpIndirectWithinSegment(operation byte, d *Decoder) (string, error) {
	const isWord = true
	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand in 'JMP: Indirect within segment'")
	}
	mod, reg, rm := decodeOperand(operand)
	if reg != 0b100 {
		return "", fmt.Errorf("expected to get a register value of 100 in 'JMP: Indirect within segment'")
	}
	address, err := d.decodeUnaryRegOrMem("JMP: Indirect within segment", mod, rm, isWord)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("jmp %s\n", address), nil
}

// [11101010] [ip-lo] [ip-hi] [cs-lo] [cs-hi]
// definitions.IP_LO definitions.IP_HI
func jumpDirectIntersegment(operation byte, d *Decoder) (string, error) {
	ipLow, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a lower instruction pointer byte in 'JMP: Direct intersegment'")
	}
	ipHigh, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a higher instruction pointer byte in 'JMP: Direct intersegment'")
	}

	codeSegmentLow, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a lower code segment byte in 'JMP: Direct intersegment'")
	}

	codeSegmentHigh, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a higher code segment byte in 'JMP: Direct intersegment'")
	}

	instructionPointer := binary.LittleEndian.Uint16([]byte{ipLow, ipHigh})
	codeSegment := binary.LittleEndian.Uint16([]byte{codeSegmentLow, codeSegmentHigh})

	return fmt.Sprintf("jmp %d:%d\n", codeSegment, instructionPointer), nil
}

// [11111111] [mod|101|r/m] [disp-lo?] [disp-hi?]
func jumpIndirectIntersegment(operation byte, d *Decoder) (string, error) {
	const isWord = true
	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand in 'JMP: Indirect intersegment'")
	}
	mod, reg, rm := decodeOperand(operand)
	if reg != 0b101 {
		return "", fmt.Errorf("expected to get a register value of 101 in 'JMP: Indirect intersegment'")
	}
	address, err := d.decodeUnaryRegOrMem("JMP: Indirect intersegment", mod, rm, isWord)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("jmp far %s\n", address), nil
}

func jumpConditionally(operation byte, d *Decoder) (string, error) {
	name := JumpNames[operation]
	comment := ""
	altName, ok := JumpAlternativeNames[operation]
	if ok {
		comment = fmt.Sprintf("; %s", altName)
	}

	instructionPointer, ok := d.next()

	if ok == false {
		return "", fmt.Errorf("expected to get a jump instruction pointer for the '%s' instruction", name)
	}

	offset := int8(instructionPointer) // signed value

	labelLocation := d.pos + int(offset)
	labelName := createLabelName(labelLocation)
	d.labels[labelLocation] = labelName

	if comment == "" {
		return fmt.Sprintf("%s %s\n", name, labelName), nil
	} else {
		return fmt.Sprintf("%s %s %s\n", name, labelName, comment), nil
	}
}

func createLabelName(pos int) string {
	return fmt.Sprintf("label__%d", pos)
}
