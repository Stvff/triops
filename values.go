package main

var value_head int
var all_values []byte

type Value struct {
	form Value_Form
	pos, len int
}
type Value_Form int

const (
	VALUE_FORM_NONE = 0
	VALUE_FORM_WILD = 1 + iota
	VALUE_FORM_INTEGER
	VALUE_FORM_FLOAT
	VALUE_FORM_STRING
	VALUE_FORM_BYTES
)

func integer_to_value(integer int) (value Value) {
	temp := integer
	byte_amount := 0
	if temp == 0 {
		byte_amount = 1
	} else { for temp != 0 {
		temp /= 255
		byte_amount += 1
	}}
	value = make_value(byte_amount)
	value.form = VALUE_FORM_INTEGER
	for i := 0; i < value.len; i += 1 {
		all_values[value.pos + i] = byte(integer >> (8*i))
	}
	return value
}

func integer_at_value(value Value, integer int) {
	for i := 0; i < 8; i += 1 {
		all_values[value.pos + i] = byte(integer >> (8*i))
	}
}

func integer_to_sized_value(integer int, size int) (value Value) {
	value = make_value(size)
	value.form = VALUE_FORM_INTEGER
	for i := 0; i < value.len && i < 8; i += 1 {
		all_values[value.pos + i] = byte(integer >> (8*i))
	}
	return value
}

func value_to_integer(value Value) (integer int, exists bool) {
	for i := 0; i < value.len; i += 1 {
		integer += int(all_values[value.pos+i]) << (8*i)
	}
	return integer, value.form == VALUE_FORM_INTEGER
}

func increment_value(value Value) (new_value Value) {
	if value.len == 0 { new_value = make_value(1)
	} else { new_value = make_value(value.len) }
	copy_value(&new_value, value)
	for i := 0; i < value.len; i += 1 {
		v := all_values[new_value.pos + i]
		all_values[new_value.pos + i] = v + 1
		if v == 255 { continue }
		break
	}
	return new_value
}

func make_value(size int) (value Value) {
	value.pos = value_head
	value.len = size
	value_head += size
	all_values = append(all_values, make([]byte, size)...)
	return value
}

func copy_value(a *Value, b Value) {
	a.form = b.form
	copy(all_values[a.pos:a.pos+a.len], all_values[b.pos:b.pos+b.len])
}
