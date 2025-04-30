package main

func parse_asm(set *Token_Set, scope *Scope) int {
	single_statement := false
	if curr(set).tag == KEYWORD_SEMICOLON {
		return 0
	} else if curr(set).tag != KEYWORD_OPEN_BRACE {
		single_statement = true
	} else {
		set.codebraces += 1
		inc(set)
	}
	error_count := 0

	statloop: for ; !set.end ; inc(set) {
		//print_error_line(set, "where are we")
		token := curr(set)

		/* special cases */
		switch token.tag {
		case DIRECTIVE_LBL:
			inc(set)
			if !validate_name(set, scope) {
				error_count += 1
				skip_statement(set)
				if single_statement { break statloop } else { continue statloop }
			}
			var instruction Asm_Instruction
			instruction.mnemonic = prev(set);
			instruction.alignment = 8;
			instruction.args[0].verbatim = curr(set);
			add_label_to_scope(scope, token_str(set), len(scope.assembly.instructions))
			scope.assembly.instructions = append(scope.assembly.instructions, instruction)

			if !finish_statement(inc(set)) { error_count += 1 }
			if single_statement { break statloop } else { continue statloop }
		case DIRECTIVE_REG:
			inc(set)
			this := where_is(scope, token_str(set))
			if this.named_thing != NAME_DECL {
				print_error_line(set, "Expected a valid variable to bind a register to");
				error_count += 1
				skip_statement(set)
				if single_statement { break statloop } else { continue statloop }
			}
			var instruction Asm_Instruction
			instruction.mnemonic = prev(set)

			associated_decl := all_decls[this.index]
			if size_of_type(associated_decl.typ) > 8 {
				print_error_line(set, "Total size of the variable to be bound must be less than 8 bytes (was %v)", size_of_type(associated_decl.typ))
				error_count += 1;
				skip_statement(set)
				if single_statement { break statloop } else { continue statloop }
			}
			instruction.args[0].verbatim = curr(set)

			inc(set)
			if curr(set).tag != KEYWORD_EQUALS {
				print_error_line(set, "Expected an `=`")
				error_count += 1;
				skip_statement(set)
				if single_statement { break statloop } else { continue statloop }
			}
			inc(set)
			all_decls[this.index].bound_register = curr(set)
			instruction.args[1].verbatim = curr(set)
			scope.assembly.instructions = append(scope.assembly.instructions, instruction)
			
			if !finish_statement(inc(set)) { error_count += 1 }
			if single_statement { break statloop } else { continue statloop }
		case KEYWORD_CLOSE_BRACE: break statloop
		}
				

		/* produce instruction */
		var instruction Asm_Instruction
		/* TODO: check more thoroughly if the mnemonic and registers, labels etc are usable names */
		instruction.mnemonic = curr(set)
		inc(set)
		arg_nr := 0
		alignment_of_instruction := 0
		for ; !set.end ; inc(set) { /* argument loop */
			if curr(set).tag == KEYWORD_SEMICOLON { break }
//			print_error_line("prog", set)
			token = curr(set)
			switch token.tag {

			/* indexing */
			case KEYWORD_OPEN_BRACKET:
				inc(set)
				offset, exists := resolve_integer(set, scope)
				if !exists { error_count += 1; if single_statement { break statloop } else { continue statloop } }
				instruction.args[arg_nr].immediate = integer_to_value(offset)
				inc(set)
				if curr(set).tag != KEYWORD_CLOSE_BRACKET {
					error_count += 1
					print_error_line(set, "Missing closing bracket")
					skip_statement(set)
					if single_statement { break statloop } else { continue statloop }
				}

			/* a label as argument */
			case DIRECTIVE_LBL:
				inc(set)
				if arg_nr == 0 {
					alignment_of_instruction = 8
				} else if alignment_of_instruction != 8 {
					print_error_line(set, "A label has an alignment of 8, which is not the same as the alignment used in this instruction (%v bytes)", alignment_of_instruction);
					skip_statement(set)
					if single_statement { break statloop } else { continue statloop }
				}
				instruction.args[arg_nr].verbatim = curr(set)
				instruction.args[arg_nr].verbatim.tag = DIRECTIVE_LBL
				scope.label_uses = append(scope.label_uses, curr(set))

			/* a register as argument */
			case DIRECTIVE_REG_BYTE, DIRECTIVE_REG_WORD, DIRECTIVE_REG_DOUB, DIRECTIVE_REG_QUAD, DIRECTIVE_REG_OCTO:
				reg_alignments := [5]int{1, 2, 4, 8, 16}
				reg_nr := token.tag - DIRECTIVE_REG_BYTE
				reg_alignment := reg_alignments[reg_nr]
				if arg_nr == 0 {
					alignment_of_instruction = reg_alignment
				} else if alignment_of_instruction != reg_alignment {
					error_count += 1
					print_error_line(set, "This register does not have the alignment expected in its context, It has %v bytes, while %v bytes was expected", reg_alignment, alignment_of_instruction)
					skip_statement(set)
					if single_statement { break statloop } else { continue statloop }
				}
				inc(set)
				if curr(set).tag != NONE {
					error_count += 1
					print_error_line(set, "Register must not be a keyword or value")
					skip_statement(set)
					if single_statement { break statloop } else { continue statloop }
				}
				instruction.args[arg_nr].verbatim = curr(set)
				instruction.args[arg_nr].verbatim.tag = token.tag

			/* comma */
			case KEYWORD_COMMA:
				arg_nr += 1
				if arg_nr >= 3 {
					error_count += 1
					print_error_line(set, "The maximum amount of arguments for assembly instructions is 3")
					skip_statement(set)
					if single_statement { break statloop } else { continue statloop }
				}

			/* Values of some kind (only integers are supported rn) */
			case VALUE_INTEGER:
				integer, exists := resolve_integer(set, scope)
				if !exists {
					error_count += 1
					print_error_line(set, "Malformed integer (somehow)")
					skip_statement(set)
					if single_statement { break statloop } else { continue statloop }
				}
				instruction.args[arg_nr].immediate = integer_to_value(integer)

			/* Variable, (label?), or enum */
			case NONE:
				this := where_is(scope, token_str(set))
				switch this.named_thing {
					case NAME_DECL:
					decl := all_decls[this.index]
					if arg_nr == 0 {
						alignment_of_instruction = align_of_type(decl.typ)
					} else if align_of_type(decl.typ) != alignment_of_instruction {
						error_count += 1
						print_error_line(set, "This variable does not have the alignment expected in its context. It is aligned to %v bytes, while %v bytes was expected", align_of_type(decl.typ), alignment_of_instruction)
						skip_statement(set)
						if single_statement { break statloop } else { continue statloop }
					}
					/* TODO: Check for the case of .data and .count */
					/* TODO: Check we're not indexing an array */
					instruction.args[arg_nr].verbatim = curr(set)

					default:
					old_index := set.index
					value, enum_typ, vexists := resolve_enum_value(set, scope)
					if vexists && align_of_type(enum_typ) != alignment_of_instruction {
						error_count += 1
						print_error_line(set, "Enum value does not have the alignment expected in its context. It is aligned to %v bytes, while %v bytes was expected",  align_of_type(enum_typ), alignment_of_instruction)
						skip_statement(set)
						if single_statement { break statloop } else { continue statloop }
					} else if !vexists && old_index == set.index {
						error_count += 1
						print_error_line(set, "Enum or variable was not defined")
						skip_statement(set)
						if single_statement { break statloop } else { continue statloop }
					} else if !vexists {
						error_count += 1
						skip_statement(set)
						if single_statement { break statloop } else { continue statloop }
					}
					instruction.args[arg_nr].immediate = value
				}

			default:
				error_count += 1
				/* TODO: just unexpected token encountered?? */
				print_error_line(set, "Keyword, enum, variable or value was not defined or recognized")
				skip_statement(set)
				if single_statement { break statloop } else { continue statloop }
			} /* argument switchcase */

		} /* argument loop */

		instruction.alignment = alignment_of_instruction
		scope.assembly.instructions = append(scope.assembly.instructions, instruction)

		if single_statement { break statloop }
	} /* statement loop */

	if !single_statement {
		set.codebraces -= 1
	}

	return error_count
}
