```go
// For an on-disk B+tree, the database file is an array of pages (nodes) referenced by page numbers (pointers)
type BTree struct {
	// pointer (non-zero page number)
	root uint64
	// callbacks for managing on-disk pages
  // These are NOT function prototypes - Go does not allow defining function prototypes in structs
  // These are first class objects that sit in as placeholders for runtime-defined methods 
	get func(uint64) []byte // reads a page from disk
	new func([]byte) uint64 // Allocates and writes a new page (copy-on-write)
	del func(uint64)        // deallocates a page
}
```
- Example usage with a truncated example
```go
package main

import "fmt"

type BTree struct {
	root uint64
	get  func(uint64) []byte // The function field
}

// 1. You write a real function somewhere else that talks to the disk
func readFromDisk(pageID uint64) []byte {
	fmt.Printf("Reading page %d from disk...\n", pageID)
	return []byte{0x01, 0x02} // fake disk data
}

func main() {
	// 2. You instantiate the struct and "plug in" your real function
	tree := BTree{
		root: 1,
		get:  readFromDisk, // Passing the function as data!
	}

	// 3. The tree can now execute that code by calling the field name
	pageData := tree.get(tree.root) 
}
```

- Only `new` is a reserved keyword but you can still override the default keyword with a variable name
- Methods vs. Function Fields

| Feature | Method: func (node BNode) getOffset() | Function Field : get func(uint64) []byte |
| -------- | -------- | -------- |
| Where it's defined | Outside the struct, hardcoded at compile time | Inside the struct as a data field | 
| Can it change? | No. The logic is permanent for that type | Yes. You can swap the function out at runtime | 
|Memory Cost | Free. It doesn't add bytes to the struct size | Costs 8 bytes per field (it's a pointer to code) | 
| Purpose | To define what a type does.To let the struct borrow behavior from somewhere else. | 

- The syntax `func (node BNode) getPointer(index uint16) uint64` means that we are defining `getPointer()` as a **method** on the BNode struct
  - So we could call this method as `node.getPointer(...)` 

- In Go, when you slice an array using `[start:end]`, both numbers must be absolute positions measured from the very beginning of the array (index 0).
- `pos+4` is an absolute position (e.g., byte 3004).
- `klen` is just a length (e.g., 4 bytes long).
-  It is a relative size, not a position from the start of the page.
-  If you try to write `node.data[3004 : 4]`, Go sees that your end number (4) is smaller than your start number (3004).
-  This will cause your program to panic instantly with a "slice bounds out of range" error.
-  The other option is `node.data[pos+4:][:klen]` 
   -  `node.data[pos+4:]` creates a brand new, temporary slice where the key starts at index 0
   -  Because index 0 is reset to the start of the key, doing [:klen] correctly clips it at the length of the key.
-  

# The ...Page Syntax (Variadic Parameter)
`func ReplaceChildNode(tree *BTree, new_p Page, old_p Page, index uint16, children ...Page) {}`
The `...` before the type Page makes this a variadic parameter.
In Go, it means you can pass zero, one, or many Page objects to this function, and the function will treat them as a slice (`[]Page`) inside the function body.
```
Example Call:
ReplaceChildNode(myTree, new_p, old_p, 2, child1) (Passes 1 child)
ReplaceChildNode(myTree, new_p, old_p, 2, child1, child2) (Passes 2 children)
```

This is incredibly useful for B-trees because when a node splits, it might result in two children (a standard split), or potentially more if you were doing advanced rebalancing.

# What does tree *BTree mean?
`func ReplaceChildNode(tree *BTree, new_p Page, old_p Page, index uint16, children ...Page) {}`
The * symbol indicates that you are passing a pointer to the BTree struct, not a copy of the struct itself.
- Passing by value (tree BTree): Go would make a full copy of the BTree struct every time the function is called. If you modified tree.root inside the function, the original BTree outside the function wouldn't change.
- Passing by pointer (tree *BTree): Go passes the memory address of the BTree. This allows the function to see the "real" BTree and make permanent changes to it.
- By passing a pointer, you can access the configuration and the callback functions of the tree efficiently without copying data.