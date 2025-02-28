package main

import "core:fmt"
import "core:strings"

Prefix :: enum {
	None,
	LOCK,
	REP,
	Count,
}

Mnemonic :: enum {
	None,
	MOV,
	PUSH,
	POP,
	XCHG,
	IN,
	OUT,
	XLAT,
	LEA,
	LDS,
	LES,
	LAHF,
	SAHF,
	PUSHF,
	POPF,
	ADD,
	ADC,
	INC,
	AAA,
	DAA,
	SUB,
	SBB,
	DEC,
	NEG,
	CMP,
	MUL,
	IMUL,
	AAM,
	DIV,
	IDIV,
	AAD,
	CBW,
	CWD,
	NOT,
	SHL,
	SHR,
	SAR,
	ROL,
	ROR,
	RCL,
	RCR,
	AND,
	TEST,
	OR,
	XOR,
	MOVS,
	CMPS,
	SCAS,
	LODS,
	STOS,
	CALL,
	JMP,
	REU,
	JE,
	JK,
	JLE,
	JB,
	JBE,
	JP,
	JO,
	JS,
	JNZ,
	JGE,
	JG,
	JAE,
	JA,
	JNP,
	JNO,
	JNS,
	JCXZ,
	LOOP,
	LOOPE,
	LOOPNE,
	INT,
	INT3,
	INTO,
	IRET,
	CLC,
	CMC,
	STC,
	CLD,
	STD,
	CLI,
	STI,
	HTL,
	WAIT,
	Count,
}

Mod :: enum {
	None = -1,
	Memory_No_Displacement = 0b00,
	Memory_8Displacement = 0b01,
	Memory_16Displacement = 0b10,
	Register = 0b11,
	Count,
}

Effective_Address_Equation :: enum {
	Direct, // if MOD = 00
	BX_SI,
	BX_DI,
	BP_SI,
	BP_DI,
	SI,
	DI,
	BP,
	BX,
	Count,
}

Effective_Address :: struct {
	segment:      Register,
	equation:     Effective_Address_Equation,
	displacement: i16,
}

get_effective_address :: proc(
	eff_addr: Effective_Address,
	allocator := context.allocator,
) -> string {

	index := [Effective_Address_Equation.Count]string {
		"",
		"bx + si",
		"bx + di",
		"bp + si",
		"bp + di",
		"si",
		"di",
		"bp",
		"bx",
	}

	eq := index[eff_addr.equation]
	builder := strings.builder_make(allocator)
	writer := strings.to_writer(&builder)

	if eff_addr.segment != Register.DS {
		fmt.wprintf(writer, "%s:", "")
	}

	if eq == "" {
		fmt.wprintf(writer, "[%d]", u16(eff_addr.displacement))
	} else if eff_addr.displacement < 0 {
		fmt.wprintf(writer, "[%s - %d]", eq, eff_addr.displacement)
	} else {
		fmt.wprintf(writer, "[%s + %d]", eq, eff_addr.displacement)
	}

	return strings.to_string(builder)
}

// REG (Register) field encoding
// | REG | W = 0 | W = 1|
// ---------------------
// | 000 | AL    | AX   |
// | 001 | CL    | CX   |
// | 010 | DL    | DX   |
// | 011 | BL    | BX   |
// | 100 | AH    | SP   |
// | 101 | CH    | BP   |
// | 110 | DH    | SI   |
// | 111 | BH    | DI   |
Register :: enum {
	None,
	A, // Accumulator
	B, // Base
	C, // Count
	D, // Data
	SP, // stack pointer
	BP, // base pointer
	SI, // source index
	DI, // destination index
	ES, // extra segment
	CS, // code segment
	SS, // stack segment
	DS, // data segment
	IP, // instruction pointer
	FLAGS, // bit flags
	Count,
}

Flag :: enum {
	TF, // trap
	DF, // direction
	IF, // interrup-enable
	OF, // overflow
	SF, // sign
	ZF, // zero
	AF, // auxiliary carry
	PF, // parity
	CF, // carry
}

Bit_Field :: enum {
	S,
	W,
	D,
	V,
	Z,
}

Register_Access :: struct {
	name:     Register,

	// as we can use high or low segments of some registers
	offset:   u8,
	capacity: u8,
}

get_register :: proc(reg: Register_Access) -> string {
	assert(reg.capacity <= 2, "8086 has only 16 bit registers")
	// TODO(Kostia) is this going to be re-created every time?
	index := [Register.Count][3]string {
		{"", "", ""},
		{"al", "ah", "ax"},
		{"bl", "bh", "bx"},
		{"cl", "ch", "cx"},
		{"dl", "dh", "dx"},
		{"sp", "sp", "sp"},
		{"bp", "bp", "bp"},
		{"si", "si", "si"},
		{"di", "di", "di"},
		{"es", "es", "es"},
		{"cs", "cs", "cs"},
		{"ss", "ss", "ss"},
		{"ds", "ds", "ds"},
		{"ip", "ip", "ip"},
		{"flags", "flags", "flags"},
	}

	tuple := index[reg.name]

	if reg.capacity == 2 {
		return tuple[2]
	} else {
		assert(reg.offset == 0 || reg.offset == 1, "Either low or high register is allowed")
		return tuple[reg.offset]
	}
}

Operand_Type :: enum {
	None,
	Register,
	Memory,
	Immediate, // usually referred to as "data" in the "Instruction Reference"
	Relative_Immediate, // IP-INC8, IP-INC-LO, IP-INC-HI in the "Instruction Reference"
}

Register_Operand :: struct {
	type:     Operand_Type,
	register: Register_Access,
}

Immediate_Operand :: struct {
	type:             Operand_Type,
	immediate:        u32,
	signed_immediate: i32,
}

Effective_Address_Operand :: struct {
	type:              Operand_Type,
	effective_address: Effective_Address,
}

Void_Operand :: struct {
	type: Operand_Type,
}

Instruction_Operand :: union #no_nil {
	Void_Operand,
	Register_Operand,
	Immediate_Operand,
	Effective_Address_Operand,
}

Instruction_Flags :: bit_set[Bit_Field]

Insturction_Prefixes :: bit_set[Prefix]

Instruction :: struct {
	opcode:   u8,
	mnemonic: Mnemonic,
	operands: [2]Instruction_Operand,
	flags:    Instruction_Flags,
	prefixes: Insturction_Prefixes,
	address:  u32,
	size:     u32,
}
