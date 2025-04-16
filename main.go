package main

import (
	"fmt"
	"os"
)

func main() {
	file, err := os.ReadFile("stest.trs")
	if err != nil {
		fmt.Println(err)
		return
	}
	full_text := string(file)

	tokens := make([]Token, 0)
	var token Token
	for i := 0; i < len(full_text); i += 1 {
		char := rune(full_text[i])
//		fmt.Printf("%c", (char))
		/* this skips text */
		if is_white(char) {
			if token.len != 0 {
				token.tag = keywords[token_txt_str(token, full_text)]
				tokens = append(tokens, token)
				token = Token{0, 0, 0}
			}
			continue
		}
		if char == '#' && full_text[i+1] == '*' {
			depth := 0
			for ; i < len(full_text); i += 1 {
				if full_text[i] == '#' && full_text[i+1] == '*' {
					depth += 1; i += 1; continue
				}
				if full_text[i] == '*' && full_text[i+1] == '#' {
					depth -= 1; i += 1;
					if depth == 0 { break }
				}
			}
			continue
		}
		if char == '#' && full_text[i+1] == '#' {
			for ; i < len(full_text); i += 1 {
				if full_text[i] == '\n' { break }
			}
			continue
		}
		/* time for real tokens */
		if token.len == 0 {token.pos = uint32(i)}
		token.len += 1
		/* special cases */
		if char == '#' && is_alphabetic(rune(full_text[i+1])) { continue }
		if lang_operator(char) {
			if token.len > 1 { token.len -= 1; i -= 1 } 
			token.tag = keywords[token_txt_str(token, full_text)]
			tokens = append(tokens, token)
			token = Token{0, 0, 0}
			continue
		}
		if token.len == 1 && is_digit(char) {/* TODO: this basically only really handles integers */
			var is_float bool
			for is_digit(rune(full_text[i])) {
				token.len += 1; i += 1
				if token.len == 2 && full_text[i] == 'x' { token.len += 1; i += 1 }
				if full_text[i] == '.' { is_float = true; token.len += 1; i += 1 }
			}
			token.len -= 1; i -= 1
			if is_float { token.tag = VALUE_FLOAT
			} else { token.tag = VALUE_INTEGER }
			tokens = append(tokens, token)
			token = Token{0, 0, 0}
			continue
		}
		if stumble(char) {
			if token.len == 1{/* the same stumble character repeated will count as 1 token */
				for full_text[i] == byte(char) { token.len += 1; i += 1 }
			}
			token.len -= 1; i -= 1
			token.tag = keywords[token_txt_str(token, full_text)]
			tokens = append(tokens, token)
			token = Token{0, 0, 0}
			continue
		}
		if char == '"' {
			if token.len == 1 {
				token.len += 1; i += 1
				var prev_char rune
				for {
					if full_text[i] == '"' && prev_char != '\\' { break }
					if full_text[i] == '\\' && prev_char == '\\' { prev_char = ' '
					} else { prev_char = rune(full_text[i]) }
					token.len += 1; i += 1
				}
				token.tag = VALUE_STRING
			} else {
				token.tag = keywords[token_txt_str(token, full_text)]
			}
			tokens = append(tokens, token)
			token = Token{0, 0, 0}
			continue
		}
	}

	global_scope := Scope{
		types : make(map[string]Type_Index),
		enums : make(map[string]Enum_Des),
		decls : make(map[string]Decl_Des),
		assembly : Asm_Block{
			label_defs : make(map[string]int),
		},
	}
	set := Token_Set {
		index : 0,
		text : full_text,
		tokens : tokens,
	}
//	ggscope = &global_scope
	error_count := 0
	for ; !set.end ; inc(&set) {
		token := set.tokens[set.index]
//		fmt.Println(token_txt_str(token, full_text))
		switch token.tag {
		case KEYWORD_TYPE:
			inc(&set)
			if !parse_type_decl(&set, &global_scope) { error_count += 1 }
		case KEYWORD_ENUM:
			inc(&set)
			if !parse_enum_decl(&set, &global_scope) { error_count += 1 }
		case KEYWORD_SEMICOLON:
		case NONE:
			old_index := set.index
			if !parse_variable_decl(&set, &global_scope) { error_count += 1 }
			if set.index == old_index {
				print_error_line("Unknown type for global declaration", &set)
				error_count += 1
//				print_error_line("Runtime expressions are not allowed in the global scope (`entry` would be the place for that)", tokens[i], &global_scope)
			}
		case KEYWORD_ASM: fallthrough
		case KEYWORD_ENTRY:
			if !parse_asm(&set, &global_scope) { error_count += 1 }
		default:
			print_error_line("Unexpected toplevel token", &set)
			skip_statement(&set)
			error_count += 1
		}
	}
/*	fmt.Println()
	for _, typ := range bare_types { fmt.Println(typ) }
	for _, typ := range indirect_types { fmt.Println(typ) }
	fmt.Println()
	for name, enum := range global_scope.enums { fmt.Println(name, enum) }
	for name, value := range enum_values { fmt.Println(name, value) }
	fmt.Println()
	for name, decl := range global_scope.decls { fmt.Println(name, decl) }
	fmt.Println(all_values)
*/
	if error_count == 0 {
		full_asm, _ := generate_assembly(&global_scope, &set)
		fmt.Println(full_asm)
		err = os.WriteFile("asm/test.nasm", []byte(full_asm), 0666)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		fmt.Printf("Amount of errors: %v\n", error_count)
	}
}

//var ggscope *Scope

func is_white(char rune) bool {
	switch char {
		case ' ', '\t', '\n', '\r', '\v', '\f': return true
		default: return false
	}
}

func stumble(char rune) bool {
	switch char {
		case '+', '-', '/', '\\', '*',
		     '%', '$', '@', '!', '|',
		     '~', '>', '<': return true
		default: return false
	}
}

func lang_operator(char rune) bool {
	switch char {
		case ';', ',', '^', '[', ']',
		     '(', ')', '{', '}', '=',
		     '\'', '#', '.': return true
		default: return false
	}
}

func is_digit(char rune) bool {
	return char >= '0' && char <= '9'
}

func is_alphabetic(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char == '_')
}

func curr(set *Token_Set) Token {
	return set.tokens[set.index]
}

func inc(set *Token_Set) *Token_Set {
	if set.index == len(set.tokens) - 1 { set.end = true; return set }
	set.index += 1
	if t := set.tokens[set.index].tag; t == KEYWORD_OPEN_BRACE { set.braces += 1
	} else if t == KEYWORD_CLOSE_BRACE { set.braces -= 1 }
//	fmt.Print("\"", token_str(set), "\"i", set.braces, ", ")
	return set
}

func dec(set *Token_Set) *Token_Set {
	if t := set.tokens[set.index].tag; t == KEYWORD_OPEN_BRACE { set.braces -= 1
	} else if t == KEYWORD_CLOSE_BRACE { set.braces += 1 }
	set.index = max(set.index - 1, 0)
//	fmt.Print("\"", token_str(set), "\"d", set.braces, ", ")
	return set
}
