; ========================================================================
;
; (C) Copyright 2023 by Molly Rocket, Inc., All Rights Reserved.
;
; This software is provided 'as-is', without any express or implied
; warranty. In no event will the authors be held liable for any damages
; arising from the use of this software.
;
; Please see https://computerenhance.com for further information
;
; ========================================================================

; ========================================================================
; LISTING 39
; ========================================================================

bits 16

; Register-to-register
mov si, bx
mov dh, al

; Source address calculation
mov al, [bx + si]
mov bx, [bp + di]
mov dx, [bp]

; Source address calculation plus 8-bit displacement
mov ah, [bx + si + 4]

; Source address calculation plus 16-bit displacement
mov al, [bx + si + 4999]

; Dest address calculation
mov [bx + di], cx
mov [bp + si], cl
mov [bp], ch

; Direct address
mov bp, [5]
mov bx, [3458]

