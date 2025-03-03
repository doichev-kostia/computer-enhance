package decoder

import (
	"fmt"
	"strings"
)

// Common pattern
//
// |  op  | pattern |
// ------------------
// 100000|s|w
// | ADD  | 000     |
// | ADC  | 010     |
// | SBB  | 011     |
// | SUB  | 101     |
// | CMP  | 111     |
// ----------
// 1111011|w
// | NEG |  011    |
// | MUL  | 100     |
// | IMUL | 101     |
// | DIV  | 110     |
// | IDIV | 111     |

// [00110111]
func aaa(operation byte, d *Decoder) (string, error) {
	return "aaa\n", nil
}

// [00100111]
func daa(operation byte, d *Decoder) (string, error) {
	return "daa\n", nil
}

// [00111111]
func aas(operation byte, d *Decoder) (string, error) {
	return "aas\n", nil
}

// [00101111]
func das(operation byte, d *Decoder) (string, error) {
	return "das\n", nil
}

// [11010100] [00001010] [disp-lo?] [disp-hi?]
// definitions.INSTRUCTION_REFERENCE
func aam(operation byte, d *Decoder) (string, error) {
	// Note(Kostia)
	// I don't know why the INSTRUCTION_REFERENCE says there should be displacement, but there are no fields to figure that out.
	// Moreover, in the Table 4-13. Machine Instruction Decoding Guide the element 11010100 00001010 has no displacement either
	next, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the 'AAM' instruction")
	}
	if next != 0b00001010 {
		return "", fmt.Errorf("expected the operand to be 00001010 for the 'AAM' instruction")
	}
	return "aam\n", nil
}

// [11010101] [00001010] [disp-lo?] [disp-hi?]
// definitions.INSTRUCTION_REFERENCE
func aad(operation byte, d *Decoder) (string, error) {
	// Note(Kostia)
	// I don't know why the "Instruction reference" says there should be displacement, but there are no fields to figure that out.
	// Moreover, in the Table 4-13. Machine Instruction Decoding Guide the element 11010101 00001010 has no displacement either
	next, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the 'AAD' instruction")
	}
	if next != 0b00001010 {
		return "", fmt.Errorf("expected the operand to be 00001010 for the 'AAD' instruction")
	}
	return "aad\n", nil
}

// [10011000]
func cbw(operation byte, d *Decoder) (string, error) {
	return "cbw\n", nil
}

// [10011001]
func cwd(operation byte, d *Decoder) (string, error) {
	return "cwd\n", nil
}

// [000000|d|w] [mod|reg|r/m] [disp-lo?] [disp-hi?]
func addRegOrMemToReg(operation byte, d *Decoder) (string, error) {
	dest, src, err := d.regOrMemWithReg("ADD: Reg/memory with register to either", operation)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("add %s, %s\n", dest, src), nil
}

// [100000|s|w] [mod|000|r/m] [disp-lo?] [disp-hi?] [data] [data if s|w = 0|1]
func addImmediateToRegOrMem(operation byte, d *Decoder) (string, error) {
	instruction, err := buildImmediateWithRegOrMemArithmeticInstruction("add", 0b000, "ADD: immediate to register/memory", operation, d)
	if err != nil {
		return "", err
	}
	return instruction, nil
}

// [0000010|w] [data] [data if w = 1]
func addImmediateToAccumulator(operation byte, d *Decoder) (string, error) {
	regName, immediateValue, err := d.immediateWithAccumulator("ADD: immediate to accumulator", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("add %s, %d\n", regName, immediateValue), nil
}

// [000100|d|w] [mod|reg|r/m] [disp-lo?] [disp-hi?]
func adcRegOrMemToReg(operation byte, d *Decoder) (string, error) {
	dest, src, err := d.regOrMemWithReg("ADC: Reg/memory with register to either", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("adc %s, %s\n", dest, src), nil
}

// [100000|s|w] [mod|010|r/m] [disp-lo?] [disp-hi?] [data] [data if s|w = 0|1]
func adcImmediateToRegOrMem(operation byte, d *Decoder) (string, error) {
	instruction, err := buildImmediateWithRegOrMemArithmeticInstruction("adc", 0b010, "ADC: immediate to register/memory", operation, d)
	if err != nil {
		return "", err
	}
	return instruction, nil
}

// [0001010|w] [data] [data if w = 1]
func adcImmediateToAccumulator(operation byte, d *Decoder) (string, error) {
	regName, immediateValue, err := d.immediateWithAccumulator("ADC: immediate to accumulator", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("adc %s, %d\n", regName, immediateValue), nil
}

// [1111111|w] [mod|000|r/m] [disp-lo?] [disp-hi?]
func incRegOrMem(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the 'INC: register/memory' instruction")
	}

	mod := operand >> 6
	reg := (operand >> 3) & 0b00000111
	rm := operand & 0b00000111

	// must be 000 according to the "Instruction reference"
	if reg != 0b000 {
		return "", fmt.Errorf("expected the reg field to be 000 for the 'INC: register/memory' instruction")
	}

	dest, err := d.decodeUnaryRegOrMem("INC: register/memory", mod, rm, isWord)
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
		return fmt.Sprintf("inc %s %s\n", size, dest), nil
	} else {
		return fmt.Sprintf("inc %s\n", dest), nil
	}
}

// [01000|reg]
// Word operation
func incReg(operation byte, d *Decoder) (string, error) {
	reg := operation & 0b00000111
	regName := WordOperationRegisterFieldEncoding[reg]

	return fmt.Sprintf("inc %s\n", regName), nil
}

// [001010|d|w] [mod|reg|r/m] [disp-lo?] [disp-hi?]
func subRegOrMemFromReg(operation byte, d *Decoder) (string, error) {
	dest, src, err := d.regOrMemWithReg("SUB: Reg/memory and register to either", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("sub %s, %s\n", dest, src), nil
}

// [100000|s|w] [mod|101|r/m] [disp-lo?] [disp-hi?] [data] [data if s|w = 0|1]
func subImmediateFromRegOrMem(operation byte, d *Decoder) (string, error) {
	instruction, err := buildImmediateWithRegOrMemArithmeticInstruction("sub", 0b101, "SUB: immediate from register/memory", operation, d)
	if err != nil {
		return "", err
	}
	return instruction, nil
}

// [0010110|w] [data] [data if w = 1]
func subImmediateFromAccumulator(operation byte, d *Decoder) (string, error) {
	regName, immediateValue, err := d.immediateWithAccumulator("SUB: immediate from accumulator", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("sub %s, %d\n", regName, immediateValue), nil
}

// [000110|d|w] [mod|reg|r/m] [disp-lo?] [disp-hi?]
func sbbRegOrMemFromReg(operation byte, d *Decoder) (string, error) {
	dest, src, err := d.regOrMemWithReg("SBB: Reg/memory and register to either", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("sbb %s, %s\n", dest, src), nil

}

// [100000|s|w] [mod|011|r/m] [disp-lo?] [disp-hi?] [data] [data if s|w = 0|1]
func sbbImmediateFromRegOrMem(operation byte, d *Decoder) (string, error) {
	instruction, err := buildImmediateWithRegOrMemArithmeticInstruction("sbb", 0b011, "SBB: immediate from register/memory", operation, d)
	if err != nil {
		return "", err
	}
	return instruction, nil
}

// [0010110|w] [data] [data if w = 1]
func sbbImmediateFromAccumulator(operation byte, d *Decoder) (string, error) {
	regName, immediateValue, err := d.immediateWithAccumulator("SBB: immediate from accumulator", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("sbb %s, %d\n", regName, immediateValue), nil
}

// [1111111|w] [mod|001|r/m] [disp-lo?] [disp-hi?]
func decRegOrMem(operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the 'DEC: register/memory' instruction")
	}

	mod := operand >> 6
	reg := (operand >> 3) & 0b00000111
	rm := operand & 0b00000111

	// must be 001 according to the "Instruction reference"
	if reg != 0b001 {
		return "", fmt.Errorf("expected the reg field to be 001 for the 'DEC: register/memory' instruction")
	}

	dest, err := d.decodeUnaryRegOrMem("DEC: register/memory", mod, rm, isWord)
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
		return fmt.Sprintf("dec %s %s\n", size, dest), nil
	} else {
		return fmt.Sprintf("dec %s\n", dest), nil
	}
}

// [01001|reg]
// Word operation
func decReg(operation byte, d *Decoder) (string, error) {
	reg := operation & 0b00000111
	regName := WordOperationRegisterFieldEncoding[reg]

	return fmt.Sprintf("dec %s\n", regName), nil
}

// [001110|d|w] [mod|reg|r/m] [disp-lo?] [disp-hi?]
func cmpRegOrMemWithReg(operation byte, d *Decoder) (string, error) {
	dest, src, err := d.regOrMemWithReg("CMP: Reg/memory and register", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("cmp %s, %s\n", dest, src), nil
}

// [100000|s|w] [mod|111|r/m] [disp-lo?] [disp-hi?] [data] [data if s|w = 0|1]
func cmpImmediateWithRegOrMem(operation byte, d *Decoder) (string, error) {
	instruction, err := buildImmediateWithRegOrMemArithmeticInstruction("cmp", 0b111, "CMP: immediate with register/memory", operation, d)
	if err != nil {
		return "", err
	}
	return instruction, nil
}

// [0011110|w] [data] [data if w = 1]
func cmpImmediateWithAccumulator(operation byte, d *Decoder) (string, error) {
	regName, immediateValue, err := d.immediateWithAccumulator("CMP: immediate with accumulator", operation)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("cmp %s, %d\n", regName, immediateValue), nil
}

// [1111011|w] [mod|011|r/m] [disp-lo?] [disp-hi?]
func neg(operation byte, d *Decoder) (string, error) {
	return mulOrDiv("neg", 0b011, "NEG: Change sign", operation, d)
}

// [1111011|w] [mod|100|r/m] [disp-lo?] [disp-hi?]
func mul(operation byte, d *Decoder) (string, error) {
	return mulOrDiv("mul", 0b100, "MUL: Unsigned multiplication", operation, d)
}

// [1111011|w] [mod|101|r/m] [disp-lo?] [disp-hi?]
func imul(operation byte, d *Decoder) (string, error) {
	return mulOrDiv("imul", 0b101, "IMUL: Signed multiplication", operation, d)
}

// [1111011|w] [mod|110|r/m] [disp-lo?] [disp-hi?]
func div(operation byte, d *Decoder) (string, error) {
	return mulOrDiv("div", 0b110, "DIV: Unsigned division", operation, d)
}

// [1111011|w] [mod|111|r/m] [disp-lo?] [disp-hi?]
func idiv(operation byte, d *Decoder) (string, error) {
	return mulOrDiv("idiv", 0b111, "IDIV: Signed division", operation, d)
}

// [100000|s|w] [mod|<regPattern>|r/m] [disp-lo?] [disp-hi?] [data] [data if s|w = 0|1]
func buildImmediateWithRegOrMemArithmeticInstruction(mnemonic string, regPattern byte, instructionName string, operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	sign := (operation >> 1) & 0b00000001
	verifySign(sign)
	isSigned := sign == SignExtension

	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the '%s' instruction", instructionName)
	}

	mod, reg, rm := decodeOperand(operand)

	if reg != regPattern {
		return "", fmt.Errorf("expected the reg field to be %.3b for the '%s' instruction", regPattern, instructionName)
	}

	dest, err := d.decodeUnaryRegOrMem(instructionName, mod, rm, isWord)
	if err != nil {
		return "", err
	}

	// the 8086 uses optimization technique - instead of using two bytes to represent a 16-bit immediate value, it can use one byte and sign-extend it, saving a byte in the instruction encoding when the immediate value is small enough to fit in a signed byte.
	immediateValue, err := d.decodeImmediate(instructionName, isWord && !isSigned)
	if err != nil {
		return "", err
	}

	size := ""
	if isWord {
		size = "word"
	} else {
		size = "byte"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "%s %s, ", mnemonic, dest)

	// we need to specify the size of the value
	if mod != RegisterModeFieldEncoding {
		// add [bp + 75], byte 12
		// sub [bp + 75], word 512
		builder.WriteString(size + " ")
	}

	if isSigned {
		truncated := uint8(immediateValue)
		fmt.Fprintf(&builder, "%d", int8(truncated))
	} else {
		fmt.Fprintf(&builder, "%d", immediateValue)
	}

	builder.WriteString("\n")

	return builder.String(), nil
}

// [1111011|w] [mod|<regPattern>|r/m] [disp-lo?] [disp-hi?]
// definitions.INSTRUCTION_REFERENCE refers to mul, imul, aam as "Multiplication" and div, idiv, aad, cbw, cwd as "Division"
func mulOrDiv(mnemonic string, regPattern byte, instructionName string, operation byte, d *Decoder) (string, error) {
	// the & 0b00 is to discard all the other bits and leave the ones we care about
	operationType := operation & 0b00000001
	verifyOperationType(operationType)
	isWord := operationType == WordOperation

	operand, ok := d.next()
	if ok == false {
		return "", fmt.Errorf("expected to get an operand for the '%s' instruction", instructionName)
	}

	mod := operand >> 6
	reg := (operand >> 3) & 0b00000111
	rm := operand & 0b00000111

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

	if mod != RegisterModeFieldEncoding {
		return fmt.Sprintf("%s %s %s\n", mnemonic, size, dest), nil
	} else {
		return fmt.Sprintf("%s %s\n", mnemonic, dest), nil
	}
}
