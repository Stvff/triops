# Layout of basic types

## Dynamic arrays
A dynamic array is a `2 columns of 8 bytes` type consisting of a pointer and a length (in that order).
The length is expressed in terms of elements of the type the dynamic array contains, so not the size in bytes.
The pointer is first to make the dynamic array 'compatible' with pointers; the final generated assembly for indexing a variable that is a pointer, and that for indexing one that is a dynamic array, is exactly the same.
In normal triops code, indexing the contents can be done directly on the variable. However, because of its `2 columns of 8 bytes` layout, indexing a dynamic array variable in Assembly mode can only select between its pointer and length.

The pointer of the array points to the first value in the array. 8 bytes in front of that first value, the 'allocated size' is stored, which is expressed in terms of bytes (unlike array length).
This is to tell functions that deal with dynamic allocations when they have to allocate more, or new, space. If the desired length is greater than or equal to the allocated length, another allocation needs to happen.
Consequently, if that value is 0, any change in size will result in an allocation (and `unmap()` will not be called on the old pointer).

## Initializing dynamic arrays at runtime
Initialization of dynamic array is a complex task, for multiple reasons.
A dynamic array has a pointer and a length. If a dynamic array is unitialized by the user, then both are zero, no problem.
However, when a literal is provided, that literal exists in the `.data` section, so that place needs to be wellknown. From here, there are two cases: The dynamic array is a variable, or it is part of a nested literal.
In the first case, the pointer to the `.data` section (and length) must be put into a variable in the stack. In the second case, those values need to be put in a different part of the `.data` section, at runtime.

The most general explanation would likely be very hard to follow, so I'll give an example first.
