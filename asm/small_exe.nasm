BITS 64
	org     0x08048000
elf_header:
	; e_ident
	db 0x7F, "ELF"
	db 2		; 64 bit
	db 1		; little endian
	db 1		; version
	db 3		; ABI: linux
	db 0		; more ABI (also dynlinker ABI for glibc?)
	db 0, 0, 0, 0, 0, 0, 0 ; 7 zeroes to pad

	dw 2		; e_type: executable
	dw 0x3E		; e_machine: AMD X86-64
	dd 1		; e_version
	dq _start		; e_entry
	dq program_header - $$		; e_phoff: program header offset (should be 0x40)
	dq 0		; e_shoff: section header offset (we don't have this rn)
	dd 0		; e_flags
	dw elf_header_size		; e_ehsize: elf header size (should be 0x40)
	dw program_header_size		; e_phentsize: program header entry size (should be 0x38)
	dw 1		; e_phnum: number of program header entries
	dw 0		; e_shentsize: section header stuff. we don't have it, so it's all zero
	dw 0		; e_shnum
	dw 0		; shstrndx

elf_header_size equ $ - elf_header

program_header:
	dd 1		; p_type: loadable segment (PT_LOAD)
	dd 0x7		; p_flags: executable (1) + writeable (2) + readable (4)
	dq 0		; p_offset (not sure why this is that value)
	dq $$		; p_vaddr (not sure why this is that value)
	dq $$		; p_paddr (not sure why this is that value)
	dq filesize		; p_filesz
	dq filesize		; p_memsz
	dq 0x1000       ; p_align (not sure why this is that value)

program_header_size equ $ - program_header

_start:
	; Triops: Global variable intialization
	sub rsp, 64; Triops: This is the size of all variables on the stack

	; Triops: init `greeting`
	mov qword [rsp + 0], vardata.greeting + 8
	mov qword [rsp + 8], 10

	; Triops: init `data`
	mov qword [rsp + 16], 0; Triops: zero init

	; Triops: init `length`
	mov qword [rsp + 24], 0; Triops: zero init

	; Triops: init `system_function`
	mov qword [rsp + 32], 0; Triops: zero init

	; Triops: init `stream`
	mov qword [rsp + 40], 0; Triops: zero init

	; Triops: init `error_code`
	mov qword [rsp + 48], 0; Triops: zero init

	; Triops: User code
	; Triops: binding data
	mov rsi,  [rsp + 16]

	; Triops: binding length
	mov rdx,  [rsp + 24]

	; Triops: binding system_function
	mov rax,  [rsp + 32]

	; Triops: binding stream
	mov rdi,  [rsp + 40]

	mov rsi, qword [rsp + 0 + 0]
	mov rdx, qword [rsp + 0 + 8]
	mov rax, 1
	mov rdi, 1
	syscall
	; Triops: binding error_code
	mov rdi,  [rsp + 48]

	mov rax, 60
	mov rdi, 0
	syscall

	; Triops: leaving the stack as I found it
	add rsp, 64; Triops: This was the size of all variables on the stack

	; Triops: Adding the unix exit, in case the user doesn't add one
	mov rax, 60; Triops: 60 is exit
	mov rdi, 0; Triops: 0 is success
	syscall

;section .data
	vardata.greeting:
		dq 0; Triops: Dynamic array, depth: 0
		db 72, 105, 32, 116, 104, 101, 114, 101, 33, 10

filesize equ $ - $$
