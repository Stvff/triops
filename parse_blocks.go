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
	prev_error_count := error_count
	parsing_args := false
	argument_body_supplied := false
	var mnm_index int
	var instruction_alignment int
	statloop: for ; !set.end ; inc(set) {
		token := curr(set)
		if prev_error_count != error_count {
			parsing_args = false
			argument_body_supplied = false
			prev_error_count = error_count
		}
		switch token.tag {
		case KEYWORD_SEMICOLON:
			parsing_args = false
			argument_body_supplied = false
			if single_statement {break statloop} else {continue statloop}
		case KEYWORD_CLOSE_BRACE:
			break statloop
		}

		if !parsing_args {
			var mnm_node Node
			mnm_node.kind = NKIND_MNEMONIC
			switch token.tag {
			case DIRECTIVE_LBL:
				inc(set)
				if !validate_name(set, scope) {
					error_count += 1
					skip_statement(set)
					if single_statement { break statloop } else { continue statloop }
				}
				mnm_node.token = curr(set)
				mnm_node.kind = NKIND_LABEL
				mnm_index = append_node(mnm_node)
				add_label_to_scope(scope, token_str(set), mnm_index)
			default:
				mnm_node.token = token
				parsing_args = true
				mnm_index = append_node(mnm_node)
			}
			instruction_alignment = 0
	
			/* link to previous mnemonic */
			var mnm_link Link;
			mnm_link.kind = LKIND_SEMICOLON
			mnm_link.right = mnm_index
			for i := len(scope.code)-1; i > 0; i -= 1 {
				link := scope.code[i]
				if link.kind == LKIND_SEMICOLON {
					mnm_link.left = link.right
					break
				}
			}
			if mnm_link.kind == LKIND_NONE && len(scope.code) > 1 {
				/* this should only really happen if this is the second toplevel token */
				mnm_link.left = 0
			}
			add_link(scope, mnm_link)
			if single_statement {break statloop} else {continue statloop}
		}

		/* loop for arguments until a semicolon */
		token = curr(set)
		var node Node
		node.token = token
		switch token.tag {
		/* indexing */
		case KEYWORD_OPEN_BRACKET:
			if !argument_body_supplied {
				error_count += 1
				print_error_line_token_txt(prev(set), set.text, "There is no argument to index");
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			inc(set)
			offset, exists := resolve_integer(set, scope)
			if !exists {
				error_count += 1
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}

			node.kind = NKIND_INDEX
			node.token = curr(set)
			node.imm = integer_to_value(offset)

			previous_node := all_nodes[len(all_nodes)-1]
			origin, next := follow_indirection_type(previous_node.ti)
			_, origin_t := unpack_ti(origin)
			if !(previous_node.kind == NKIND_REGISTER && origin_t == TYPE_POINTER) &&
			   !(previous_node.kind == NKIND_VARIABLE || previous_node.kind == NKIND_INDEX) {
				error_count += 1
				print_error_line(set, "Only variables and pointer registers can be indexed")
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			wrong_type := false
			switch origin_t {
				case TYPE_RT_ARRAY: wrong_type = true
					print_error_line(set, "Dynamic arrays cannot be directly indexed in assembly blocks (since it would generate multiple instructions)")
				case TYPE_POINTER:
					if previous_node.kind != NKIND_REGISTER {
						wrong_type = true
						print_error_line(set, "Pointer variables cannot be directly indexed in assembly blocks (since it would generate multiple instructions)")
					}
				case TYPE_STRUCT: wrong_type = true
					print_error_line(set, "Structs cannot be directly indexed")
			}
			if offset >= amount_of_type(origin) || offset < 0 {
				wrong_type = true
				print_error_line(set, "Indexing out of bounds (type has %v columns, but the index was %v)", amount_of_type(origin), offset)
			}
			if wrong_type {
				error_count += 1
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			node.ti = next

			inc(set)
			if curr(set).tag != KEYWORD_CLOSE_BRACKET {
				error_count += 1
				print_error_line(set, "Missing closing bracket")
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			/* Stealing the link to the mnemonic */
			index := append_node(node)
			for i := len(scope.code)-1; i > 0; i -= 1 {
				l := scope.code[i]
				if l.kind == LKIND_RIGHT_ARG {
					scope.code[i].right = index
					break
				}
			}
			add_link(scope, Link{LKIND_INDEX, len(all_nodes)-2, index})

		/* comma's */
		case KEYWORD_COMMA:
			if !argument_body_supplied {
				error_count += 1
				print_error_line_token_txt(prev(set), set.text, "Missing argument");
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			for i := len(scope.code)-1; i > 0; i -= 1 {
				l := scope.code[i]
				if l.kind == LKIND_RIGHT_ARG {
					node = all_nodes[scope.code[i].right]
					break
				}
			}
			var align int
			if node.ti == 0 { switch node.kind {
				case NKIND_IMMEDIATE: /* do nothing */
				case NKIND_LABEL:
					align = 8
				case NKIND_REGISTER:
					align = node.satisfied_left
			}} else {
				align = align_of_type(node.ti)
			}
			
			if instruction_alignment == 0 {
				instruction_alignment = align
			} else if instruction_alignment != align {
				error_count += 1
				print_error_line_token_txt(node.token, set.text, "The alignment of this argument is not the same as the alignment of the rest of the arguments (it has %v bytes, while the rest has %v bytes)", align, instruction_alignment)
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			argument_body_supplied = false

		/* subscripting */
		case KEYWORD_DOT:
			if !argument_body_supplied {
				error_count += 1
				print_error_line_token_txt(prev(set), set.text, "There is no argument to subscript");
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			previous_node := all_nodes[len(all_nodes)-1]
			if previous_node.kind != NKIND_VARIABLE && previous_node.kind != NKIND_INDEX {
				error_count += 1
				print_error_line(set, "Expected a struct or dynamic array to subscript")
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			origin, _ := follow_indirection_type(previous_node.ti)
			_, origin_t := unpack_ti(origin)
			if origin_t != TYPE_RT_ARRAY {
				error_count += 1
				print_error_line(set, "Only dynamic arrays can be subscripted like this for now")
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			inc(set)
			node.kind = NKIND_INDEX
			node.token = curr(set)
			node.ti = origin
			subscription := token_txt_str(node.token, set.text)
			if subscription == "data" {
				node.imm = integer_to_value(0)
			} else if subscription == "count" {
				node.imm = integer_to_value(1)
			} else {
				error_count += 1
				print_error_line(set, "Unrecognized subscript")
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			/* Stealing the link to the mnomonic */
			index := append_node(node)
			for i := len(scope.code)-1; i > 0; i -= 1 {
				l := scope.code[i]
				if l.kind == LKIND_RIGHT_ARG {
					scope.code[i].right = index
					break
				}
			}
			add_link(scope, Link{LKIND_INDEX, len(all_nodes)-2, index})

		/* literals */
		case VALUE_INTEGER: /* TODO: deal with more than just integer literals */
			if argument_body_supplied {
				error_count += 1;
				print_error_line(set, "Missing comma")
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			argument_body_supplied = true
			all_nodes[mnm_index].satisfied_right += 1
			integer, exists := resolve_integer(set, scope)
			if !exists {
				error_count += 1
				print_error_line(set, "Malformed integer (somehow)")
				skip_statement(set)
				if single_statement { break statloop } else { continue statloop }
			}
			node.kind = NKIND_IMMEDIATE
			node.imm = integer_to_value(integer)
			add_link(scope, Link{LKIND_RIGHT_ARG, mnm_index, append_node(node)})

		/* Variables, registers, labels, or enums */
		case NONE:
			if argument_body_supplied {
				error_count += 1;
				print_error_line(set, "Missing comma")
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			argument_body_supplied = true
			all_nodes[mnm_index].satisfied_right += 1
			this := where_is(scope, token_str(set))
			switch this.named_thing {
			case NAME_DECL: /* variables */
				decl := all_decls[this.index]
				if decl.has_bound_register {
					node.kind = NKIND_REGISTER
					node.token = all_registers[decl.bound_register].token
				} else {
					node.kind = NKIND_VARIABLE
				}
				node.ti = decl.typ

			case NAME_REG: /* registers */
				register := all_registers[this.index]
				node.kind = NKIND_REGISTER
				node.ti = register.typ
				node.satisfied_left = register.size

			case NAME_ENUM: /* enums */
				value, enum_typ, vexists := resolve_enum_value(set, scope)
				if !vexists {
					error_count += 1
					skip_statement(set)
					if single_statement {break statloop} else {continue statloop}
				}
				node.kind = NKIND_IMMEDIATE
				node.imm = value
				node.ti = enum_typ

			case NAME_NOT_HERE: /* This is either an undefined name or a label */
				node.kind = NKIND_LABEL
				scope.label_uses = append(scope.label_uses, curr(set))
				
			default: /* Type or proc */
				error_count += 1
				print_error_line(set, "Types and Procedures can't be used in assembly blocks")
				skip_statement(set)
				if single_statement {break statloop} else {continue statloop}
			}
			add_link(scope, Link{LKIND_RIGHT_ARG, mnm_index, append_node(node)})
			
			default:
			error_count += 1
			print_error_line(set, "Unexpected token encountered")
			skip_statement(set)
			if single_statement {break statloop} else {continue statloop}
		} /* token.tag switch */
	} /* statement loop */

	if !single_statement {
		set.codebraces -= 1
	}

	return error_count
}
