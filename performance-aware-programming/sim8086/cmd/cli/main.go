package main

import (
	"errors"
	"fmt"
	"github.com/doichev-kostia/computer-enhance/sim8086/pkg/decoder"
	"os"
)

func main() {
	// 1 - program name, 2 - filename
	if len(os.Args) < 2 {
		exit(fmt.Errorf("invalid number of arguments, expected at least one for the filename\n"))
	}

	filename := os.Args[1]
	if !fileExists(filename) {
		exit(fmt.Errorf("The specified file %s doesn't exist\n", filename))
	}

	bytes, err := os.ReadFile(filename)
	if err != nil {
		exit(fmt.Errorf("Failed to read the file %s. Error = %w\n", filename, err))
	}

	d := decoder.NewDecoder(bytes)
	var contents []byte

	func() {
		defer func() {
			if r := recover(); r != nil {
				if len(d.GetDecoded()) > 0 {
					fmt.Printf("(%s) Partial decoded contents:\n%s", filename, d.GetDecoded())
				}
				panic(fmt.Errorf("panic occurred when processing %s; error = %v", filename, r))
			}
		}()

		var err error
		contents, err = d.Decode()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s = %v", filename, err)
		}
	}()

	if err != nil {
		exit(err)
	}

	asm := printHead(filename) + string(contents)

	fmt.Print(asm)
}

func printHead(filename string) string {
	return fmt.Sprintf("; %s\nbits 16\n\n", filename)
}

func exit(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)

	return !errors.Is(err, os.ErrNotExist)
}
