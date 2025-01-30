package main

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
	register: Register,

	// as we can use high or low segments of some registers
	offset:   u8,
	capacity: u8,
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
