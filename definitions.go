package main

var (
	all_types  []Type_Index
	all_enums  []Enum_Des
	all_decls  []Decl_Des
	all_labels []int
	all_procs  []Scope

	all_nodes []Node
)

type Scope struct {
	prev_scope *Scope
	definition_location Token
	precedence int8
	is_inline bool
	names []What
	label_uses []Token
//	imports map[string]*Scope
//	code Code_Block
	assembly Asm_Block
	code []Link
}

/*** GENERIC ***/
type Link struct {
	kind Link_Kind
	left int
	right int
}
type Link_Kind int8
const (
	LKIND_NONE = 1
	LKIND_LEFT_ARG = 1 + iota
	LKIND_RIGHT_ARG
	LKIND_COMMA
	LKIND_INDEX
)

type Node struct {
	kind Node_Kind
	token Token
	imm Value
	satisfied_left int
	satisfied_right int
}
type Node_Kind int8
const (
	NKIND_NONE = 1
	NKIND_IMMEDIATE = 1 + iota
	NKIND_PROCEDURE
	NKIND_VARIABLE
	NKIND_LABEL
)
/*** GENERIC ***/

/*** ASM ***/
type Asm_Block struct {
	instructions []Asm_Instruction
}

/* this'd be sized [4][8]byte */
type Asm_Arg struct {
	verbatim Token
	immediate Value
}

type Asm_Instruction struct {
	mnemonic Token
	alignment int
	args [3]Asm_Arg
}
/*** ASM end ***/

var global_enum_id int = 0
var enum_values map[Enum_Value_ID]Value = make(map[Enum_Value_ID]Value)

type Enum_Des struct {
	typ Type_Index
	id int
}

type Enum_Value_ID struct {
	parent_id int
	name string
}

type Decl_Des struct {
	typ Type_Index
	init Value
	bound_register Token
}

type What struct {
	name string
	index int;
	named_thing Named_Thing;
	specialty Decl_Specialty;
}
type Named_Thing int8;
const (
	NAME_NOT_HERE = 0
	NAME_TYPE = 1 + iota
	NAME_ENUM
	NAME_DECL
	NAME_LABEL
	NAME_PROC
)
type Decl_Specialty int8;
const (
	SPEC_NONE = 0
	SPEC_OUTPUT = 1 + iota
	SPEC_LINPUT
	SPEC_RINPUT
)

type Token_Tag int16
var keywords = map[string]Token_Tag {
	"import"  : KEYWORD_IMPORT,
	"type"    : KEYWORD_TYPE,
	"struct"  : KEYWORD_STRUCT,
	"enum"    : KEYWORD_ENUM,
	"entry"   : KEYWORD_ENTRY,
	"macro"   : KEYWORD_MACRO,

	"is"      : KEYWORD_IS,
	"columns" : KEYWORD_COLUMNS,
	"of"      : KEYWORD_OF,
	"bytes"   : KEYWORD_BYTES,

	"prec"   : KEYWORD_PREC,
	"asm"    : KEYWORD_ASM,
	"insert" : KEYWORD_INSERT,
	"inline" : KEYWORD_INLINE,

	"for"      : KEYWORD_FOR,
	"continue" : KEYWORD_CONTINUE,
	"break"    : KEYWORD_BREAK,
	"if"       : KEYWORD_IF,
	"else"     : KEYWORD_ELSE,
	"return"   : KEYWORD_RETURN,
	"defer"    : KEYWORD_DEFER,

	"."  : KEYWORD_DOT,
	";"  : KEYWORD_SEMICOLON,
	","  : KEYWORD_COMMA,
	"^"  : KEYWORD_CARET,
	"["  : KEYWORD_OPEN_BRACKET,
	"]"  : KEYWORD_CLOSE_BRACKET,
	"("  : KEYWORD_OPEN_PAREN,
	")"  : KEYWORD_CLOSE_PAREN,
	"{"  : KEYWORD_OPEN_BRACE,
	"}"  : KEYWORD_CLOSE_BRACE,
	"="  : KEYWORD_EQUALS,
	"\"" : KEYWORD_DQUOTE,
	"'"  : KEYWORD_SQUOTE,
	"#"  : KEYWORD_POUND,

	"#intform"    : DIRECTIVE_INTFORM,
	"#floatform"  : DIRECTIVE_FLOATFORM,
	"#stringform" : DIRECTIVE_STRINGFORM,

	"#rb" : DIRECTIVE_REG_BYTE,
	"#rw" : DIRECTIVE_REG_WORD,
	"#rd" : DIRECTIVE_REG_DOUB,
	"#rq" : DIRECTIVE_REG_QUAD,
	"#ro" : DIRECTIVE_REG_OCTO,
	"#reg": DIRECTIVE_REG,

	"#lbl" : DIRECTIVE_LBL,

	"#type"      : DIRECTIVE_TYPE,
	"#var"       : DIRECTIVE_VAR,
	"#decl"      : DIRECTIVE_DECL,
	"#statement" : DIRECTIVE_STATEMENT,
	"#block"     : DIRECTIVE_BLOCK,
}

const (
	NONE = 0
	KEYWORDS_START = 1 + iota
		KEYWORD_IMPORT
		KEYWORD_TYPE
		KEYWORD_STRUCT
		KEYWORD_ENUM
		KEYWORD_ENTRY
		KEYWORD_MACRO
	
		KEYWORD_IS
		KEYWORD_COLUMNS
		KEYWORD_OF
		KEYWORD_BYTES
	
		KEYWORD_PREC
		KEYWORD_ASM
		KEYWORD_INSERT
		KEYWORD_INLINE
	
		KEYWORD_FOR
		KEYWORD_CONTINUE
		KEYWORD_BREAK
		KEYWORD_IF
		KEYWORD_ELSE
		KEYWORD_RETURN
		KEYWORD_DEFER
	
		KEYWORD_DOT
		KEYWORD_SEMICOLON
		KEYWORD_COMMA
		KEYWORD_CARET
		KEYWORD_OPEN_BRACKET
		KEYWORD_CLOSE_BRACKET
		KEYWORD_OPEN_PAREN
		KEYWORD_CLOSE_PAREN
		KEYWORD_OPEN_BRACE
		KEYWORD_CLOSE_BRACE
		KEYWORD_EQUALS
		KEYWORD_DQUOTE
		KEYWORD_SQUOTE
		KEYWORD_POUND
	KEYWORDS_END

	DIRECTIVES_START
		DIRECTIVE_INTFORM
		DIRECTIVE_FLOATFORM
		DIRECTIVE_STRINGFORM
		DIRECTIVE_BYTEFORM

		DIRECTIVE_REGS_START
			DIRECTIVE_REG_BYTE
			DIRECTIVE_REG_WORD
			DIRECTIVE_REG_DOUB
			DIRECTIVE_REG_QUAD
			DIRECTIVE_REG_OCTO
			DIRECTIVE_REG
		DIRECTIVE_REGS_END

		DIRECTIVE_LBL

		DIRECTIVE_TYPE
		DIRECTIVE_VAR
		DIRECTIVE_DECL
		DIRECTIVE_STATEMENT
		DIRECTIVE_BLOCK
	DIRECTIVES_END

	VALUES_START
		VALUE_NONE
		VALUE_INTEGER
		VALUE_FLOAT
		VALUE_STRING
		VALUE_BYTES
		VALUE_OTHER
	VALUES_END
)

type Token_Set struct {
	index int
	codebraces int
	braces int
	commas_and_parens_as_semis bool
	end bool
	text string
	tokens []Token
}

type Token struct {
	tag Token_Tag
	len uint16
	pos uint32
}
