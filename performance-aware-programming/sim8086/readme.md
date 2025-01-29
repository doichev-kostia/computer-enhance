# 8086 simulation

This is a part of the "performance-aware programming" course. This project has git tags as reference to the milestones
of the course.

## Tags

__part-1-slow-decode__. The only feature implemented is instruction decoding. Samples of those instructions can be
found in the `../part-1` directory. There are no optimizations, and the decoding is straight like a door. Basically,
it's a big `switch case` with pattern matching. 


## Running
`go run ./main.go ../part-1/listingxxx`

## Resources

8086 manual
https://edge.edx.org/c4x/BITSPilani/EEE231/asset/8086_family_Users_Manual_1_.pdf

Page 160 (pdf) - instruction encoding

homework
https://github.com/cmuratori/computer_enhance/tree/main/perfaware/part1

final version of 8086 simulator
https://github.com/cmuratori/computer_enhance/tree/main/perfaware/sim86

