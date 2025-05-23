# Compiler internals (unfinished)
For the compiler, I wanted to minimize amount of seperately allocated regions in memory, and to maximise the usage of that memory.
This is for a couple intended effects:
- Reducing the load (and reliance) on go's garbage collector
- Simplifying implementation
- Optimizing cache usage
I spent quite some time thinking about to achieve these goals (for better or for worse), so here I'll document some of those thoughts, and the current solution.
For now, I won't focus too much on the actual architecture of the compiler. Maybe I'll make another document for that later. (besides, the full architecture is not done yet)

Naturally, a familiarity with Triops (for example, as outlined in my "Triops as (x64) Assembler" guide) is assumed.

## Initial 'solutions'
The main `scope` struct contained:
```go
types map[string]Type_Des
enums map[string]Enum_Des
decls map[string]Decl_Des
```
First, the type description struct was:
```go
type Type_Des struct {
	name string
	element_size, element_amount int
}
```
Notibly, there was no way to define a type to have indirection. The dimensions of this struct are `8 by 4`, so 32 bytes, relatively efficiently packed (that is, it needs no padding).
Since `name` was used to index the `types` map, that value is duplicated.

Then, the variable declaration description:
```go
type Decl_Des struct {
	name string
	amount []int
	typ Type_Des
	init Value
}
```
Indirection was saved on the variable (as `amount`, a dynamic array of indirections...), not on the type.
An element of `amount` would contain a value greater than zero for static arrays (the amount of elements in said array), 0 for pointers, and -1 for dynamic arrays (potentially -amount_of_elements).
The dimensions of this struct are `8 by 13`, so 104 bytes. There's no padding, but it's quite inefficient, as most of that bitwidth will never be used.
On top of that, since `amount` is an array, every variable declaration will have an extra (almost never more than 4 elements long) array flying around in memory.
In fact, the `Value` struct:
```go
type Value struct {
	v []byte
	value_type int
}
```
Contains another dynamic array, which is also usually a small set of elements.

Continuing at the enum definition struct, we can see that that too could not have any indirection. Enums were limited to 'bare types'
```go
type Enum_Des struct {
	name string
	typ Type_Des
	values map[string]Value
}
```
I don't exactly know the fields of a map, or how much space it takes, but I do know that it likely has at least two loose memory regions attached to it.

Lastly, I should mention that there was no real Token type. Tokens were `[]rune`, and parsed on-the-fly. This method had some advantages in terms of memory, but it
ended up becoming very unwieldy.

## Thinking about types
The main thing that was bothering me was the typesystem. I wanted indirections to be attached to the type, not only variables. That, and the fact that I was constantly littering
my small arrays all over memory, made me extremely unhappy.\
I briefly considered limiting the amount of indirections to 4 (because honestly, have you ever seen someone use more than 4 levels in declaration?), and making it a static array
on the type struct:
```go
type Type_Des struct {
	name string
	element_size, element_amount int
	indirection [4]int
}
```
However, now my types are all always `4 by 8` (twice as large!), and this hasn't solved another problem:
if we want to declare a variable with an indirection, what should that type's name be? Should we hash it along with its dimensions and indirections?

In other words, that wouldn't cut it.

We'll ignore any overly academic typetheory, but when we look at what a semi-capable typesystem is, it's essentially a tree, with a limited set of kinds of branches.
In this tree, root nodes are your basic types, and the 'kinds of branches' would be something like an array subtype, a pointer subtype, or just, a renaming of a previous type.

The way I chose to implement this behaviour is by having a set of lists with each different type of branched node.
