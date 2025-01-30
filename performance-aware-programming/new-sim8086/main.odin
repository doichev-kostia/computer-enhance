package main

import "core:fmt"

Memory :: struct {
	buff: [1024 * 1024]u8, // 1MB
}

MEMORY_MASK :: 0xFFFFF // 2^20 -> 1 048 576 bytes -> 1MB

// TODO: Build a table of mnemonics
// instruction struct


Context :: struct {
	segment: string, // string as there is no clear limitation to how many segment prefixes you can add
	labels:  map[u32]string, // pos:name
}

Decoder :: struct {
	memory: Memory,
	pos:    u32,
}

// TODO: somehow I need to request byte from the memory, as it has to do the wrapping
decoder_next :: proc(decoder: ^Decoder) -> (u8, bool) {
	if len(decoder.memory.buff) > decoder.pos {
		b := decoder.memory.buff[decoder.pos]
		decoder.pos += 1
		return b, true
	} else {
		return 0, false
	}
}

// offset 1 -> next char
decoder_peek_forward :: proc(decoder: ^Decoder, offset: u32) -> (u8, bool) {
	pos: u32 = 0

	if offset == 0 && decoder.pos == 0 {
		return 0, false
	}

	if offset == 1 {
		pos = decoder.pos
	} else if offset == 0 {
		pos = decoder.pos - 1
	} else {
		pos = decoder.pos + offset
	}

	if len(decoder.memory.buff) > pos {
		return decoder.memory.buff[pos], true
	} else {
		return 0, false
	}
}

main :: proc() {
	fmt.printfln("Hello, world")
}
