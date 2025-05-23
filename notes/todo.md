# Things for when not sure what to do (so also, currently in progress):
Today I want done:
1. [x] New asm parsing
2. [x] Implement register typesystem
2. [ ] New asm codegen
3. [ ] Think really hard about expression codegen
4. [ ] Finalize calling convention
5. [ ] Figure out how I want conditionals to look (related to labels in inline blocks)
6. [ ] Optional: initialization for static arrays

## Normal expressions
### Syntax tree representation
There is an array of nodes, and an array of links.\
Links contain two indices into the nodes array, as well as information about what sort of link they are.\
The first index is always smaller than the second index (indeed, they are `left` and `right`).
Nodes contain:
- Kind of the node (variable, constant, function)
- Possible value stored in it
- List of required types for the arguments? (left and right) (or a way to get them)
- Amount of satisfied arguments (left and right)

### Parsing towards the syntax tree
With a function that needs arguments on the right, link them all to the function (calling `parse_node` on them before moving too much, to deal with `[]`, `.`, deeper function calls).
With a function that needs arguments on the left, it has to check for ownership of the already node-ified arguments (by looking at the links at the top of the link list),
and if its precendence beats that of the other function, it gets to steal them. If its precendence is lower, it will have to take the other function as argument.

### Codegen and register allocation

## Initialization codegen
### Some examples
- `type[][][][]` is already solved: it involves keeping a ledger of initialized arrays, and when the depth of the previous ledger is further than 1 from the current depth, that array can be written to `.data`/`.text`.
- `type[][0][][0]` is the same process as `type[][][][]`, where a pointer is an array with one element, and no length/allocation precautions.
- `type[3][4][5]` is really just `type[60]`, which itself is just a modification of the `columns`-amount, and the ordering/majorness is taken into account by the parser.

- `type[][3]` seems to be a more unique operation, maybe simply 'repeat the process for `type[]` 3 times...
- `type[][3][][4]` should follow from the 3rd case here, but I'm not convinced it's so obvious

### Somewhat of a formalization
Given Interactions:
- ` T `
- ` T[] `
- ` *[][]* `

Possible Interaction Substitutions:
- ` T[n]*  ->  T'* `         Via adjusting the columns amount
- ` *[0]*  ->  *[]'* `       Via a special case of one element and a `mov [label], rsp + varpos`
- ` *[u][v]*  ->  *[n']* `   Via multiplying u by v

Needed Interaction:
- ` *[][n]* `
It seems this last interaction is needed as an axiom, and cannot really be a substitution