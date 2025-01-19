
## assembling
`ls ./assets/ | grep .asm | xargs -P 4 -I {} ./assemble.fish ./assets/{}`
or
`./assemble.fish ./assets/<filename>.asm`

## running
`go run ./main.go ./listingxxx`

8086 manual
https://edge.edx.org/c4x/BITSPilani/EEE231/asset/8086_family_Users_Manual_1_.pdf

Page 160 (pdf) - instruction encoding

opcode – first 6 bits of the instruction. Instruction type
D field – 7th bit of the instruction. "Direction" of the operation. 0 means register to register, 1 means register to
memory or memory to register.
W field – 8th bit of the instruction. "Word" or "byte" operation. 0 means byte, 1 means word.

The second byte of the instruction usually identifies the instruction's operands.

MOD – 2 bits. "Mode" field indicates whether one of the operands is in memory or whether both operands are in registers.
REG - 3 bits. "Register" field identifies a register that is one of the instruction operands
R/M - 3 bits. "Register/Memory" field. If the MOD field indicates that the operand is in memory, the R/M field
identifies the memory location. If the MOD field indicates that both operands are in registers, the R/M field identifies the second register.
