package main

import "core:fmt"
import "core:os"

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
	file := "../part-1/listing_0042_completionist_decode"

	mem := Memory{}
	fd, err := os.open(file, os.O_RDONLY)
	if err != nil {
		panic(string(err)) // TODO
	}
	load_machine_code(fd, &mem, 0)
	os.close(fd)


}
