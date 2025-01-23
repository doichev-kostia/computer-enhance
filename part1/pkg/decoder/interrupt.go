package decoder

import "fmt"

// [11001101] [data]
func interruptWithType(operation byte, d *Decoder) (string, error) {
	data, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get a type for the 'INT: type specified' instruction")
	}

	return fmt.Sprintf("int %d\n", data), nil
}

// [11001100]
func interruptType3(operation byte, d *Decoder) (string, error) {
	return "int3\n", nil
}

// [11001110]
func interruptOnOverflow(operation byte, d *Decoder) (string, error) {
	return "into\n", nil
}

// [11001111]
func interruptReturn(operation byte, d *Decoder) (string, error) {
	return "iret\n", nil
}
