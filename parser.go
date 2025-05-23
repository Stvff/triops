package main

func resolve_decl_value(set *Token_Set, scope *Scope, ti Type_Index) (value Value, exists bool) {
	old_index := set.index
	/* Check if this is an enum */
	var enum_typ Type_Index
	value, enum_typ, exists = resolve_enum_value(set, scope)
	if exists {
		if !are_types_equal(ti, enum_typ) {
			/* TODO: print definitions */
			print_error_line(set, "Type mismatch between declared variable and enum value")
			return value, false
		}
		return value, true
	} else if old_index != set.index {
		set.index = old_index
		return value, false
	}

	tag_associated_index, tag := unpack_ti(ti)
	value = Value{form : VALUE_FORM_NONE, pos : value_head}
	tagswitch: switch tag {
		case TYPE_ERR: print_error_line(set, "resolve_decl_value: internal type error")
		case TYPE_BARE:
			/* TODO: this has to account for the different forms of a bare type, not just ints (this lets through incorrect assignments */
			var integer int
			integer, exists = resolve_integer(set, scope)
			value = integer_to_sized_value(integer, size_of_type(ti))
			return value, exists
		case TYPE_INDIRECT:
			return resolve_decl_value(set, scope, follow_type(ti))
		case TYPE_RT_ARRAY, TYPE_ST_ARRAY:
			/* Of arrays, there are two cases: it is an array with curly braces, or it is a string.
			   There are some wild card possibilities, but for now, let us just deal with these two cases.
			   In the case of a string, the value form of the type needs to be checked if it even accepts strings */
			if tag == TYPE_RT_ARRAY {
				/* This holds the size of the array, which later has to be filled in,
				   and then later set to zero when we get to the assembly stage, because this region
				   also holds the size of the allocated buffer (which is zero when in section .data) */
				make_value(8)
			}
			nested_ti := follow_type(ti)
			/* the stringform case */
			if si, st := unpack_ti(nested_ti); st == TYPE_BARE {
				var tvalue Value
				tvalue, exists = resolve_string_value(set, scope)
				if exists && bare_types[si].form != VALUE_FORM_WILD && bare_types[si].form != VALUE_FORM_STRING {
					print_error_line(set, "This variable/constant does not allow string assignment")
					return value, false
				} else if exists {
					if tag == TYPE_RT_ARRAY {
						/* TODO: this and the else clause should take into account types that are
						   larger than bytes, and pad or error if it's mismatched */
						integer_at_value(value, tvalue.len)
					} else if st_array_types[tag_associated_index].size != tvalue.len {
						print_error_line(set, "Given string was larger than the size of the array")
						return value, false
					}
					break tagswitch
				}
			}
			/* the curly braces array case */
			array_len := 0
			for curr(set).tag != KEYWORD_CLOSE_BRACE && !set.end {
				inc(set)
				old_index = set.index
				_, exists = resolve_decl_value(set, scope, nested_ti)
				/* TODO: Handle extra comma's more gracefully */
				if !exists {
					if old_index == set.index { print_error_line(set, "Value in array is malformed") }
					dec(set)
					return value, false
				}
				inc(set)
				if curr(set).tag != KEYWORD_COMMA && curr(set).tag != KEYWORD_CLOSE_BRACE{
					print_error_line(set, "Unexpected token in array literal")
					return value, false
				}
				array_len += 1
				if tag == TYPE_ST_ARRAY && st_array_types[tag_associated_index].size < array_len {
					print_error_line(set, "Too many values in static array literal (expected %v values, got %v)", st_array_types[tag_associated_index].size, array_len)
					return value, false
				}
			}
			if tag == TYPE_ST_ARRAY && st_array_types[tag_associated_index].size > array_len {
				print_error_line(set, "Too few values in static array literal (expected %v values, got %v)", st_array_types[tag_associated_index].size, array_len)
				return value, false
			}
			if tag == TYPE_RT_ARRAY {
				integer_at_value(value, array_len)
			}
		case TYPE_POINTER:
			/* TODO: Add typechecking here, as well as some sort of indication of where to find which variable is being pointed to,
			   as the actual pointer is naturally a runtime variable, and we have to communicate that at compiletime to the backend somehow */
			print_error_line(set, "resolve_decl_value: pointers are unfinished")
			return value, false
		case TYPE_STRUCT:
			print_error_line(set, "resolve_decl_value: structs are unfinished")
	}
	value.len = value_head - value.pos
	return value, true
}

func parse_type(set *Token_Set, scope *Scope) (ti Type_Index, exists bool) {
	this := where_is(scope, token_str(set))
	exists = this.named_thing == NAME_TYPE
	if !exists {
		return ti, exists
	}
	ti = all_types[this.index]
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
			print_error_line(set, "Value for array size needs to be a positive integer")
			return ti, false
		}
		inc(set)
		if curr(set).tag != KEYWORD_CLOSE_BRACKET {
			/* TODO: print where the open bracket was? */
			print_error_line(set, "Array size bracket not closed")
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

func resolve_enum_value(set *Token_Set, scope *Scope) (value Value, typ Type_Index, exists bool) {
	if curr(set).tag != NONE {
		return value, typ, false
	}
	enum_name := token_str(set)
	this := where_is(scope, enum_name)
	exists = this.named_thing == NAME_ENUM
	if !exists {
		return value, typ, false
	}
	enum := all_enums[this.index]
	typ = enum.typ
	var evid Enum_Value_ID
	evid.parent_id = enum.id
	evid.name = "#"
	value, exists = enum_values[evid]
	if exists {
		return value, typ, true
	}
	inc(set)
	if curr(set).tag != KEYWORD_DOT {
		print_error_line(set, "This enum is not a single constant, a subname needs to be specified")
		return value, typ, false
	}
	inc(set)
	if curr(set).tag != NONE {
		print_error_line(set, "Invalid enum subname")
		return value, typ, false
	}
	evid.name = token_str(set)
	value, exists = enum_values[evid]
	if !exists {
		print_error_line(set, "Enum subvalue does not exist")
		return value, typ, false
	}
	return value, typ, true
}

func resolve_integer(set *Token_Set, scope *Scope) (v int, exists bool) {
	sign := 1
	s := token_str(set)
	if s[0] == '-' { sign = -1; inc(set) }
	if curr(set).tag == NONE {
		var value Value
		old_index := set.index
		value, _, exists = resolve_enum_value(set, scope)
		if !exists {
			if old_index != set.index { set.index = old_index; return v, false }
			print_error_line(set, "Enum does not exist")
			return v, false
		}
		v, exists = value_to_integer(value)
		if !exists {
			print_error_line(set, "Enum value was not an integer")
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

func resolve_string_value(set *Token_Set, scope *Scope) (value Value, exists bool) {
	value = Value{form : VALUE_FORM_WILD, pos : value_head}
	if curr(set).tag != VALUE_STRING {
		return value, false
	}
	value.form = VALUE_FORM_STRING
	str := token_str(set)
	prev_char := byte(' ')
	for i := 1; i < len(str)-1; i += 1 {
		char := str[i]
		if prev_char == '\\' {
			switch char {
				case 'n': all_values[value_head-1] = '\n'
				case '\\': all_values[value_head-1] = '\\'
				case '"': all_values[value_head-1] = '"'
			}
			prev_char = ' '
			continue
		}
		all_values = append(all_values, char)
		value_head += 1
		prev_char = char
	}
	value.len = value_head - value.pos
	return value, true
}

func validate_name(set *Token_Set, scope *Scope) bool {
	name_str := token_str(set)
	if curr(set).tag != NONE {
		print_error_line(set, "Name must not be a number, string or language keyword")
		return false
	}
	this := where_is(scope, name_str);
	if this.named_thing != NAME_NOT_HERE {
		/* TODO: print where the it was defined */
		things_it_could_be_defined_as := []string{"a type", "an enum", "a variable", "a label", "a procedure"}
		print_error_line(set, "Name was already declared as %s", things_it_could_be_defined_as[this.named_thing-NAME_TYPE])
		return false
	}
	return true
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
	brace_detected := set.braces > set.codebraces
	skipping: for !set.end {
		tagswitch: switch curr(set).tag {
		case KEYWORD_CLOSE_PAREN: fallthrough
		case KEYWORD_COMMA:
			if !set.commas_and_parens_as_semis { break tagswitch }
			fallthrough
		case KEYWORD_SEMICOLON: if set.braces == set.codebraces { break skipping }
		case KEYWORD_OPEN_BRACE: brace_detected = true
		case KEYWORD_CLOSE_BRACE:
		}
		if brace_detected && set.braces == set.codebraces { break skipping }
		inc(set)
	}
	if set.index < len(set.tokens)-1 && set.tokens[set.index + 1].tag == KEYWORD_SEMICOLON { inc(set) }
//	fmt.Println("done skipping")
	return false
}

func finish_statement(set *Token_Set) bool {
	if set.commas_and_parens_as_semis {
		if curr(set).tag == KEYWORD_COMMA || curr(set).tag == KEYWORD_CLOSE_PAREN { return true }
		print_error_line(dec(set), "Comma or closing parenthesis missing")
		return skip_statement(inc(set))
	} else {
		if curr(set).tag == KEYWORD_SEMICOLON { return true }
		print_error_line(dec(set), "Statement must be ended with a semicolon")
		return skip_statement(inc(set))
	}
}
