package main

/* this returns a set of (configurable) instructions */
func parse_asm_block(set *Token_Set, scope *Scope) bool {
	return finish_block(set)
}

func finish_block(set *Token_Set) bool {
	return skip_statement(set)
}
