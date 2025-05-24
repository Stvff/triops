package main

import (
	"fmt"
	"strings"
	"slices"
)

type Var_Pos struct {
	is_on_stack bool
	index int
	reg string
}

type Nasm struct {
	text_sec strings.Builder
	data_sec strings.Builder
	stack_size int
	var_poss map[string]Var_Pos
}

func generate_assembly(scope *Scope, set *Token_Set, scope_path string, make_small_exe bool) (string, bool) {
	var (
		nasm Nasm
	)
	addf(&nasm.text_sec, "_start:\n")
	//text_sec.WriteString("\tpush rbx; Triops: cdecl politeness\n")
	//text_sec.WriteString("\tpush rbp; Triops: idem\n")

	/* collecting all variables, concluding their offsets */
	addf(&nasm.text_sec, "\t; Triops: Variable intialization for %v\n", scope_path)
	var ordered_vars [5][]What;
	for _, this := range scope.names {
		if this.named_thing != NAME_DECL { continue }
		decl := all_decls[this.index]
		align := align_of_type(decl.typ)
		ordered_vars[log2(align)] = append(ordered_vars[log2(align)], this)
	}

	nasm.var_poss = make(map[string]Var_Pos)
	for tier := range ordered_vars {
		nasm.stack_size += (pow2[tier] - nasm.stack_size%pow2[tier]) % pow2[tier]
		for _, this := range ordered_vars[tier] {
			decl := all_decls[this.index]
			var vp Var_Pos
			if decl.has_bound_register {
				vp.is_on_stack = false
				reg := all_registers[decl.bound_register]
				vp.reg = token_txt_str(reg.token, set.text)
			} else {
				vp.is_on_stack = true
				vp.index = nasm.stack_size
				nasm.stack_size += size_of_type(decl.typ)
			}
			nasm.var_poss[this.name] = vp
		}
	}

	addf(&nasm.text_sec, "\tsub rsp, %v; Triops: This is the size of all variables on the stack\n", nasm.stack_size)

	for tier := range ordered_vars {
		for _, this := range ordered_vars[tier] {
			gen_init_decl(&nasm, all_decls[this.index], this.name)
		}
	}

	addf(&nasm.text_sec, "\n\t; Triops: User code\n")

	for link_nr := 0; link_nr < len(scope.code); link_nr += 1 {
		link := scope.code[link_nr]
		if link.kind != LKIND_SEMICOLON { continue }

		mnm_nr := link.right
		mnm_node := all_nodes[mnm_nr]
		if mnm_node.kind == NKIND_LABEL {
			addf(&nasm.text_sec, "\t%v.%v:\n", scope_path, token_txt_str(mnm_node.token, set.text));
			continue
		}
		addf(&nasm.text_sec, "\t%v", token_txt_str(mnm_node.token, set.text));

		for arg_nr := range mnm_node.satisfied_right {
			if arg_nr != 0 { addf(&nasm.text_sec, ",") }

			link_nr += 1
			for ; link_nr < len(scope.code); link_nr += 1 {
				link = scope.code[link_nr]
				if link.kind == LKIND_RIGHT_ARG {
					break
				}
			}
			arg_index := link.right
			arg_node := all_nodes[arg_index]
			do_offset := false
			var (
				alignment int
				source string
				offset int
			)
			for ; arg_node.kind == NKIND_INDEX; {
				alignment = align_of_type(arg_node.ti)
				v, _ := value_to_integer(arg_node.imm)
				offset += alignment*v
				do_offset = true
				arg_index -= 1
				arg_node = all_nodes[arg_index]
			}
			node_string := token_txt_str(arg_node.token, set.text)
			switch arg_node.kind {
				case NKIND_IMMEDIATE:
					v, _ := value_to_integer(arg_node.imm)
					addf(&nasm.text_sec, " %v", v)
				case NKIND_INDEX: panic("Should be taken care of")
				case NKIND_VARIABLE:
					var_pos := nasm.var_poss[node_string]
					alignment = align_of_type(arg_node.ti)
					source = "rsp"
					offset += var_pos.index
					do_offset = true
				case NKIND_REGISTER:
					addf(&nasm.text_sec, " %v", node_string);
				case NKIND_LABEL:
					addf(&nasm.text_sec, " %v.%v", scope_path, node_string)
			}
			if do_offset {
				addf(&nasm.text_sec, " %v [%v + %v]", indexing_word(alignment), source, offset);
			}
		}
		addf(&nasm.text_sec, "\n")
	}

	addf(&nasm.text_sec, "\n\t; Triops: leaving the stack as I found it\n")
	addf(&nasm.text_sec, "\tadd rsp, %v; Triops: This was the size of all variables on the stack\n", nasm.stack_size)
	/* TODO: think of linking options or just program options that change the behaviour of exiting code */
	addf(&nasm.text_sec, "\n\t; Triops: Adding the unix exit, in case the user doesn't add one\n")
	addf(&nasm.text_sec, "\tmov rax, 60; Triops: 60 is exit\n")
	addf(&nasm.text_sec, "\tmov rdi, 0; Triops: 0 is success\n")
	addf(&nasm.text_sec, "\tsyscall\n")
	//nasm.text_sec.WriteString("\tpush rbp; Triops: cdecl politeness\n")
	//nasm.text_sec.WriteString("\tpush rbx; Triops: idem\n")

	return nasm_file_preamble(&nasm, make_small_exe), true
}

func gen_init_decl(nasm *Nasm, decl Decl_Des, name string) {
	var_pos := nasm.var_poss[name]

	addf(&nasm.text_sec, "\n\t; Triops: init `%v`\n", name)
	if decl.init.len == 0 {
		if !var_pos.is_on_stack {
			addf(&nasm.text_sec, "\tmov %v, 0; Triops: zero init\n", var_pos.reg)
			return
		}
		type_align := align_of_type(decl.typ)
		index_word := indexing_word(type_align)
		amount := amount_of_type(decl.typ)
		for i := range amount {
			addf(&nasm.text_sec, "\tmov %v [rsp + %v], 0; Triops: zero init\n", index_word, var_pos.index + i*type_align)
		}
		return
	}
	
	/* several init types */
	ti := decl.typ
	ti_i, ti_t := unpack_ti(ti)
	for ti_t == TYPE_INDIRECT {
		ti = indirect_types[ti_i].target
		ti_i, ti_t = unpack_ti(ti)
	}
	switch ti_t {
		case TYPE_ERR: panic("codegen: internal type error")
		case TYPE_BARE:
			/* make assignment like normal */
			ival := decl.init
			if !var_pos.is_on_stack {
				ival.len = size_of_type(decl.typ)
				integer, _ := value_to_integer(ival)
				addf(&nasm.text_sec, "\tmov %v, %v; Triops: zero init\n", var_pos.reg, integer)
				return
			} else {
				type_align := align_of_type(decl.typ)
				index_word := indexing_word(type_align)
				ival.len = type_align
				for i := range bare_types[ti_i].amount {
					integer, _ := value_to_integer(ival)
					addf(&nasm.text_sec, "\tmov %v [rsp + %v], %v\n", index_word, var_pos.index + i*type_align, integer)
					ival.pos += type_align
				}
			}
		case TYPE_INDIRECT: /* already dealt with before this switch */
		case TYPE_RT_ARRAY:
			/* put stuff in data section, which itself might need initialization, so, yipiee */
			if !var_pos.is_on_stack { panic("Someone managed to fit an array into a register") }
//			array_data := rt_array_types[ti_i]
			below_ti := follow_type(ti)
			ti_i, ti_t = unpack_ti(below_ti)
			label_name := fmt.Sprintf("vardata.%v", name)
//			data_size, _ := value_to_integer(Value{VALUE_FORM_NONE, decl.init.pos, 8})

			addf(&nasm.data_sec, "\t%v:\n", label_name)
			data_sec_offset := 0;
			decl_init_copy := decl.init;
			ledger := make([]Nested_Array_Ledge, 1)
			ledger[0] = Nested_Array_Ledge{ 0, 0, 0 }
			recurse_gen_init_decl(&nasm.data_sec, label_name, ti, 0, &data_sec_offset, &decl_init_copy, &ledger)

			addf(&nasm.text_sec, "\tmov %v [rsp + %v], %v + %v\n", indexing_word(align_of_type(ti)), var_pos.index, label_name, ledger[1].offset)
			addf(&nasm.text_sec, "\tmov %v [rsp + %v], %v\n", indexing_word(8), var_pos.index + 8, ledger[1].element_amount)
			// addf(&nasm.text_sec, "\tmov %v [rsp + %v], %v + %v\n", indexing_word(align_of_type(ti)), stack_pos, label_name, data_sec_offset)
			// addf(&nasm.text_sec, "\tmov %v [rsp + %v], %v\n", indexing_word(8), stack_pos + 8, data_size)
		case TYPE_ST_ARRAY: panic("codegen: static array")
		case TYPE_POINTER: panic("codegen: pointer")
		case TYPE_STRUCT: panic("codegen: struct")
	}

//	nasm.text_sec.WriteString(fmt.Sprintf("\tmov %v [rsp + %v], 0; Triops: init `%v`\n", indexing_word(align_of_type(decl.typ)), stack_pos, name))
}

type Nested_Array_Ledge struct {
	offset, element_amount, depth int
}

func recurse_gen_init_decl(data_sec *strings.Builder, name string, ti Type_Index, depth int, data_sec_offset *int, value *Value, ledger *[]Nested_Array_Ledge) {
	//addf(&nasm.text_sec, "\t; Triops: entering depth %v of init\n", depth)
	_, ti_t := unpack_ti(ti)
	switch ti_t {
		case TYPE_ERR: panic("codegen: internal type error")
		case TYPE_BARE:
			// bare_typesize := size_of_type(ti)
			// add_value_to_section(data_sec, "", Value{VALUE_FORM_NONE, value.pos, bare_typesize})
			// value.pos += bare_typesize
			// value.len -= bare_typesize
			// *data_sec_offset += bare_typesize
		case TYPE_INDIRECT: /* already dealt with before this switch */
		case TYPE_RT_ARRAY:
			below_ti := follow_type(ti)
			_, below_ti_t := unpack_ti(below_ti)
			// if value.len == 0 {
			// 	addf(data_sec, "\t\tdq 0; Triops: Dynamic array initialized to zero\n", depth)
			// 	addf(data_sec, "\t\tdq 0", depth)
			// 	*ledger = append(*ledger, Nested_Array_Ledge{*data_sec_offset + 8, 0, depth})
			// 	return
			// }

			element_amount, _ := value_to_integer(Value{VALUE_FORM_NONE, value.pos, 8})
			value.pos += 8
			value.len -= 8

			array_offset := *data_sec_offset
			if below_ti_t == TYPE_BARE {
				*data_sec_offset += 8;
				array_offset = *data_sec_offset
				addf(data_sec, "\t\tdq 0; Triops: Dynamic array, depth: %v\n", depth)
				bare_typesize := size_of_type(below_ti)
				addf(data_sec, "\t\t%v ", defining_word(bare_typesize))
				for i := range element_amount {
					if i != 0 { addf(data_sec, ", ") }
					nr, _ := value_to_integer(Value{VALUE_FORM_NONE, value.pos, bare_typesize})
					addf(data_sec, "%v", nr)
					value.pos += bare_typesize
					value.len -= bare_typesize
					*data_sec_offset += bare_typesize
				}
				addf(data_sec, "\n")
			} else {
				for range element_amount {
					recurse_gen_init_decl(data_sec, name, below_ti, depth + 1, data_sec_offset, value, ledger)
				}
			}

			last_ledge := (*ledger)[len(*ledger)-1]
			if depth < last_ledge.depth {
				addf(data_sec, "\t\tdq 0; Triops: Dynamic array, depth: %v, last_depth: %v\n", depth, last_ledge.depth)
				*data_sec_offset += 8;
				array_offset = *data_sec_offset
				for _, ledge := range (*ledger)[1:] {
					addf(data_sec, "\t\tdq %v + %v, %v\n", name, ledge.offset, ledge.element_amount)
					*data_sec_offset += 16;
				}
				*ledger = slices.Delete(*ledger, 1, len(*ledger))
			}
			*ledger = append(*ledger, Nested_Array_Ledge{array_offset, element_amount, depth})
			addf(data_sec, "\n")
			//addf(data_sec, "\t\tdq %v, %v\n", data_size_offset, element_amount)
		case TYPE_ST_ARRAY: panic("codegen: static array")
		case TYPE_POINTER: panic("codegen: pointer")
		case TYPE_STRUCT: panic("codegen: struct")
	}
	//addf(&nasm.text_sec, "\t; Triops: exiting depth %v of init\n", depth)
}

func add_value_to_data_section(nasm *Nasm, name string, value Value) {
	if len(name) != 0 {
		addf(&nasm.data_sec, "\t%v:\n", name)
	}
	addf(&nasm.data_sec, "\t\tdb ")
	for i := range value.len {
		if i != 0 { addf(&nasm.data_sec, ", ") }
		addf(&nasm.data_sec, "%v", all_values[value.pos + i])
	}
	addf(&nasm.data_sec, "\n")
}

func add_value_to_section(section *strings.Builder, name string, value Value) {
	if len(name) != 0 {
		addf(section, "\t%v:\n", name)
	}
	addf(section, "\t\tdb ")
	for i := range value.len {
		if i != 0 { addf(section, ", ") }
		addf(section, "%v", all_values[value.pos + i])
	}
	addf(section, "\n")
}

func indexing_word(alignment int) string {
	switch alignment {
		case 1: return "byte"
		case 2: return "word"
		case 4: return "dword"
		case 8: return "qword"
	}
	return ""
}

func defining_word(alignment int) string {
	switch alignment {
		case 1: return "db"
		case 2: return "dw"
		case 4: return "dd"
		case 8: return "dq"
	}
	return ""
}

func addf(builder *strings.Builder, format string, a ...any) {
	builder.WriteString(fmt.Sprintf(format, a...))
}

func nasm_file_preamble(nasm *Nasm, make_small_exe bool) string {
	header_small_exe := `; Triops: This is a custom ELF header for smaller executables. It is less memorysafe.
; Triops: It has annotations not prepended with 'Triops', for brevity's sake.
BITS 64
	org     0x08048000
elf_header:
	; e_ident
	db 0x7F, "ELF"
	db 2		; 64 bit
	db 1		; little endian
	db 1		; version
	db 0		; ABI: UNIX system V (3 is GNU/Linux, but I'm not using any GNU)
	db 0		; more ABI (also dynlinker ABI for glibc?)
	db 0, 0, 0, 0, 0, 0, 0 ; 7 zeroes to pad

	dw 2		; e_type: executable
	dw 0x3E		; e_machine: AMD X86-64
	dd 1		; e_version
	dq _start		; e_entry
	dq program_header - $$		; e_phoff: program header offset (should be 0x40)
	dq 0		; e_shoff: section header offset (we don't have this rn)
	dd 0		; e_flags
	dw 0x40		; e_ehsize: elf header size (should be 0x40 for 64 bit)
	dw 0x38		; e_phentsize: program header entry size (should be 0x38 for 64 bit)
	dw 1		; e_phnum: number of program header entries
	dw 0		; e_shentsize: section header stuff. we don't have it, so it's all zero
	dw 0		; e_shnum
	dw 0		; shstrndx

program_header:
	dd 1		; p_type: loadable segment (PT_LOAD)
	dd 0x7		; p_flags: executable (1) + writeable (2) + readable (4)
	dq 0		; p_offset (I think this is offset from _start)
	dq $$		; p_vaddr: $$ is that org number at the top, which is where it'll get loaded in
	dq $$		; p_paddr (I'm pretty sure this is ignored)
	dq filesize		; p_filesz
	dq filesize		; p_memsz
	dq 0x1000		; p_align: align to linux page size (4096 bytes)
`

	var full_file strings.Builder
	addf(&full_file, "; This code is generated by Triops (in first person). If I have something to say, it'll be prepended with 'Triops'.\n")
	addf(&full_file, "; Triops version: 2\n")

	if make_small_exe {
		addf(&full_file, "\n%v\n", header_small_exe)
	} else {
		addf(&full_file, "\nglobal _start\n")
		addf(&full_file, "\nsection .text\n")
	}
	addf(&full_file, "%v", nasm.text_sec.String())
	if make_small_exe {
		addf(&full_file, "\n.data:\n")
	} else {
		addf(&full_file, "\nsection .data\n")
	}
	addf(&full_file, "%v", nasm.data_sec.String())
	if make_small_exe {
		addf(&full_file, "filesize equ $ - $$; Triops: This is for the custom ELF header.\n")
	}
	return full_file.String()
}

var pow2 = [5]int{1, 2, 4, 8, 16}

func log2(align int) int {
	switch align {
	case  1: return 0
	case  2: return 1
	case  4: return 2
	case  8: return 3
	case 16: return 4
	}
	panic("log2: was only expecting normal numbers")
}