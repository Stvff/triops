global _start

section .text

_start:
	mov rax, 1		; write syscall
	mov rdi, 1		; stdout
	mov rsi, msg.1
	mov rdx, msglen.1
	syscall
	
	mov rax, 60		; exit syscall
	mov rdi, 0		; success
	syscall
	
section .rodata
	msg.1: db "Hewwoo >.<", 10	; 10 is newline
	msglen.1: equ $ - msg.1			; equ defines a constant, $ is current pos, msg is the data pos
