global _start

section .text

_start:
	mov rax, 1		; write syscall
	mov rdi, 1		; stdout
	mov rsi, msg
	mov rdx, msglen
	syscall
	
	mov rax, 60		; exit syscall
	mov rdi, 0		; success
	syscall

section .data
	msg: db "Hewwoo >.<", 10	; 10 is newline
	msglen: equ $ - msg			; equ defines a constant, $ is current pos, msg is the data pos

;	mov al, byte[msg]
;	push rax
;	call increment
;	pop rax
;	mov byte[msg], al
;
;increment:
;	push rbp
;	mov rbp, rsp
;	add byte [rbp + 0x10], 1
;	leave
;	ret
