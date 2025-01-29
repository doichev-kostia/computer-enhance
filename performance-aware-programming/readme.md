
## Assembling
`ls ./part-1/ | grep .asm | xargs -P 4 -I {} ./assemble.fish ./part-1/{}`
or
`./assemble.fish ./part-1/<filename>.asm`
