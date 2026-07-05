package btree

import (
	. "BYO_database/internal/utils"
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

const (
	BNODE_NODE = 1 // internal nodes without values
	BNODE_LEAF = 2 // leaf nodes with values
)

func main() {
	node1max := HEADER + B_TREE_POINTER_SIZE + B_TREE_OFFSET_SIZE + 4 + B_TREE_MAX_KEY_SIZE + B_TREE_MAX_VAL_SIZE
	fmt.Println(node1max)
	Assert(node1max > B_TREE_PAGE_SIZE, "Error: Max node size is larger than Page size")
}
