package decoder

import (
	"encoding/binary"
	"fmt"
)

// [11101000] [ip-inc-lo] [ip-inc-hi]
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

	// we start position counting from 1, instead of 0
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
