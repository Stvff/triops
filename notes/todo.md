# Things for when not sure what to do:

## Normal expressions
1. Figure out representation for both syntax trees and assembly instructions
2. Only then can a proper parser be written
3. And then the actual codegen ofc

## Initialization codegen
Really it can only oscillate between two types: reference, and direct.
Static arrays are just blobs of subtypes (up until a reference type).
Pointers are just static arrays without the size pointer.
So the 'decision tree' during init gen looks at:
- is this a reference or direct type?
- until where is it that type?
- recurse and fill appropriately based on whether or not it's a direct or reference type below that
