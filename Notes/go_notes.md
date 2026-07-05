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