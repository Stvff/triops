# Triops as (x64) Assembler
Triops is designed to, as a language, define as few things as possible about the operations it performs.
All it aims to do for a programmer is:
1. Keep track of types, variables and constants
2. Keep track of instructions and procedures
3. Move values between variables and procedures

This starts at the typesystem, which is only based on size and alignment, and ends at inline assembly.
Triops does not provide any runtime operators or procedures out-of-the-box, aside from procedure calling and assignment.\
For anyone who wants to not first define all arithmetic operators etc, the provided core library will seamlessly provide all such standard commodities.\
Rest assured, triops will still be able to do:
```
int a = 7;
int b = 3;
int c = a + b;
```
Given the correct imports (or user definitions).

Here's a quick example of what it looks like to use Triops as an assembly language:
```
type int is 4 bytes #intform;
type char is 1 bytes #stringform;

enum int syscalls {write = 1, exit = 60};
enum int stdout = 1;

char[] msg = "Heyaa ^.^\n";

## this program says hello using linux syscalls

entry asm {
	mov #rq rax, syscalls.write;
	mov #rq rdi, stdout;
	mov #rq rsi, msg[0];
	mov #rq rdx, msg[1];
	syscall;

	mov #rq rax, syscalls.exit;
	mov #rq rdi, 0; ## success
	syscall;
}
```
Let's build up to that, starting at the typesystem.

#### Quick aside on statements and comments
Statements (and nesting) are whitespace independent, and end on mandatory semicolons.
Single line comments start with `##` and end with a newline.
Multiline comments start with `#*` and end with `*#`.

## Types
As mentioned, the typesystem is only based on size and alignment. Lets take the common integer type.
```
type int is 4 bytes;
```
This is a type declaration. It starts with `type`, then the name of the type to be declared, then `is`, and finishes with a description of that type.
In this case, `int` is now known to Triops of having an alignment of 4 bytes. Since `int` is only 1 element, this means that its size matches its alignment.

We could imagine an opaque fat pointer type (where 'opaque' means its elements are not directly named or indexable as they would be in a struct or array),
consisting of a normal 8 byte pointer, and an 8 byte length. This would have an alignment of 8 bytes, but be a size of 16 bytes, since there are two elements of 8 bytes.
So, in Triops:
```
type fat_pointer is 2 columns of 8 bytes;
```
We can imagine a small table with 2 columns and 8 rows: 16 bytes in total. Alignment is limited to powers of two, upto 16 bytes.
Alignment and element count are called the dimensions of the type (as per the table analogy).

To get back to our `int`, triops doesn't know what sort of literals this type would want, so currently it would accept any literal that has a size of 4 bytes or less.
If we wanted to restrict this, there are 4 directives we can use to share our needs with the compiler:
- `#intform`
- `#floatform`
- `#stringform`
- `#byteform`

These form annotations are placed after the `bytes` keyword, so for our `int`:
```
type int is 4 bytes #intform;
```
This can only be done for single-element types, so `fat_pointer` can't have such restrictions.

### Variable declaration and indirection
As we saw in the introduction, variables are declared 'like C', but only superficially.
In C, the type is actually declared _around_ the variable name, so that declaration is equal to usage, whereas in Triops, the type is really placed _before_ the name.

Variables can be declared without explicit initialization (which means they are set to 0), or with initialization:
```
int a;
int b = 7;
```
Furthermore, there are static and dynamic arrays, as well as pointers. These are called type 'indirections' in Triops, and they attach to the type.
A static array is an array of which the size is known at compiletime.
```
int[3] position;
```
A dynamic array is an array of which the size is not fully known at compiletime. Internally, it contains a size and a pointer to the backing memory.
```
int[] all_expenses;
```
Both static arrays and dynamic arrays can be initialized using C's array literal syntax:
```
int[3] position = {5, 3, 7};
int[] all_expenses = {50, 20, 50, 65, 99};
```
And can be indexed using normal (zero-based) array indexing:
```
int y = position[1];
int third_expense = all_expenses[2];
```

Finally, pointers are somewhat unusual in their syntax. To avoid using up operators the user might want to use, they're denoted as an array with zero elements:
```
int[0] counter;
```
They can be initialized by placing a variable of the correct type in curly braces, and dereferenced using array indexing, without an actual index value.
```
int toplevel_counter;
int[0] counter = {toplevel_counter};
int look_ahead = counter[] + 1;
counter[] = look_ahead;
```

Naturally, there can be more than just one level of indirection. They can simply be daisychained behind the type (unlike in C):
```
int[3][] position_array = {{1, 5, 7}, {8, 4, 2}};
int[3][][0] position_array_pointer = {position_array};
```

### Type aliases
Aside from being able to define bare types, the typesystem can also alias types:
```
type s32 is int;
```
`s32` is now a distinct type, but it copied over the size, alignment and form annotation from `int`.
It being 'distinct', means that two variables that are `s32` and `int` can not be assigned to eachother (without some sort of typecasting procedure).

Indirect types can also be aliased. A common example would be a floating point vec3:
```
type float is 4 bytes #floatform;
type vec3 is float[3];
```
This means that a `vec3` variable can be assigned to and indexed as an array, but only when all elements are either floating point literals, or variables of type `float`.

Aliases are always distinct, but in these cases with indirection, their indexed values are not distinct. In this case:
```
type vec3 is float[3];
type Vector3 is float[3];
```
This is not allowed:
```
vec3 speed_a = {3.4, 4.5, 0.1};
Vector3 speed_b = speed_a;
```
But this _is_ allowed:
```
vec3 speed_a = {3.4, 4.5, 0.1};
Vector3 speed_b;
speed_b[0] = speed_a[2];
```
#### Quick aside on the dimensions of indirect types
For static arrays, their alignment is the alignment of the type of which the array is constructed,
and its size is therefore the element count of the array times the element count of the type times the alignment of the type.\
For example, say we were to make an array of 5 `fat_pointer`s:
```
type fat_pointer is 2 columns of 8 bytes;
fat_pointer[5] ptrs;
```
`ptrs`'s size would be `5*2*8 = 80` bytes, and its aligment would be 8 bytes.
Dynamic arrays, since they are a size and a pointer, are `2 columns of 8 bytes`, regardless of underlaying type.
Pointers are simply 8 bytes in size and alignment.

## Enums and constants
The constant system is built up completely out of typed enums. They are declared with the `enum` keyword, then the type, the name, and then a block of optionally initialized names:
```
enum int numbers {one = 1, two, three, seven = 7};
```
Enums can be of any type, and if no initalization is given, it will be the previous value incremented by one.
This incrementation is done either as an arbitrary length integer (in the `#intform` and general cases), as a float (in the `#floatform` case), or ascii-wise (in the `#intform` case).
To use an enum, the syntax is the enum name, a period, and the subvalue name:
```
int a = numbers.three;
```
A single constant is also declared using the `enum` keyword, but without a block of names:
```
enum int ten = 10;
```

If an enum or constant is `#intform`, they can also be used in type declarations and array sizes.
```
type w_char is numbers.two bytes;
w_char[ten] ten_windows_characters;
```

## Filescope program structure (this section is a draft)
I'll be honest, procedure declarations are not done yet! I have a good idea for what it'll look like, but for now I'll have to skip it.
What I do have is `entry` and `asm`.\
In a program, there is one `entry` block. As the name implies, this is where the execution of the program starts.
As I haven't finalized expression blocks yet, we'll make the `entry` block an `asm` block as well, and we'll start at what `asm` blocks are.
### Inline assembly
`asm` blocks are always inlined, and contain a wrapper over nasm assembly, with awareness of the program's types, variables and constants.
Each statement in an `asm` block starts with an instruction, then has a variable amount of arguments, and is terminated with a semicolon.
```
add a, b;
```
Defining variables is done outside of `asm` blocks.
The instruction is copied verbatim to nasm, and not further typechecked (for now).
[The great unofficial x86 reference](https://www.felixcloutier.com/x86/) is great for seeing what we need. It might seem daunting at first, but it really is quite manageable.
(At some point I might get around to copying all of the instructions and their expected values, but other things have my priority)

For the arguments, there are three options:
- value literals (these will be translated to hexadecimal)
- variables and constant (indexed or unindexed)
- `#rb`, `#rw`, `#rd`, `#rq` followed by a specific register name (which will be copied to nasm verbatim)

Regarding the indexing of variables and constants, anything with a type that is not of one element (like `fat_pointer`, or `int[3]`),
must be indexed by a constant, until the type is a single element of its specified alignment.\
This is straightforward in the `int[3]` case:
```
int[3] position;
asm {
	push position[0];
	push position[1];
	push position[2];
}
```
In the case of `fat_pointer`, we can recall that its dimensions are `2 columns of 8`, so we need to reduce it to a single element.
In assembly blocks, this can be done by indexing.
```
fat_pointer ptr;
asm {
	push ptr[0];
	push ptr[1];
}
```
If we have an array of `fat_pointers`, we need to index multiple times, from left to right.
As an example, this `push`es the third element of `ptrs`:
```
fat_pointer[3] ptrs;
asm {
	push ptrs[2][0];
	push ptrs[2][1];
}
```
There are some caveats to this, which can be grasped with a good understanding of pointers. In an `asm` block, Triops will not de-reference pointers for us.
At least, not the ones from a variable directly.

