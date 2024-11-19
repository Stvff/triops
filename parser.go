package main
import "fmt"

func resolve_decl_value(set *Token_Set, scope *Scope, ti Type_Index) (value Value, exists bool) {
	old_index := set.index
	/* Check if this is an enum */
	var enum_typ Type_Index
	value, enum_typ, exists = resolve_enum_value(set, scope)
	if exists {
		if !are_types_equal(ti, enum_typ) {
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
			/* TODO: this has to account for the different forms of a bare type!! */
			var integer int
			integer, exists = resolve_integer(set, scope)
			return integer_to_value(integer), exists
		case TYPE_INDIRECT:
			return resolve_decl_value(set, scope, follow_type(ti))
		case TYPE_RT_ARRAY, TYPE_ST_ARRAY:
			/* Of arrays, there are two cases: it is an array with curly braces, or it is a string.
			   There are some wild card possibilities, but for now, let us just deal with these two cases.
			   In the case of a string, the value form of the type needs to be checked if it even accepts strings */
			nested_ti := follow_type(ti)
			/* the stringform case */
			if si, st := unpack_ti(nested_ti); st == TYPE_BARE {
				value, exists = resolve_string_value(set, scope)
				if exists && bare_types[si].form != VALUE_FORM_WILD && bare_types[si].form != VALUE_FORM_STRING {
					print_error_line("This variable/constant does not allow string assignment", set)
					return value, false
				} else if exists {
					return value, true
				}
			}
			/* the curly braces array case */
			for curr(set).tag != KEYWORD_CLOSE_BRACE && !set.end {
				inc(set)
				_, exists = resolve_decl_value(set, scope, nested_ti)
				if !exists {
					print_error_line("errorloop", set)
					return value, false
				}
				inc(set)
			}
		case TYPE_POINTER:
			/* TODO: Add typechecking here, as well as some sort of indication of where to find which variable is being pointed to,
			   as the actual pointer is naturally a runtime variable, and we have to communicate that at compiletime to the backend somehow */
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

func resolve_enum_value(set *Token_Set, scope *Scope) (value Value, typ Type_Index, exists bool) {
	if curr(set).tag != NONE {
		return value, typ, false
	}
	enum_name := token_str(set)
	var (
		enum Enum_Des
		evid Enum_Value_ID
	)
	enum, exists = scope.enums[enum_name]
	if !exists {
		return value, typ, false
	}
	typ = enum.typ
	evid.parent_id = enum.id
	evid.name = "#"
	value, exists = enum_values[evid]
	if exists {
		return value, typ, true
	}
	inc(set)
	if curr(set).tag != KEYWORD_DOT {
		print_error_line("This enum is not a single constant, a subname needs to be specified", set)
		return value, typ, false
	}
	inc(set)
	if curr(set).tag != NONE {
		print_error_line("Invalid enum subname", set)
		return value, typ, false
	}
	evid.name = token_str(set)
	value, exists = enum_values[evid]
	if !exists {
		print_error_line("Enum subvalue does not exist", set)
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
