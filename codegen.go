package main

import (
	"fmt"
	"strings"
	"slices"
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

func generate_assembly(scope *Scope, set *Token_Set, scope_path string) (string, bool) {
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
	for _, this := range scope.names {
		if this.named_thing != NAME_DECL { continue }
		name := this.name
		decl := all_decls[this.index]
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

	for _, this := range scope.names {
		if this.named_thing != NAME_DECL { continue }
		gen_init_decl(&nasm, all_decls[this.index], this.name)
	}

	addf(&nasm.text_sec, "\n\t; Triops: User code\n")
	for _, instruction := range scope.assembly.instructions {
		addf(&nasm.text_sec, "\t")
		if token_txt_str(instruction.mnemonic, set.text) == "#lbl" {
			addf(&nasm.text_sec, "%v.%v:\n", scope_path, token_txt_str(instruction.args[0].verbatim, set.text))
			continue
		}
		nasm.text_sec.WriteString(token_txt_str(instruction.mnemonic, set.text))
		for arg_nr, arg := range instruction.args {
			has_verbatim := arg.verbatim.pos != 0
			has_immediate := arg.immediate.len != 0
			//is_register := DIRECTIVE_REGS_START < arg.verbatim.tag && arg.verbatim.tag < DIRECTIVE_REGS_END
			if !has_verbatim && !has_immediate {
				break
			}
			if arg_nr != 0 { addf(&nasm.text_sec, ",") }

			/* TODO: differentiate between registers, labels and variables in the 'verbatim' catagory
			   Also, we might want to check if a register is rsp, and warn once for that. We can do that
			   in the backend, but not in the frontend.*/
//			offset := 0
			verbatim_str := token_txt_str(arg.verbatim, set.text)
			if has_verbatim && arg.verbatim.tag == NONE {
				pos, exists := nasm.var_poss[verbatim_str]
				if !exists {
					fmt.Println(verbatim_str, arg.verbatim.tag)
					panic("codegen: There was an unrecognized variable all the way in codegen")
				}
				verbatim_str = fmt.Sprintf("rsp + %v", nasm.stack_offsets[pos.l2_alignment] + l2_to_align[pos.l2_alignment]*pos.index)
			}

			if has_verbatim && has_immediate {
				v, _ := value_to_integer(arg.immediate)
				addf(&nasm.text_sec, " %v [%v + %v]", indexing_word(instruction.alignment), verbatim_str, v*instruction.alignment)
			} else if has_verbatim && !has_immediate {
				if arg.verbatim.tag == DIRECTIVE_LBL {
					addf(&nasm.text_sec, " %v.%v", scope_path, verbatim_str)
				} else {
					addf(&nasm.text_sec, " %v", verbatim_str)
				}
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
	addf(&full_file, "; Triops version: 1\n")
	addf(&full_file, "\nglobal _start\n")
	addf(&full_file, "%v", nasm.text_sec.String())
	addf(&full_file, "%v", nasm.data_sec.String())
	//fmt.Println(full_file.String())
	return full_file.String(), true
}

func gen_init_decl(nasm *Nasm, decl Decl_Des, name string) {
	var_pos := nasm.var_poss[name]
	stack_pos := nasm.stack_offsets[var_pos.l2_alignment] + l2_to_align[var_pos.l2_alignment]*var_pos.index
	addf(&nasm.text_sec, "\n\t; Triops: init `%v`\n", name)
	if decl.init.len == 0 {
		type_align := align_of_type(decl.typ)
		index_word := indexing_word(type_align)
		amount := amount_of_type(decl.typ)
		for i := range amount {
			addf(&nasm.text_sec, "\tmov %v [rsp + %v], 0; Triops: zero init\n", index_word, stack_pos + i*type_align)
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
			type_align := align_of_type(decl.typ)
			index_word := indexing_word(type_align)
			ival := decl.init
			ival.len = type_align
			for i := range bare_types[ti_i].amount {
				integer, _ := value_to_integer(ival)
				addf(&nasm.text_sec, "\tmov %v [rsp + %v], %v\n", index_word, stack_pos + i*type_align, integer)
				ival.pos += type_align
			}
		case TYPE_INDIRECT: /* already dealt with before this switch */
		case TYPE_RT_ARRAY:
			/* put stuff in data section, which itself might need initialization, so, yipiee */
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

			addf(&nasm.text_sec, "\tmov %v [rsp + %v], %v + %v\n", indexing_word(align_of_type(ti)), stack_pos, label_name, ledger[1].offset)
			addf(&nasm.text_sec, "\tmov %v [rsp + %v], %v\n", indexing_word(8), stack_pos + 8, ledger[1].element_amount)
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
