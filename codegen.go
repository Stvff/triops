package main

import (
	"fmt"
	"strings"
)

type Var_Pos struct {
	l2_alignment int
	index int
}

var l2_to_align = [5]int{1, 2, 4, 8, 16}

type Var_Amounts struct {
	vars_1B, vars_2B, vars_4B, vars_8B, vars_16B int
//	frame_size int
}

type Nasm struct {
	text_sec strings.Builder
	data_sec strings.Builder
	stack_offsets [6]int
	var_poss map[string]Var_Pos
}

func generate_assembly(scope *Scope, set *Token_Set) bool {
	var (
		nasm Nasm
		var_amounts Var_Amounts
	)
	nasm.data_sec.WriteString("\nsection .data\n")

	nasm.text_sec.WriteString("\nsection .text\n_start:\n")
	//text_sec.WriteString("\tpush rbx; Triops: cdecl politeness\n")
	//text_sec.WriteString("\tpush rbp; Triops: idem\n")

	/* collecting all variables, concluding their offsets */
	nasm.text_sec.WriteString("\t; Triops: Global variable intialization\n")

	nasm.var_poss = make(map[string]Var_Pos)
	for name, decl := range scope.decls {
		/* logs its existence on the stack */
		amount := amount_of_type(decl.typ)
		align := align_of_type(decl.typ)
		switch align {
		case 1:
			nasm.var_poss[name] = Var_Pos{0, var_amounts.vars_1B}
			var_amounts.vars_1B += amount
		case 2:
			nasm.var_poss[name] = Var_Pos{1, var_amounts.vars_2B}
			var_amounts.vars_2B += amount
		case 4:
			nasm.var_poss[name] = Var_Pos{2, var_amounts.vars_4B}
			var_amounts.vars_4B += amount
		case 8:
			nasm.var_poss[name] = Var_Pos{3, var_amounts.vars_8B}
			var_amounts.vars_8B += amount
		case 16:
			nasm.var_poss[name] = Var_Pos{4, var_amounts.vars_16B}
			var_amounts.vars_16B += amount
		}
	}

	nasm.stack_offsets[0] = 0
	nasm.stack_offsets[1] = nasm.stack_offsets[0] + var_amounts.vars_1B
	nasm.stack_offsets[1] += (2 - nasm.stack_offsets[1]%2) % 2
	nasm.stack_offsets[2] = nasm.stack_offsets[1] + 2*var_amounts.vars_2B
	nasm.stack_offsets[2] += (4 - nasm.stack_offsets[2]%4) % 4
	nasm.stack_offsets[3] = nasm.stack_offsets[2] + 4*var_amounts.vars_4B
	nasm.stack_offsets[3] += (8 - nasm.stack_offsets[3]%8) % 8
	nasm.stack_offsets[4] = nasm.stack_offsets[3] + 8*var_amounts.vars_8B
	nasm.stack_offsets[4] += (16 - nasm.stack_offsets[4]%16) % 16
	nasm.stack_offsets[5] = nasm.stack_offsets[4] + 16*var_amounts.vars_16B
	nasm.stack_offsets[5] += (16 - nasm.stack_offsets[5]%16) % 16 /* aligns the top to 16B */
	fmt.Println(nasm.var_poss)
	fmt.Println(var_amounts)
	fmt.Println(nasm.stack_offsets)
	nasm.text_sec.WriteString(fmt.Sprintf("\tsub rsp, %v; Triops: This is the size of all variables on the stack\n", nasm.stack_offsets[5]))

	for name, decl := range scope.decls {
		gen_init_decl(&nasm, decl, name)
	}
/*
		fmt.Println(name, decl)
		amount := amount_of_type(decl.typ)
		align := align_of_type(decl.typ)
		does the value initialization
		i, t := unpack_ti(decl.typ)
		try_over:
		switch t {
			case TYPE_ERR: panic("codegen: internal type error")
			case TYPE_BARE:
				
			case TYPE_INDIRECT:
				ti = indirect_types[i].target
			case TYPE_RT_ARRAY:
				ti = rt_array_types[i].target
			case TYPE_ST_ARRAY:
				ti = st_array_types[i].target
			case TYPE_POINTER:
				ti = pointer_types[i].target
			case TYPE_STRUCT:
				panic("codegen: struct")
		}
		i, t = unpack_ti(ti)
		for t == TYPE_INDIRECT {
			ti = indirect_types[i].target
			i, t = unpack_ti(ti)
		}
	}
*/

	nasm.text_sec.WriteString("\n\t; Triops: User code\n")
	for _, instruction := range scope.assembly.instructions {
		nasm.text_sec.WriteString("\t")
		nasm.text_sec.WriteString(token_txt_str(instruction.mnemonic, set.text))
		for arg_nr, arg := range instruction.args {
			has_verbatim := arg.verbatim.pos != 0
			has_immediate := arg.immediate.len != 0
			if !has_verbatim && !has_immediate {
				break
			}
			if arg_nr != 0 { nasm.text_sec.WriteString(",") }

			/* TODO: differentiate between registers, labels and variables in the 'verbatim' catagory
			   We might want to check if a register is rsp, and warn once for that. We can do that
			   in the backend, but not in the frontend.*/
//			offset := 0
			verbatim_str := token_txt_str(arg.verbatim, set.text)
			if has_verbatim && !(DIRECTIVE_REGS_START < arg.verbatim.tag && arg.verbatim.tag < DIRECTIVE_REGS_END) {
				pos, exists := nasm.var_poss[verbatim_str]
				if !exists {
					fmt.Println(verbatim_str, DIRECTIVE_REGS_START, arg.verbatim.tag, DIRECTIVE_REGS_END)
					panic("codegen: There was an unrecognized variable all the way in codegen")
				}
				verbatim_str = fmt.Sprintf("rsp + %v", nasm.stack_offsets[pos.l2_alignment] + l2_to_align[pos.l2_alignment]*pos.index)
			}

			if has_verbatim && has_immediate {
				v, _ := value_to_integer(arg.immediate)
				nasm.text_sec.WriteString(fmt.Sprintf(" %v [%v + %v]", indexing_word(instruction.alignment), verbatim_str, v*instruction.alignment))
			} else if has_verbatim && !has_immediate {
				nasm.text_sec.WriteString(fmt.Sprintf(" %v", verbatim_str))
			} else if !has_verbatim && has_immediate {
				v, _ := value_to_integer(arg.immediate)
				nasm.text_sec.WriteString(fmt.Sprintf(" %v", v))
			}
		}
		nasm.text_sec.WriteString("\n")
	}
	nasm.text_sec.WriteString("\n\t; Triops: leaving the stack as I found it\n")
	nasm.text_sec.WriteString(fmt.Sprintf("\tadd rsp, %v; Triops: This was the size of all variables on the stack\n", nasm.stack_offsets[5]))
	/* TODO: think of linking options or just program options that change the behaviour of exiting code */
	nasm.text_sec.WriteString("\n\t; Triops: Adding the unix exit, in case the user doesn't add one\n")
	nasm.text_sec.WriteString("\tmov rax, 60; Triops: 60 is exit\n")
	nasm.text_sec.WriteString("\tmov rdi, 0; Triops: 0 is success\n")
	nasm.text_sec.WriteString("\tsyscall\n")
	//nasm.text_sec.WriteString("\tpush rbp; Triops: cdecl politeness\n")
	//nasm.text_sec.WriteString("\tpush rbx; Triops: idem\n")

	var full_file strings.Builder
	full_file.WriteString("; This code is generated by Triops (in first person). If I have something to say, it'll be prepended with 'Triops'.\n")
	full_file.WriteString("; Triops version: 0\n")
	full_file.WriteString("\nglobal _start\n")
	full_file.WriteString(nasm.text_sec.String())
	full_file.WriteString(nasm.data_sec.String())
	fmt.Println(full_file.String())
	return true
}

func gen_init_decl(nasm *Nasm, decl Decl_Des, name string) {
	var_pos := nasm.var_poss[name]
	stack_pos := nasm.stack_offsets[var_pos.l2_alignment] + l2_to_align[var_pos.l2_alignment]*var_pos.index

	
	ti := decl.typ
	ti_i, ti_t := unpack_ti(ti)
	for ti_t == TYPE_INDIRECT {
		ti = indirect_types[ti_i].target
		ti_i, ti_t = unpack_ti(ti)
	}
	switch ti_t {
		case TYPE_ERR: panic("codegen: internal type error")
		case TYPE_BARE:
			nasm.text_sec.WriteString(fmt.Sprintf("\tmov %v [rsp + %v], 0; Triops: init `%v`\n", indexing_word(align_of_type(decl.typ)), stack_pos, name))
			/* make assignment like normal */
		case TYPE_INDIRECT:
			/* loop to top until it's a normal type */
		case TYPE_RT_ARRAY:
			/* put stuff in data section, which itself might need initialization, so, yipiee */
			below_ti := follow_type(ti)
			ti_i, ti_t = unpack_ti(below_ti)
			if ti_t == TYPE_BARE /*|| !is_reference_type(below_ti)*/ {
				array_size := decl.init.len/size_of_type(below_ti)
				nasm.data_sec.WriteString(fmt.Sprintf("\t%v:\n", name))
				nasm.data_sec.WriteString("\tdq 0\n")
				nasm.data_sec.WriteString("\tdb ")
				for i := range decl.init.len {
					if i != 0 { nasm.data_sec.WriteString(", ") }
					nasm.data_sec.WriteString(fmt.Sprintf("%v", all_values[decl.init.pos + i]))
				}
				nasm.data_sec.WriteString("\n")
	
				nasm.text_sec.WriteString(fmt.Sprintf("\tmov %v [rsp + %v], %v + 8; Triops: init `%v`\n", indexing_word(align_of_type(ti)), stack_pos, name, name))
				nasm.text_sec.WriteString(fmt.Sprintf("\tmov %v [rsp + %v], %v\n", indexing_word(8), stack_pos + 8, array_size))
			} else {
				nasm.text_sec.WriteString("AAAAAAA\n")
			}
		case TYPE_ST_ARRAY:
			/* make assignment of the type below this, repeated a bunch of time */
			/*
				push rax
				xor rax, rax
				initiation_loop.<varname>.<initdepth>:
					<do assignment, so do a function call that generates that>
					cmp rax, <size>
					ljmp initiation_loop.<varname>.<initdepth>
				pop rax
			*/
		case TYPE_POINTER:
			/* see RT_ARRAY */
			ti = pointer_types[ti_i].target
		case TYPE_STRUCT:
			panic("codegen: struct")
	}

//	nasm.text_sec.WriteString(fmt.Sprintf("\tmov %v [rsp + %v], 0; Triops: init `%v`\n", indexing_word(align_of_type(decl.typ)), stack_pos, name))
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
