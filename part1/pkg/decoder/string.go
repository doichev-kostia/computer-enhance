package decoder

// [1111001|z]
// __REPE__ and __REPZ__ are, by convention, used with the __CMPS__ (Compare string) and __SCAS__ (Scan string) instructions
// and require __ZF__ to be set before initializing the next repetition.
//
// __REPNE__ and __REPNZ__ are, by convention, used with the __CMPS__ (Compare string) and __SCAS__ (Scan string) instructions
// and require  __ZF__ flag to be cleared or the repetition is terminated.
func repeatPrefix(operation byte, d *Decoder) string {
	loop := operation & 0b00000001
	verifyLoop(loop)

	next, ok := d.peekNext()
	if ok == false {
		return "rep"
	}

	mnemonic := ""
	switch next >> 1 { // discard the "W" flag
	case 0b1010010:
		mnemonic = "movs"
	case 0b1010011:
		mnemonic = "cmps"
	case 0b1010111:
		mnemonic = "scas"
	case 0b1010110:
		mnemonic = "lods"
	case 0b1010101:
		mnemonic = "stos"
	default:
		return "rep"
	}

	// As this is a convention to use
	if mnemonic == "cmps" || mnemonic == "scas" {
		if loop == LoopWhileZero {
			return "repz"
		} else {
			return "repnz"
		}
	} else {
		return "rep"
	}
}

// [1010010|w]
func movs(operation byte, d *Decoder) (string, error) {
	operationType := operation & 0b00000001
	verifyOperationType(operationType)

	if operationType == WordOperation {
		return "movsw\n", nil
	} else {
		return "movsb\n", nil
	}
}

// [1010011|w]
func cmps(operation byte, d *Decoder) (string, error) {
	operationType := operation & 0b00000001
	verifyOperationType(operationType)

	if operationType == WordOperation {
		return "cmpsw\n", nil
	} else {
		return "cmpsb\n", nil
	}
}

// [1010111|w]
func scas(operation byte, d *Decoder) (string, error) {
	operationType := operation & 0b00000001
	verifyOperationType(operationType)

	if operationType == WordOperation {
		return "scasw\n", nil
	} else {
		return "scasb\n", nil
	}
}

// [1010110|w]
func lods(operation byte, d *Decoder) (string, error) {
	operationType := operation & 0b00000001
	verifyOperationType(operationType)

	if operationType == WordOperation {
		return "lodsw\n", nil
	} else {
		return "lodsb\n", nil
	}
}

// [1010101|w]
func stos(operation byte, d *Decoder) (string, error) {
	operationType := operation & 0b00000001
	verifyOperationType(operationType)

	if operationType == WordOperation {
		return "stosw\n", nil
	} else {
		return "stosb\n", nil
	}
}
