package btree

import (
	. "BYO_database/internal/storage"
	. "BYO_database/internal/utils"
	"bytes"
	"fmt"
)

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

// Go syntax for enums
const (
	// We need to make sure we start at 1 because uninitialized memory is all filled with 0s
	BNODE_NODE = 1 // internal nodes without values
	BNODE_LEAF = 2 // leaf nodes with values
)

func NodeLookupLE(node Page, key []byte) uint16 {
	nkeys := node.NKeys()
	found := uint16(0) // Default answer if the search key is bigger than everything

	// Binary Search
	var low uint16 = 0
	var high uint16 = nkeys - 1

	for low <= high {
		mid := low + (high-low)/2  // Index of midpoint
		midKey := node.GetKey(mid) // Value of midpoint key

		//    0 if equal
		//   -1 if midKey < key
		//   +1 if midKey > key
		cmp := bytes.Compare(midKey, key)

		if cmp >= 0 {
			found = mid

			if mid == 0 { // Prevent integer underflow because high is unsigned and 0 - 1 will become 65535
				break
			}

			high = mid - 1
		} else {
			low = mid + 1
		}
	}

	return found
}

func main() {
	node1max := HEADER + B_TREE_POINTER_SIZE + B_TREE_OFFSET_SIZE + 4 + B_TREE_MAX_KEY_SIZE + B_TREE_MAX_VAL_SIZE
	fmt.Println(node1max)
	Assert(node1max > B_TREE_PAGE_SIZE, "Error: Max node size is larger than Page size")
}
