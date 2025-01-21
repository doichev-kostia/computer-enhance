package decoder

import "fmt"

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
