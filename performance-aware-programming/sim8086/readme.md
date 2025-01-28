
## assembling
`ls ../part-1/ | grep .asm | xargs -P 4 -I {} ./assemble.fish ../part-1/{}`
or
`./assemble.fish ../part-1/<filename>.asm`

## running
`go run ./main.go ./listingxxx`

8086 manual
https://edge.edx.org/c4x/BITSPilani/EEE231/asset/8086_family_Users_Manual_1_.pdf

Page 160 (pdf) - instruction encoding

homework
https://github.com/cmuratori/computer_enhance/tree/main/perfaware/part1

final version of 8086 simulator
https://github.com/cmuratori/computer_enhance/tree/main/perfaware/sim86
