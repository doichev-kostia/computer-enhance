#!/usr/local/bin/fish

# Check if argument is provided
if test (count $argv) -ne 1
    echo "Usage: $argv[0] <filename.asm>"
    exit 1
end

set file $argv[1]

# Check if file exists
if not test -f $file
    echo "File not found: $file"
    exit 1
end

# Check if file ends with .asm
if not string match -q "*.asm" $file
    echo "File must have .asm extension"
    exit 1
end

# Get directory and basename separately
set dirname (dirname $file)
set basename (basename $file .asm)

# Assemble using nasm, preserving directory structure
nasm -o $dirname/$basename $file

# Create binary representation using xxd
xxd -b $dirname/$basename > $dirname/$basename.bits