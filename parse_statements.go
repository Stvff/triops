package main

func parse_proc_decl(set *Token_Set, scope *Scope) int {
	set.commas_and_parens_as_semis = true
	error_count := 0
	var proc Scope
	proc.prev_scope = scope

	/* checking for return variables */
	for curr(set).tag != KEYWORD_CLOSE_PAREN {
		parse_variable_decl(set, &proc, SPEC_OUTPUT)
		if curr(set).tag != KEYWORD_CLOSE_PAREN && curr(set).tag != KEYWORD_COMMA {
			print_error_line(set, "Expected a comma or closing parenthesis")
			skip_statement(set)
			return error_count + 1
		}
		if curr(set).tag == KEYWORD_COMMA { inc(set) }
	}
	inc(set)

	/* checking for left variables */
	if curr(set).tag == KEYWORD_OPEN_PAREN {
		inc(set)
		for curr(set).tag != KEYWORD_CLOSE_PAREN {
			parse_variable_decl(set, &proc, SPEC_LINPUT)
			if curr(set).tag != KEYWORD_CLOSE_PAREN && curr(set).tag != KEYWORD_COMMA {
				print_error_line(set, "Expected a comma or closing parenthesis")
				skip_statement(set)
				return error_count + 1
			}
			if curr(set).tag == KEYWORD_COMMA { inc(set) }
		}
		inc(set)
	}

	/* registering function name */
	if curr(set).tag != NONE {
		print_error_line(set, "Expected procedure name")
		skip_statement(set)
		return error_count + 1
	}
	if !validate_name(set, scope) {
		skip_statement(set)
		return error_count + 1
	}
	proc.definition_location = curr(set)
	inc(set)

	/* checking for right variables */
	if curr(set).tag == KEYWORD_OPEN_PAREN {
		inc(set)
		for curr(set).tag != KEYWORD_CLOSE_PAREN {
			parse_variable_decl(set, &proc, SPEC_RINPUT)
			if curr(set).tag != KEYWORD_CLOSE_PAREN && curr(set).tag != KEYWORD_COMMA {
				print_error_line(set, "Expected a comma or closing parenthesis")
				skip_statement(set)
				return error_count + 1
			}
			if curr(set).tag == KEYWORD_COMMA { inc(set) }
		}
		inc(set)
	}
	set.commas_and_parens_as_semis = false

	if curr(set).tag == KEYWORD_PREC {
		precedence, exists := resolve_integer(inc(set), scope)
		if !exists {
			print_error_line(set, "Expected an integer to set the precedence to")
			skip_statement(set)
			return error_count + 1
		}
		if precedence < -128 || precedence > 127 {
			print_error_line(set, "Precedence value should be between -128 and 127")
			skip_statement(set)
			return error_count + 1
		}
		proc.precedence = int8(precedence)
		inc(set)
	}

	if curr(set).tag == KEYWORD_ASM {
		proc.is_inline = true
		inc(set)
	}
	error_count += parse_asm(set, &proc)

	add_proc_to_scope(scope, token_txt_str(proc.definition_location, set.text), proc)

	//if !finish_statement(set) { error_count += 1 }
	return error_count
}

func parse_variable_decl(set *Token_Set, scope *Scope, specialty Decl_Specialty) bool {
	var new_decl Decl_Des
	var exists bool
	// print_error_line("Decl?", set)

	if curr(set).tag == KEYWORD_REGISTER {
		var register Reg_Des
		/* get register name */
		inc(set)
		if !validate_name(set, scope) { return skip_statement(set) }
		register.token = curr(set)
		if curr(inc(set)).tag != KEYWORD_IS {
			print_error_line(set, "The word `is` was expected")
			return skip_statement(set)
		}
		inc(set)

		/* checking type */
		this := where_is(scope, token_str(set))
		exists = this.named_thing == NAME_TYPE
		if !exists {
			/* in case of just a size */
			integer, exists := resolve_integer(set, scope)
			if !exists {
				return skip_statement(set)
			}
			if integer != 1 && integer != 2 && integer != 4 && integer != 8 && integer != 16 {
				print_error_line(set, "Register size can only be a power of two, up to 16")
				return skip_statement(set)
			}
			if curr(inc(set)).tag != KEYWORD_BYTES {
				print_error_line(set, "The word `bytes` was expected")
				return skip_statement(set)
			}
			inc(set)
			register.size = integer
			add_reg_to_scope(scope, token_txt_str(register.token, set.text), register, specialty)
			return finish_statement(set)
		}
		new_decl.typ, exists = parse_type(set, scope)
		if !exists {
			return skip_statement(set)
		}
		register.typ = new_decl.typ

		add_reg_to_scope(scope, token_txt_str(register.token, set.text), register, specialty)
		if curr(set).tag == KEYWORD_SEMICOLON {
			return true
		}
	} else {
		/* checking type */
		this := where_is(scope, token_str(set))
		register_size := 0
		register_type := Type_Index(0)
		if this.named_thing == NAME_REG {
			new_decl.bound_register = all_registers[this.index].token
			register_size = all_registers[this.index].size
			register_type = all_registers[this.index].typ
			inc(set)
		}

		this = where_is(scope, token_str(set))
		exists = this.named_thing == NAME_TYPE
		if register_type == 0 {
			if !exists {
				print_error_line(set, "Unexpected token while looking for a type")
				return skip_statement(set)
			}
			new_decl.typ, exists = parse_type(set, scope)
			if !exists {
				return skip_statement(set)
			}
			if register_size != 0 && register_size != size_of_type(new_decl.typ) {
				print_error_line(set, "The given register does not have the same size as the given type (%v bytes vs %v bytes)", register_size, size_of_type(new_decl.typ))
				return skip_statement(set)
			}
		} else {
			if exists {
				print_error_line(set, "Type was already given by register")
				return skip_statement(set)
			}
			new_decl.typ = register_type
		}
	}

	/* getting the name */
	if !validate_name(set, scope) { return skip_statement(set) }
	name := token_str(set)
	inc(set)

	/* checking if there's an init constant */
	if curr(set).tag != KEYWORD_EQUALS {
		add_decl_to_scope(scope, name, new_decl, specialty)
		return finish_statement(set)
	}
	inc(set)
	
	/* getting the init constant */
	new_decl.init, exists = resolve_decl_value(set, scope, new_decl.typ)
	inc(set)
	if !exists {
		return skip_statement(set)
	}
	add_decl_to_scope(scope, name, new_decl, specialty)

	return finish_statement(set)
}

func parse_type_decl(set *Token_Set, scope *Scope) bool {
	/* getting the name */
	if !validate_name(set, scope) { return skip_statement(set) }
	name := token_str(set)
	inc(set)
	if curr(set).tag != KEYWORD_IS {
		print_error_line(set, "The word `is` was expected")
		return skip_statement(set)
	}
	inc(set)

	var (
		ti Type_Index
		exists bool
		integer int
		new_type Type_Des_Bare
	)

	/* indirection types */
	if ti, exists = parse_type(set, scope); exists {
		i, tag := unpack_ti(ti)
		switch tag {
			case TYPE_ERR:
				print_error_line(set, "Internal unknown type error")
				return skip_statement(set)
			case TYPE_BARE:
				new_type = bare_types[i]
				new_type.name = name
				add_type_to_scope(scope, new_type.name, append_bare_type(new_type))
			case TYPE_INDIRECT:
				indirect_type := indirect_types[i]
				i_indir, tag_indir := unpack_ti(indirect_type.target)
				if tag_indir == TYPE_BARE {
					new_type = bare_types[i_indir]
					new_type.name = name
					add_type_to_scope(scope, new_type.name, append_bare_type(new_type))
					return finish_statement(set)
				}
				indirect_type.name = name
				add_type_to_scope(scope, indirect_type.name, append_indirect_type(indirect_type))
			default:
				add_type_to_scope(scope, name, append_indirect_type(Type_Des_Indirect{name, ti}))
		}
		return finish_statement(set)
	}


	new_type.name = name
	new_type.amount = 1
	new_type.alignment = 1
	new_type.form = VALUE_FORM_WILD

	/* getting the first value */
	integer, exists = resolve_integer(set, scope)
	if !exists {
		if curr(set).tag == NONE {
			print_error_line(set, "Value or type was not defined")
		} else {
			print_error_line(set, "Type declarations expect the alignment and column size to be positive integers")
		}
		return skip_statement(set)
	}
	inc(set)

	alignment_set := false
	if curr(set).tag == KEYWORD_COLUMNS {
		new_type.amount = integer
	} else if curr(set).tag == KEYWORD_BYTES {
		if integer != 1 && integer != 2 && integer != 4 && integer != 8 && integer != 16 {
			print_error_line(set, "Alignment can only be a power of two, up to 16")
			return skip_statement(set)
		}
		new_type.alignment = integer
		alignment_set = true
	} else {
		print_error_line(set, "Expected either the keyword `bytes` or `columns`")
		return skip_statement(set)
	}
	inc(set)

	/* deciding if that was all or not */
	if curr(set).tag == KEYWORD_SEMICOLON {
		add_type_to_scope(scope, new_type.name, append_bare_type(new_type))
		return true
	}

	if alignment_set != true && curr(set).tag == KEYWORD_OF {
		inc(set)

		/* getting the alignment value */
		integer, exists = resolve_integer(set, scope)
		if !exists {
			if curr(set).tag == NONE {
				print_error_line(set, "Value or type was not defined")
			} else {
				print_error_line(set, "Type declarations expect the alignment column size to be a positive integer")
			}
			return skip_statement(set)
		}
		inc(set)
		
		if curr(set).tag != KEYWORD_BYTES {
			print_error_line(set, "Expected the keyword `bytes`")
			return skip_statement(set)
		}
		if integer != 1 && integer != 2 && integer != 4 && integer != 8 && integer != 16 {
			print_error_line(set, "Alignment can only be a power of two, up to 16")
			return skip_statement(set)
		}
		new_type.alignment = integer
		inc(set)
	}

	/* deciding if that was all or not */
	if curr(set).tag == KEYWORD_SEMICOLON {
		add_type_to_scope(scope, new_type.name, append_bare_type(new_type))
		return true
	}

	if t := curr(set).tag; t >= DIRECTIVE_INTFORM && t <= DIRECTIVE_BYTEFORM {
		new_type.form = VALUE_FORM_INTEGER + (Value_Form(t) - DIRECTIVE_INTFORM)
	} else {
		print_error_line(set, "Expected a semicolon or typeform")
		return skip_statement(set)
	}
	inc(set)

	add_type_to_scope(scope, new_type.name, append_bare_type(new_type))
	return finish_statement(set)
}

/* TODO: make enums accept more than integers, any value than the type allows */
func parse_enum_decl(set *Token_Set, scope *Scope) bool {
	var (
		typ Type_Index
		evid Enum_Value_ID
		integer int
		value Value
		exists bool
	)

	/* getting the type */
	typ, exists = parse_type(set, scope)
	if !exists {
		print_error_line(set, "Given type does not exist")
		return skip_statement(set)
	}

	/* getting the name */
	if !validate_name(set, scope) { skip_statement(set); return false }
	name := token_str(set)
	inc(set)

	/* registering the enum */
	add_enum_to_scope(scope, name, Enum_Des{typ, global_enum_id})
	evid.parent_id = global_enum_id
	global_enum_id += 1

	/* single constant case */
	if curr(set).tag == KEYWORD_EQUALS {
		inc(set)
		integer, exists = resolve_integer(set, scope)
		if !exists {
			print_error_line(set, "Invalid integer")
			return skip_statement(set)
		}
		value = integer_to_sized_value(integer, size_of_type(typ))
		evid.name = "#"
		enum_values[evid] = value
		return finish_statement(inc(set))
	}
	/* normal block of names case */
	if curr(set).tag != KEYWORD_OPEN_BRACE {
		print_error_line(set, "Enum must have a block of names (and optional values) or be a single assignment")
		return skip_statement(set)
	}
	inc(set)

	/* enum values */
	for first := true; curr(set).tag != KEYWORD_CLOSE_BRACE && !set.end; first = false {
		//print_error_line("test", tokens[index], scope)
		if curr(set).tag != NONE {
			print_error_line(set, "Names must not be reserved keywords or values")
			return skip_statement(set)
		}
		/* getting value name */
		evid.name = token_str(set)
		_, exists = enum_values[evid]
		if exists {
			print_error_line(set, "Name is already in use in this enum")
			return skip_statement(set)
		}
		inc(set)
		/* finding value here */
		if curr(set).tag == KEYWORD_EQUALS {
			inc(set)
			integer, exists = resolve_integer(set, scope)
			if !exists {
				print_error_line(set, "Invalid integer")
				return skip_statement(set)
			}
			inc(set)
			value = integer_to_sized_value(integer, size_of_type(typ))
			enum_values[evid] = value
		} else {
			if first {
				value = integer_to_sized_value(0, size_of_type(typ))
			} else {
				value = increment_value(value)
			}
			enum_values[evid] = value
		}

		if curr(set).tag != KEYWORD_COMMA && curr(set).tag != KEYWORD_CLOSE_BRACE {
			print_error_line(set, "Expected a comma or closing brace")
			return skip_statement(set)
		}
		if curr(set).tag == KEYWORD_COMMA { inc(set) }
	}
	return finish_statement(inc(set))
}
