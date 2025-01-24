package decoder

// [11111000]
func clc(operation byte, d *Decoder) (string, error) {
	return "clc\n", nil
}

// [11110101]
func cmc(operation byte, d *Decoder) (string, error) {
	return "cmc\n", nil
}

// [11111001]
func stc(operation byte, d *Decoder) (string, error) {
	return "stc\n", nil
}

// [11111100]
func cld(operation byte, d *Decoder) (string, error) {
	return "cld\n", nil
}

// [11111101]
func std(operation byte, d *Decoder) (string, error) {
	return "std\n", nil
}

// [11111010]
func cli(operation byte, d *Decoder) (string, error) {
	return "cli\n", nil
}

// [11111011]
func sti(operation byte, d *Decoder) (string, error) {
	return "sti\n", nil
}

// [11110100]
func hlt(operation byte, d *Decoder) (string, error) {
	return "hlt\n", nil
}

// [10011011]
func wait(operation byte, d *Decoder) (string, error) {
	return "wait\n", nil
}
