package main

func parse_proc_decl(set *Token_Set, scope *Scope) bool {
	set.commas_and_parens_as_semis = true
	parse_variable_decl(set, scope)
	inc(set)
	inc(set)
	parse_variable_decl(set, scope)
	inc(set)
	inc(set)
	inc(set)
	parse_variable_decl(set, scope)
	inc(set)

	set.commas_and_parens_as_semis = false
	// parse_proc_block(set, scope)
	return finish_statement(set)
}

func parse_variable_decl(set *Token_Set, scope *Scope) bool {
	var new_decl Decl_Des
	/* checking type */
	// print_error_line("Decl?", set)
	_, exists := scope.types[token_str(set)]
	if !exists {
		return false
	}
	new_decl.typ, exists = parse_type(set, scope)
	if !exists {
		return skip_statement(set)
	}

	/* getting the name */
	if !validate_name(set, scope) { return skip_statement(set) }
	name := token_str(set)
	inc(set)

	/* checking if there's an init constant */
	if curr(set).tag != KEYWORD_EQUALS {
		scope.decls[name] = new_decl
		return finish_statement(set)
	}
	inc(set)
	
	/* getting the init constant */
	new_decl.init, exists = resolve_decl_value(set, scope, new_decl.typ)
	inc(set)
	if !exists {
		return skip_statement(set)
	}
	scope.decls[name] = new_decl

	return finish_statement(set)
}

func parse_type_decl(set *Token_Set, scope *Scope) bool {
	/* getting the name */
	if !validate_name(set, scope) { return skip_statement(set) }
	name := token_str(set)
	inc(set)
	if curr(set).tag != KEYWORD_IS {
		print_error_line("The word 'is' was expected", set)
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
				print_error_line("Internal unknown type error", set)
				return skip_statement(set)
			case TYPE_BARE:
				new_type = bare_types[i]
				new_type.name = name
				scope.types[new_type.name] = append_bare_type(new_type)
			case TYPE_INDIRECT:
				indirect_type := indirect_types[i]
				i_indir, tag_indir := unpack_ti(indirect_type.target)
				if tag_indir == TYPE_BARE {
					new_type = bare_types[i_indir]
					new_type.name = name
					scope.types[new_type.name] = append_bare_type(new_type)
					return finish_statement(set)
				}
				indirect_type.name = name
				scope.types[indirect_type.name] = append_indirect_type(indirect_type)
			default:
				scope.types[name] = append_indirect_type(Type_Des_Indirect{name, ti})
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
			print_error_line("Value or type was not defined", set)
		} else {
			print_error_line("Type declarations expect the alignment and column size to be positive integers", set)
		}
		return skip_statement(set)
	}
	inc(set)

	alignment_set := false
	if curr(set).tag == KEYWORD_COLUMNS {
		new_type.amount = integer
	} else if curr(set).tag == KEYWORD_BYTES {
		if integer != 1 && integer != 2 && integer != 4 && integer != 8 && integer != 16 {
			print_error_line("Alignment can only be a power of two, up to 16", set)
			return skip_statement(set)
		}
		new_type.alignment = integer
		alignment_set = true
	} else {
		print_error_line("Expected either the keyword `bytes` or `columns`", set)
		return skip_statement(set)
	}
	inc(set)

	/* deciding if that was all or not */
	if curr(set).tag == KEYWORD_SEMICOLON {
		scope.types[new_type.name] = append_bare_type(new_type)
		return true
	}

	if alignment_set != true && curr(set).tag == KEYWORD_OF {
		inc(set)

		/* getting the alignment value */
		integer, exists = resolve_integer(set, scope)
		if !exists {
			if curr(set).tag == NONE {
				print_error_line("Value or type was not defined", set)
			} else {
				print_error_line("Type declarations expect the alignment column size to be a positive integer", set)
			}
			return skip_statement(set)
		}
		inc(set)
		
		if curr(set).tag != KEYWORD_BYTES {
			print_error_line("Expected either the keyword `bytes`", set)
			return skip_statement(set)
		}
		if integer != 1 && integer != 2 && integer != 4 && integer != 8 && integer != 16 {
			print_error_line("Alignment can only be a power of two, up to 16", set)
			return skip_statement(set)
		}
		new_type.alignment = integer
		inc(set)
	}

	/* deciding if that was all or not */
	if curr(set).tag == KEYWORD_SEMICOLON {
		scope.types[new_type.name] = append_bare_type(new_type)
		return true
	}

	if t := curr(set).tag; t >= DIRECTIVE_INTFORM && t <= DIRECTIVE_BYTEFORM {
		new_type.form = VALUE_FORM_INTEGER + (Value_Form(t) - DIRECTIVE_INTFORM)
	} else {
		print_error_line("Invalid typeform directive", set)
		return skip_statement(set)
	}
	inc(set)

	scope.types[new_type.name] = append_bare_type(new_type)
	return finish_statement(set)
}

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
		print_error_line("Given type does not exist", set)
		return skip_statement(set)
	}

	/* getting the name */
	if !validate_name(set, scope) { skip_statement(set); return false }
	name := token_str(set)
	inc(set)

	/* registering the enum */
	scope.enums[name] = Enum_Des{typ, global_enum_id}
	evid.parent_id = global_enum_id
	global_enum_id += 1

	/* single constant case */
	if curr(set).tag == KEYWORD_EQUALS {
		inc(set)
		integer, exists = resolve_integer(set, scope)
		if !exists {
			print_error_line("Invalid integer", set)
			return skip_statement(set)
		}
		value = integer_to_sized_value(integer, size_of_type(typ))
		evid.name = "#"
		enum_values[evid] = value
		return finish_statement(inc(set))
	}
	/* normal block of names case */
	if curr(set).tag != KEYWORD_OPEN_BRACE {
		print_error_line("Enum must have a block of names (and optional values)", set)
		return skip_statement(set)
	}
	inc(set)

	/* enum values */
	for curr(set).tag != KEYWORD_CLOSE_BRACE && !set.end {
		//print_error_line("test", tokens[index], scope)
		if curr(set).tag != NONE {
			print_error_line("Names must not be reserved keywords or values", set)
			return skip_statement(set)
		}
		/* getting value name */
		evid.name = token_str(set)
		_, exists = enum_values[evid]
		if exists {
			print_error_line("Name is already in use in this enum", set)
			return skip_statement(set)
		}
		inc(set)
		/* finding value here */
		if curr(set).tag == KEYWORD_EQUALS {
			inc(set)
			integer, exists = resolve_integer(set, scope)
			if !exists {
				print_error_line("Invalid integer", set)
				return skip_statement(set)
			}
			inc(set)
			value = integer_to_sized_value(integer, size_of_type(typ))
			enum_values[evid] = value
		} else {
			value = increment_value(value)
			enum_values[evid] = value
		}

		if curr(set).tag != KEYWORD_COMMA && curr(set).tag != KEYWORD_CLOSE_BRACE {
			print_error_line("Expected a comma or closing brace", set)
			return skip_statement(set)
		}
		if curr(set).tag == KEYWORD_COMMA { inc(set) }
	}
	return finish_statement(inc(set))
}
