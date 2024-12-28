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
	addf(&nasm.data_sec, "\nsection .data\n")

	addf(&nasm.text_sec, "\nsection .text\n_start:\n")
	//text_sec.WriteString("\tpush rbx; Triops: cdecl politeness\n")
	//text_sec.WriteString("\tpush rbp; Triops: idem\n")

	/* collecting all variables, concluding their offsets */
	addf(&nasm.text_sec, "\t; Triops: Global variable intialization\n")

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
	// fmt.Println(nasm.var_poss)
	// fmt.Println(var_amounts)
	// fmt.Println(nasm.stack_offsets)
	addf(&nasm.text_sec, "\tsub rsp, %v; Triops: This is the size of all variables on the stack\n", nasm.stack_offsets[5])

	for name, decl := range scope.decls {
		gen_init_decl(&nasm, decl, name)
	}

	addf(&nasm.text_sec, "\n\t; Triops: User code\n")
	for _, instruction := range scope.assembly.instructions {
		addf(&nasm.text_sec, "\t")
		nasm.text_sec.WriteString(token_txt_str(instruction.mnemonic, set.text))
		for arg_nr, arg := range instruction.args {
			has_verbatim := arg.verbatim.pos != 0
			has_immediate := arg.immediate.len != 0
			if !has_verbatim && !has_immediate {
				break
			}
			if arg_nr != 0 { addf(&nasm.text_sec, ",") }

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
				addf(&nasm.text_sec, " %v [%v + %v]", indexing_word(instruction.alignment), verbatim_str, v*instruction.alignment)
			} else if has_verbatim && !has_immediate {
				addf(&nasm.text_sec, " %v", verbatim_str)
			} else if !has_verbatim && has_immediate {
				v, _ := value_to_integer(arg.immediate)
				addf(&nasm.text_sec, " %v", v)
			}
		}
		addf(&nasm.text_sec, "\n")
	}
	addf(&nasm.text_sec, "\n\t; Triops: leaving the stack as I found it\n")
	addf(&nasm.text_sec, "\tadd rsp, %v; Triops: This was the size of all variables on the stack\n", nasm.stack_offsets[5])
	/* TODO: think of linking options or just program options that change the behaviour of exiting code */
	addf(&nasm.text_sec, "\n\t; Triops: Adding the unix exit, in case the user doesn't add one\n")
	addf(&nasm.text_sec, "\tmov rax, 60; Triops: 60 is exit\n")
	addf(&nasm.text_sec, "\tmov rdi, 0; Triops: 0 is success\n")
	addf(&nasm.text_sec, "\tsyscall\n")
	//nasm.text_sec.WriteString("\tpush rbp; Triops: cdecl politeness\n")
	//nasm.text_sec.WriteString("\tpush rbx; Triops: idem\n")

	var full_file strings.Builder
	addf(&full_file, "; This code is generated by Triops (in first person). If I have something to say, it'll be prepended with 'Triops'.\n")
	addf(&full_file, "; Triops version: 0\n")
	addf(&full_file, "\nglobal _start\n")
	addf(&full_file, nasm.text_sec.String())
	addf(&full_file, nasm.data_sec.String())
	fmt.Println(full_file.String())
	return true
}

func gen_init_decl(nasm *Nasm, decl Decl_Des, name string) {
	var_pos := nasm.var_poss[name]
	stack_pos := nasm.stack_offsets[var_pos.l2_alignment] + l2_to_align[var_pos.l2_alignment]*var_pos.index
	addf(&nasm.text_sec, "\t; Triops: init `%v`\n", name)

	/* putting the init data in the text section */
	addf(&nasm.data_sec, "\tvardata.%v:\n", name)
	addf(&nasm.data_sec, "\tdb ")
	for i := range decl.init.len {
		if i != 0 { addf(&nasm.data_sec, ", ") }
		addf(&nasm.data_sec, "%v", all_values[decl.init.pos + i])
	}
	addf(&nasm.data_sec, "\n")
	
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
			type_align := align_of_type(decl.typ)
			index_word := indexing_word(type_align)
			ival := decl.init
			ival.len = type_align
			for i := range bare_types[ti_i].amount {
				integer, _ := value_to_integer(ival)
				addf(&nasm.text_sec, "\tmov %v [rsp + %v], %v\n", index_word, stack_pos + i*type_align, integer)
				ival.pos += type_align
			}
		case TYPE_INDIRECT:
			/* already dealt with before this switch */
		case TYPE_RT_ARRAY:
			/* put stuff in data section, which itself might need initialization, so, yipiee */
			array_data := rt_array_types[ti_i]
			below_ti := follow_type(ti)
			ti_i, ti_t = unpack_ti(below_ti)
			if ti_t == TYPE_BARE /*|| !is_reference_type(below_ti)*/ {
				addf(&nasm.text_sec, "\tmov %v [rsp + %v], vardata.%v + 8\n", indexing_word(align_of_type(ti)), stack_pos, name)
				addf(&nasm.text_sec, "\tmov %v [rsp + %v], %v\n", indexing_word(8), stack_pos + 8, array_data.size)
			} else {
				addf(&nasm.text_sec, "AAAAAAA\n")
			}
		case TYPE_ST_ARRAY:
			/* looping over indices */
			array_data := st_array_types[ti_i]
			addf(&nasm.text_sec, "\tpush rax\n\txor rax, rax\n")
			addf(&nasm.text_sec, "\tinitiation_loop.%v:\n", name)

			addf(&nasm.text_sec, "\t\tinc rax\n")
			addf(&nasm.text_sec, "\t\tcmp rax, %v\n", array_data.size)
			addf(&nasm.text_sec, "\t\tljmp initiation_loop.%v\n", name)
			addf(&nasm.text_sec, "\tpop rax\n")
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

func addf(builder *strings.Builder, format string, a ...any) {
	builder.WriteString(fmt.Sprintf(format, a...))
}
