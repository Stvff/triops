package main

import (
	"fmt"
	"strings"
	"os"
	"os/exec"
)

const triops_help_message = `A lower level assembly macro programming language

        Usage:
        $ triops <file.trs> [options]                # Compile 'file.trs' to 'file'
        $ triops <file.trs> <output> [options]       # Compile 'file.trs' to 'ouput'
        $ triops <file.trs> <output.nasm> [options]  # Transpile 'file.trs' to 'ouput.nasm', doesn't generate an executable

        Options:
        -small  # Package the executable to be small
        -run    # Run the executable after compiling

`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(triops_help_message)
		os.Exit(1)
	}

	make_small_exe := false
	generate_executable := true
	output_provided := false
	run_executable := false

	input_filename := os.Args[1]
	output_filename := input_filename
	if len(os.Args) > 2 {
		for _, arg := range os.Args[2:] {
			if arg == "-small" {
				make_small_exe = true
				continue
			}
			if arg == "-run" {
				run_executable = true
				continue
			}
			if output_provided {
				fmt.Printf("Triops: An output file (`%v`) was already provided\n", output_filename)
				fmt.Print(triops_help_message)
				os.Exit(1)
			}
			output_filename = arg
			output_provided = true
		}
		l := len(output_filename)
		if l > 5 && output_filename[l - 5:] == ".nasm" {
			generate_executable = false
		}
	}

	if !output_provided {
		clippage := len(input_filename)-1;
		for ; clippage > 0; clippage -= 1 {
			if output_filename[clippage] == '.' {
				break
			}
		}
		if clippage >= 1 {
			output_filename = output_filename[:clippage]
		}
		//output_filename = fmt.Sprintf("%v.nasm", output_filename)
	}

	file, err := os.ReadFile(input_filename)
	if err != nil {
		fmt.Printf("Triops: Couldn't read file `%v`", input_filename)
		fmt.Print(triops_help_message)
		os.Exit(1)
	}
	full_text := string(file)
	if len(full_text) == 0 {
		fmt.Println("Triops: Empty file, expected at least `entry`")
		os.Exit(1)
	}

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
			token.tag = keywords[token_txt_str(token, full_text)] /* TODO: anytime keywords[] is checked, also check for # if it isn't a keyword, then error */
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
	if len(tokens) == 0 {
		fmt.Println("There is no code in this file (only comments), expected at least `entry`")
		os.Exit(1)
	}

	var global_scope Scope
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
			if !parse_variable_decl(&set, &global_scope, SPEC_NONE) { error_count += 1 }
			if set.index == old_index {
				print_error_line(&set, "Unknown type for global declaration")
				error_count += 1
//				print_error_line("Runtime expressions are not allowed in the global scope (`entry` would be the place for that)", tokens[i], &global_scope)
			}
		case KEYWORD_OPEN_PAREN:
			inc(&set)
			error_count += parse_proc_decl(&set, &global_scope)
		case KEYWORD_ASM: fallthrough
		case KEYWORD_ENTRY:
			inc(&set)
			error_count += parse_asm(&set, &global_scope)
			for _, lbl_token := range global_scope.label_uses {
				this := where_is(&global_scope, token_txt_str(lbl_token, set.text))
				if this.named_thing != NAME_LABEL {
					print_error_line_token_txt(lbl_token, set.text, "Usage of a label that doesn't exist")
					error_count += 1
				}
			}
		default:
			print_error_line(&set, "Unexpected toplevel token")
			skip_statement(&set)
			error_count += 1
		}
	}
/*	fmt.Println()
	for _, typ := range bare_types { fmt.Println(typ) }
	for _, typ := range indirect_types { fmt.Println(typ) }
	fmt.Println()/*
	for name, enum := range global_scope.enums { fmt.Println(name, enum) }
	for name, value := range enum_values { fmt.Println(name, value) }
	fmt.Println()/*
	for name, decl := range global_scope.decls { fmt.Println(name, decl) }/*
	fmt.Println(all_procs)/*
	fmt.Println(all_values)
*/
	if error_count != 0 {
		fmt.Printf("Amount of errors: %v\n", error_count)
		os.Exit(1)
	}

	full_asm, _ := generate_assembly(&global_scope, &set, "entry", make_small_exe)

	nasm_filename := output_filename
	if generate_executable { nasm_filename = fmt.Sprintf("%v.nasm", nasm_filename) }
	err = os.WriteFile(nasm_filename, []byte(full_asm), 0666)
	if err != nil {
		fmt.Printf("Triops: Could not write `%v` to disk:\n", nasm_filename)
		fmt.Println(err)
		os.Exit(1)
	}

	var program_output strings.Builder
	if generate_executable && make_small_exe {
		cmd := exec.Command("nasm", "-f", "bin", "-o", output_filename, nasm_filename)
		cmd.Stdout = &program_output
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Triops: nasm error `%v`:\n", err)
			fmt.Print(program_output.String())
			os.Exit(1)
		}
		os.Remove(nasm_filename)

		cmd = exec.Command("chmod", "+x", output_filename)
		cmd.Stdout = &program_output
		err = cmd.Run()
		if err != nil {
			fmt.Printf("Triops: chmod error `%v`:\n", err)
			fmt.Print(program_output.String())
			os.Remove(output_filename)
			os.Exit(1)
		}
	}
		
	if generate_executable && !make_small_exe {
		object_filename := fmt.Sprintf("%v.o", output_filename)
		cmd := exec.Command("nasm", "-f", "elf64", "-o", object_filename, nasm_filename)
		cmd.Stdout = &program_output
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Triops: nasm error `%v`:\n", err)
			fmt.Print(program_output.String())
			os.Exit(1)
		}
		os.Remove(nasm_filename)

		cmd = exec.Command("ld", "-o", output_filename, object_filename)
		cmd.Stdout = &program_output
		err = cmd.Run()
		os.Remove(object_filename)
		if err != nil {
			fmt.Printf("Triops: ld error `%v`:\n", err)
			fmt.Print(program_output.String())
			os.Exit(1)
		}
	}

	if generate_executable && run_executable {
		cmd := exec.Command(fmt.Sprintf("./%v", output_filename))
		cmd.Stdout = &program_output
		cmd.Run()
		fmt.Print(program_output.String())
	}
}

/* TODO: one-time explainers about specific errors */
func print_error_line(set *Token_Set, message string, args ...any) {
	print_error_line_token_txt(curr(set), set.text, message, args...)
}

func print_error_line_token_txt(token Token, text string, message string, args ...any) {
	var full_line Token
	/* line_nr and start of the line */
	line_nr := 1
	for i := token.pos; i < token.pos + 1; i -= 1 {
		char := text[i]
		if line_nr == 1 && char == '\n' { full_line.pos = i + 1 }
		if char == '\n' { line_nr += 1 }
	}
	/* end of the line */
	for i := full_line.pos; i < uint32(len(text)); i += 1 {
		full_line.len += 1
		if text[i] == '\n' { break }
	}
	/* main print */
	full_line_str := token_txt_str(full_line, text)
	fmt.Printf("%s:\n", fmt.Sprintf(message, args...))
	chars_written, _ := fmt.Printf("%d | ", line_nr)
	fmt.Printf("%s", full_line_str)
	if full_line_str[len(full_line_str)-1] != '\n' { fmt.Println() }
	/* the  c a r e t s */
	for i := 0; i < chars_written; i += 1 { fmt.Print(" ") }
	for i := uint32(0); i < token.pos - full_line.pos; i += 1 {
		if is_white(rune(full_line_str[i])) { fmt.Printf("%c", rune(full_line_str[i]))
		} else { fmt.Print(" ") }
	}
	for i := uint16(0); i < token.len; i += 1 { fmt.Print("^") }
	fmt.Println()
}

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

func prev(set *Token_Set) Token {
	return set.tokens[set.index-1]
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

func where_is(scope *Scope, name string) What {
	for _, what := range scope.names {
		if what.name == name {
			return what
		}
	}
	prev_scope := scope.prev_scope
	for prev_scope != nil {
		/* TODO: Global variable management */
		for _, what := range prev_scope.names {
			if what.named_thing != NAME_DECL && what.named_thing != NAME_LABEL && what.name == name {
				return what
			}
		}
		prev_scope = prev_scope.prev_scope
	}
	return What{"", 0, NAME_NOT_HERE, SPEC_NONE}
}

func add_type_to_scope(scope *Scope, name string, ti Type_Index) {
	all_types = append(all_types, ti)
	scope.names = append(scope.names, What{name, len(all_types)-1, NAME_TYPE, SPEC_NONE})
}

func add_enum_to_scope(scope *Scope, name string, enum_des Enum_Des) {
	all_enums = append(all_enums, enum_des)
	scope.names = append(scope.names, What{name, len(all_enums)-1, NAME_ENUM, SPEC_NONE})
}

func add_decl_to_scope(scope *Scope, name string, decl_des Decl_Des, specialty Decl_Specialty) {
	all_decls = append(all_decls, decl_des)
	scope.names = append(scope.names, What{name, len(all_decls)-1, NAME_DECL, specialty})
}

func add_label_to_scope(scope *Scope, name string, place int) {
	all_labels = append(all_labels, place)
	scope.names = append(scope.names, What{name, len(all_labels)-1, NAME_LABEL, SPEC_NONE})
}

func add_proc_to_scope(scope *Scope, name string, proc Scope) {
	all_procs = append(all_procs, proc)
	scope.names = append(scope.names, What{name, len(all_procs)-1, NAME_PROC, SPEC_NONE})
}