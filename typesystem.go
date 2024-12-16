package main
//import "fmt"
const (
	TYPE_ERR = 0
	TYPE_BARE = 1 + iota
	TYPE_INDIRECT
	TYPE_RT_ARRAY
	TYPE_ST_ARRAY
	TYPE_POINTER
	TYPE_STRUCT
)

const TI_SHIFT_AMOUNT = 3
const TI_MASK = 0b111
type Type_Index uint32 /* this is a packed integer */
type Type_Tag uint8 /* this lives in the bottom TI_SHIFT_AMOUNT bits */

var (
	bare_types []Type_Des_Bare
	indirect_types []Type_Des_Indirect
	rt_array_types []Type_Des_RT_Array
	st_array_types []Type_Des_St_Array
	pointer_types []Type_Des_Pointer
)

type Type_Des_Bare struct{
	name string
	alignment, amount int
	form Value_Form
}

type Type_Des_Indirect struct{
	name string
	target Type_Index
}

type Type_Des_RT_Array struct{
	target Type_Index
}

type Type_Des_St_Array struct{
	target Type_Index
	size int
}

type Type_Des_Pointer struct{
	target Type_Index
}
/*
type Type_Des_Struct struct{
	name string
	decls []Decl_Des
	the names of all these decls have to be allocated in a continuous buffer, similar to how Value and Token_Set is done
}
*/

func size_of_type(ti Type_Index) int {
	i, t := unpack_ti(ti)
	switch t {
		case TYPE_ERR: panic("size_of_type: internal type error")
		case TYPE_BARE:
			return bare_types[i].alignment*bare_types[i].amount
		case TYPE_INDIRECT:
			return size_of_type(follow_type(ti))
		case TYPE_RT_ARRAY:
			return 16
		case TYPE_ST_ARRAY:
			return st_array_types[i].size*size_of_type(follow_type(ti))
		case TYPE_POINTER:
			return 8
		case TYPE_STRUCT:
			panic("size_of_type: struct")
	}
	panic("size_of_type: unreachable")
}

func align_of_type(ti Type_Index) int {
	i, t := unpack_ti(ti)
	switch t {
		case TYPE_ERR: panic("align_of_type: internal type error")
		case TYPE_BARE:
			return bare_types[i].alignment
		case TYPE_INDIRECT:
			return align_of_type(follow_type(ti))
		case TYPE_RT_ARRAY:
			return 8
		case TYPE_ST_ARRAY:
			return align_of_type(follow_type(ti))
		case TYPE_POINTER:
			return 8
		case TYPE_STRUCT:
			panic("align_of_type: struct")
	}
	panic("align_of_type: unreachable")
}

func amount_of_type(ti Type_Index) int {
	i, t := unpack_ti(ti)
	switch t {
		case TYPE_ERR: panic("align_of_type: internal type error")
		case TYPE_BARE:
			return bare_types[i].amount
		case TYPE_INDIRECT:
			return amount_of_type(follow_type(ti))
		case TYPE_RT_ARRAY:
			return 2
		case TYPE_ST_ARRAY:
			return st_array_types[i].size*amount_of_type(follow_type(ti))
		case TYPE_POINTER:
			return 1
		case TYPE_STRUCT:
			panic("align_of_type: struct")
	}
	panic("align_of_type: unreachable")
}


func are_types_equal(left, right Type_Index) bool {
	var (
		left_i, right_i int
		left_tag, right_tag Type_Tag
	)
	left_i, left_tag = unpack_ti(left)
	right_i, right_tag = unpack_ti(right)
	for {
		if left == right { return true }
		if left_tag == TYPE_INDIRECT {
			left = indirect_types[left_i].target
			left_i, left_tag = unpack_ti(left)
			continue
		}
		if right_tag == TYPE_INDIRECT {
			right = indirect_types[right_i].target
			right_i, right_tag = unpack_ti(right)
			continue
		}
		if left_tag != right_tag { return false }
		if left_tag == TYPE_ST_ARRAY &&
		   st_array_types[left_i].size != st_array_types[right_i].size { return false }
		left = follow_type(left)
		right = follow_type(right)
	}
	return true
}

func follow_type(ti Type_Index) Type_Index {
	i, t := unpack_ti(ti)
	switch t {
		case TYPE_ERR: panic("follow_type: internal type error")
		case TYPE_BARE:
			return ti
		case TYPE_INDIRECT:
			ti = indirect_types[i].target
		case TYPE_RT_ARRAY:
			ti = rt_array_types[i].target
		case TYPE_ST_ARRAY:
			ti = st_array_types[i].target
		case TYPE_POINTER:
			ti = pointer_types[i].target
		case TYPE_STRUCT:
			panic("follow_type: struct")
	}
	i, t = unpack_ti(ti)
	for t == TYPE_INDIRECT {
		ti = indirect_types[i].target
		i, t = unpack_ti(ti)
	}
	return ti
}

func append_bare_type(typ Type_Des_Bare) Type_Index {
	bare_types = append(bare_types, typ)
	return pack_ti(len(bare_types) - 1, TYPE_BARE)
}

func append_indirect_type(typ Type_Des_Indirect) Type_Index {
	indirect_types = append(indirect_types, typ)
	return pack_ti(len(indirect_types) - 1, TYPE_INDIRECT)
}

func append_rt_array_type(typ Type_Des_RT_Array) Type_Index {
	rt_array_types = append(rt_array_types, typ)
	return pack_ti(len(rt_array_types) - 1, TYPE_RT_ARRAY)
}

func append_st_array_type(typ Type_Des_St_Array) Type_Index {
	st_array_types = append(st_array_types, typ)
	return pack_ti(len(st_array_types) - 1, TYPE_ST_ARRAY)
}

func append_pointer_type(typ Type_Des_Pointer) Type_Index {
	pointer_types = append(pointer_types, typ)
	return pack_ti(len(pointer_types) - 1, TYPE_POINTER)
}

func unpack_ti(ti Type_Index) (index int, tag Type_Tag) {
	index = int(ti) >> TI_SHIFT_AMOUNT
	tag = Type_Tag(ti) & TI_MASK
	return index, tag
}

func pack_ti(i int, tag Type_Tag) (ti Type_Index) {
	index_part := Type_Index(i) << TI_SHIFT_AMOUNT
	tag_part := Type_Index(tag) & TI_MASK
	return index_part | tag_part
}
