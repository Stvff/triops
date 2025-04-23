# Layout of basic types

## Static arrays
Static arrays are essentially an extension (multiplication) of columns in terms of memory layout.

## Pointers
Pointers that point to variables are the (runtime) value of the stack pointer register (`rsp`) plus some offset.

### Initializing pointers that point to variables
Because pointers are values that are really only known at runtime, they need to be initialized with `mov`s, regardless of if they're nested (and live in `.data`).

## Dynamic arrays
A dynamic array is a `2 columns of 8 bytes` type consisting of a pointer and a length (in that order).
The length is expressed in terms of elements of the type the dynamic array contains, so not the size in bytes.
The pointer is first to make the dynamic array 'compatible' with pointers; the final generated assembly for indexing a variable that is a pointer, and that for indexing one that is a dynamic array, is exactly the same.
In normal triops code, indexing the contents can be done directly on the variable. However, because of its `2 columns of 8 bytes` layout, indexing a dynamic array variable in Assembly mode can only select between its pointer and length.

The pointer of the array points to the first value in the array. 8 bytes in front of that first value, the 'allocated size' is stored, which is expressed in terms of bytes (unlike array length).
This is to tell functions that deal with dynamic allocations when they have to allocate more, or new, space. If the desired length is greater than or equal to the allocated length, another allocation needs to happen.
Consequently, if that value is 0, any change in size will result in an allocation (and `unmap()` will not be called on the old pointer).

### Initializing dynamic arrays
Initialization of dynamic array is a complex task, for multiple reasons.
A dynamic array has a pointer and a length. If a dynamic array is unitialized by the user, then both are zero, no problem.
However, when a literal is provided, that literal exists in the `.data` section, so that place needs to be wellknown. From here, there are two cases: The dynamic array is a variable, or it is part of a nested literal.
In the first case, the pointer to the `.data` section (and length) must be put into a variable in the stack. In the second case, those values need to be put in a different part of the `.data` section.

The most general explanation would likely be very hard to follow, so I'll give an example first.
```
type char is 1 byte #stringform;

char[][] array_of_strings = {
	"aaa", "bbbb"
};
```

(Some explanation about the assembly it gets converted into)

The real problem is that we need to convert between two different representations of the tree that describes the data in the variable.
From parsing, the tree (in memory) is as follows:
```
	2
		3
			"aaa"
		4
			"bbb"
```

But because each level of indirection must be contiguous during runtime, the kind of tree we need is:
```
			"aaa"
			"bbbb"
		3
		pointer_to_aaa
		4
		pointer_to_bbbb
	2
	pointer_to_3
```
or (whichever is easier to make)
```
	2
	pointer_to_3
		3
		pointer_to_aaa
		4
		pointer_to_bbbb
			"aaa"
			"bbbb"
```
For the double nesting of the example, you could hardcode a relatively simple path for it, but with triple nesting and higher, it becomes a lot more difficult.
After a lot of messing about, the algorithm I used can be described like so:\
Recurse down into the type, until the bottom is reached, at which point those values are registered in `.data`.
Every time the function exist, a 'ledger' is kept about which values were just put into `.data`, its position and amount, and at what depth of recursion.
After writing to `.data` (and before writing an entry in the ledger), the current recursion is checked against the last entry in the ledger,
and if the current depth is smaller than (and not equal to) the last depth, then the info in the ledger is emptied into `.data`.

## Calling convention
