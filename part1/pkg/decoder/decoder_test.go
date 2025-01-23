package decoder

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestDecoding(t *testing.T) {
	files := []string{
		"../../assets/listing_0037_single_register_mov",
		"../../assets/listing_0038_many_register_mov",
		"../../assets/reg-memory-with-displacement",
		"../../assets/listing_0039_more_movs",
		"../../assets/signed-displacement",
		"../../assets/listing_0040_challenge_movs",
		"../../assets/listing_0041_add_sub_cmp_jnz",
		"../../assets/listing_0042_completionist_decode",
	}

	for _, filename := range files {
		source, err := os.ReadFile(filename)
		if err != nil {
			t.Errorf("%s = %v", filename, err)
		}

		decoder := NewDecoder(source)
		var contents []byte

		func() {
			defer func() {
				if r := recover(); r != nil {
					decoded := decoder.GetDecoded()
					if len(decoded) > 0 {
						t.Logf("(%s) Partial decoded contents:\n%s", filename, decoded)
					}
					panic(fmt.Errorf("panic occurred when processing %s; error = %v", filename, r))
				}
			}()

			var err error
			contents, err = decoder.Decode()
			if err != nil {
				decoded := decoder.GetDecoded()
				if len(decoded) > 0 {
					t.Logf("(%s) Partial decoded contents:\n%s", filename, decoded)
				}
				t.Errorf("%s = %v", filename, err)
			}
		}()

		asm := []byte("bits 16\n\n")
		asm = append(asm, contents...)

		verifyAssembled(t, asm, source, filename)
	}
}

func verifyAssembled(t *testing.T, asm []byte, source []byte, filename string) {
	tmpIn, err := os.CreateTemp(os.TempDir(), "*")
	if err != nil {
		t.Errorf("%s = %v", filename, err)
	}

	if _, err = tmpIn.Write(asm); err != nil {
		t.Errorf("%s; failed to flush the decoded asm. err = %v\n", filename, err)
	}

	tmpOut, err := os.CreateTemp(os.TempDir(), "*")
	if err != nil {
		t.Errorf("%s = %v", filename, err)
	}

	nasm := exec.Command("nasm", "-o", tmpOut.Name(), tmpIn.Name())

	err = nasm.Run()
	if err != nil {
		t.Errorf("nasm err: %s = %v", filename, err)
	}

	tmpOut.Close()
	tmpIn.Close()

	fmt.Printf("in %s; out %s \n", tmpIn.Name(), tmpOut.Name())

	assembled, err := os.ReadFile(tmpOut.Name())

	if len(assembled) != len(source) {
		t.Errorf("%s there is a length mismatch between the source and assembled output", filename)
	}

	for idx, b := range assembled {
		if b != source[idx] {
			t.Errorf("%s: byte %d doesn't match, expected %d, got %d", filename, idx, source[idx], b)
			break
		}
	}
}
