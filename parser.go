package main
import "fmt"

func parse_variable_decl(set *Token_Set, scope *Scope) bool {
	var new_decl Decl_Des
	/* checking type */
	print_error_line("Decl?", set)
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

func resolve_decl_value(set *Token_Set, scope *Scope, ti Type_Index) (value Value, exists bool) {
	old_index := set.index
	value, exists = resolve_enum_value(set, scope)
	if exists {
		enum := scope.enums[token_str(set)]
		if !are_types_equal(ti, enum.typ) {
			print_error_line("Type mismatch between declared variable and enum value", set)
			return value, false
		}
		return value, true
	} else if old_index != set.index {
		return value, false
	}
	
	_, tag := unpack_ti(ti)
	value = Value{form : VALUE_FORM_NONE, pos : value_head}
	switch tag {
		case TYPE_ERR: print_error_line("resolve_decl_value: internal type error", set)
		case TYPE_BARE:
			var integer int
			integer, exists = resolve_integer(set, scope)
			return integer_to_value(integer), exists
		case TYPE_INDIRECT:
			return resolve_decl_value(set, scope, follow_type(ti))
		case TYPE_RT_ARRAY, TYPE_ST_ARRAY:
/*			if si, st := unpack_ti(follow_type(ti)); st == TYPE_BARE {
//				index, value, exists parse_string_value(
				if bare_types[si].form != VALUE_FORM_WILD && bare_types[si].form != VALUE_FORM_STRING {
					print_error_line("This variable/constant does not allow string assignment", set)
					return value, false
				}
			}*/
			for curr(set).tag != KEYWORD_CLOSE_BRACE && !set.end {
				inc(set)
				_, exists = resolve_decl_value(set, scope, follow_type(ti))
				if !exists {
					print_error_line("errorloop", set)
					return value, false
				}
				inc(set)
			}
		case TYPE_POINTER:
			
		case TYPE_STRUCT:
			print_error_line("resolve_decl_value: struct", set)
	}
	value.len = value_head - value.pos
	return value, true
}


func parse_type(set *Token_Set, scope *Scope) (ti Type_Index, exists bool) {
	ti, exists = scope.types[token_str(set)]
	if !exists {
		return ti, exists
	}
	inc(set)

	for curr(set).tag == KEYWORD_OPEN_BRACKET && !set.end{
		inc(set)
		/* [] */
		if curr(set).tag == KEYWORD_CLOSE_BRACKET {
			ti = append_rt_array_type(Type_Des_RT_Array{ti})
			inc(set)
			continue
		}

		size := 0
		size, exists = resolve_integer(set, scope)
 		if !exists || size < 0 {
			print_error_line("Value for array size needs to be a positive integer", set)
			return ti, false
		}
		inc(set)
		if curr(set).tag != KEYWORD_CLOSE_BRACKET {
			print_error_line("Array size bracket not closed", set)
			return ti, false
		}
		/* [0] */
		if size == 0 {
			ti = append_pointer_type(Type_Des_Pointer{ti})
			inc(set)
			continue
		}
		/* [42] */
		ti = append_st_array_type(Type_Des_St_Array{ti, size})
		inc(set)
	}
	return ti, true
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
		value int
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

	/* getting the alignment value */
	value, exists = resolve_integer(set, scope)
	if !exists {
		if curr(set).tag == NONE {
			print_error_line("Value or type was not defined", set)
		} else {
			print_error_line("Type declarations expect the alignment value to be a positive integer", set)
		}
		return skip_statement(set)
	}
	new_type.name = name
	new_type.amount = 1
	new_type.alignment = value
	new_type.form = VALUE_FORM_WILD
	inc(set)

	/* deciding if that was all or not */
	if curr(set).tag == KEYWORD_SEMICOLON {
		scope.types[new_type.name] = append_bare_type(new_type)
		return true
	}
	if curr(set).tag == KEYWORD_BYTES {
		if curr(inc(set)).tag == KEYWORD_SEMICOLON {
			scope.types[new_type.name] = append_bare_type(new_type)
			return true
		}

		t := curr(set).tag
		if t >= DIRECTIVE_INTFORM && t <= DIRECTIVE_BYTEFORM {
			new_type.form = VALUE_FORM_INTEGER + (Value_Form(t) - DIRECTIVE_INTFORM)
		} else {
			print_error_line("Invalid typeform directive", set)
			return skip_statement(set)
		}
		inc(set)
		scope.types[new_type.name] = append_bare_type(new_type)
		return finish_statement(set)
	}
	if curr(set).tag != KEYWORD_BY {
		print_error_line("The word 'by' or 'bytes' was expected", set)
		return skip_statement(set)
	}
	inc(set)

	/* getting the amount value */
	value, exists = resolve_integer(set, scope)
	if !exists {
		print_error_line("Type declarations expect the amount value to be a positive integer", set)
		return skip_statement(set)
	}
	new_type.amount = value
	scope.types[new_type.name] = append_bare_type(new_type)
	inc(set)

	/* finishing up */
	if curr(set).tag == KEYWORD_BYTES {
		return finish_statement(inc(set))
	}
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
		value = integer_to_value(integer)
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
			value = integer_to_value(integer)
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

func resolve_enum_value(set *Token_Set, scope *Scope) (value Value, exists bool) {
	if curr(set).tag != NONE {
		return value, false
	}
	enum_name := token_str(set)
	var (
		enum Enum_Des
		evid Enum_Value_ID
	)
	enum, exists = scope.enums[enum_name]
	if !exists {
		return value, false
	}
	evid.parent_id = enum.id
	evid.name = "#"
	value, exists = enum_values[evid]
	if exists {
		return value, true
	}
	inc(set)
	if curr(set).tag != KEYWORD_DOT {
		print_error_line("This enum is not a single constant, a subname needs to be specified", set)
		return value, false
	}
	inc(set)
	if curr(set).tag != NONE {
		print_error_line("Invalid enum subname", set)
		return value, false
	}
	evid.name = token_str(set)
	value, exists = enum_values[evid]
	if !exists {
		print_error_line("Enum subvalue does not exist", set)
		return value, false
	}
	return value, true
}

func resolve_integer(set *Token_Set, scope *Scope) (v int, exists bool) {
	sign := 1
	s := token_str(set)
	if s[0] == '-' { sign = -1; inc(set) }
	if curr(set).tag == NONE {
		var value Value
		old_index := set.index
		value, exists = resolve_enum_value(set, scope)
		if !exists {
			if old_index != set.index { set.index = old_index; return v, false }
			print_error_line("Enum does not exist", set)
			return v, false
		}
		v, exists = value_to_integer(value)
		if !exists {
			print_error_line("Enum value was not an integer", set)
			return v, false
		}
		return v, true
	}
	if curr(set).tag != VALUE_INTEGER {
		return 0, false
	}
	var integer int
	for _, v := range token_str(set) {
		integer *= 10
		integer += (int(v) - '0')
	}
	return sign*integer, true
}

func validate_name(set *Token_Set, scope *Scope) bool {
	name_str := token_str(set)
	if curr(set).tag != NONE {
		print_error_line("Name must not be a number, string or language keyword", set)
		return false
	}
	if _, exists := scope.decls[name_str]; exists {
		print_error_line("Name was already declared as variable", set)
		return false
	}
	if _, exists := scope.types[name_str]; exists {
		print_error_line("Name was already declared as type", set)
		return false
	}
	if _, exists := scope.enums[name_str]; exists {
		print_error_line("Name was already declared as enum", set)
		return false
	}
	return true
}

func print_error_line(message string, set *Token_Set) {
	var full_line Token
	token := curr(set)
	/* line_nr and start of the line */
	line_nr := 1
	for i := token.pos; i < token.pos + 1; i -= 1 {
		char := set.text[i]
		if line_nr == 1 && char == '\n' { full_line.pos = i + 1 }
		if char == '\n' { line_nr += 1 }
	}
	/* end of the line */
	for i := full_line.pos; i < uint32(len(set.text)); i += 1 {
		full_line.len += 1
		if set.text[i] == '\n' { break }
	}
	/* main print */
	full_line_str := token_txt_str(full_line, set.text)
	fmt.Printf("%s:\n", message)
	chars_written, _ := fmt.Printf("%d | ", line_nr)
	fmt.Printf("%s", full_line_str)
	/* the  c a r e t s */
	for i := 0; i < chars_written; i += 1 { fmt.Print(" ") }
	for i := uint32(0); i < token.pos - full_line.pos; i += 1 {
		if is_white(rune(full_line_str[i])) { fmt.Printf("%c", rune(full_line_str[i]))
		} else { fmt.Print(" ") }
	}
	for i := uint16(0); i < token.len; i += 1 { fmt.Print("^") }
	fmt.Println()
}

func token_str(set *Token_Set) string {
	token := set.tokens[set.index]
	return set.text[token.pos:token.pos+uint32(token.len)]
}

func token_txt_str(token Token, full_text string) string {
	return full_text[token.pos:token.pos+uint32(token.len)]
}

func skip_statement(set *Token_Set) bool {
//	fmt.Println("skip time")
	brace_detected := set.braces != 0
	skipping: for !set.end {
		switch curr(set).tag {
		case KEYWORD_SEMICOLON: if set.braces == 0 { break skipping }
		case KEYWORD_OPEN_BRACE: brace_detected = true
		case KEYWORD_CLOSE_BRACE:
		}
		if brace_detected && set.braces == 0 { break }
		inc(set)
	}
	if set.index < len(set.tokens)-1 && set.tokens[set.index + 1].tag == KEYWORD_SEMICOLON { inc(set) }
//	fmt.Println("done skipping")
	return false
}

func finish_statement(set *Token_Set) bool {
	if curr(set).tag == KEYWORD_SEMICOLON { return true }
	print_error_line("Statement must be ended with a semicolon", dec(set));
	return skip_statement(inc(set))
}
