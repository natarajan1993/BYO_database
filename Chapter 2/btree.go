package main

import (
	"encoding/binary"
	"fmt"
)

const HEADER = 4

const B_TREE_PAGE_SIZE = 4096
const B_TREE_MAX_KEY_SIZE = 1000
const B_TREE_MAX_VAL_SIZE = 3000

func assert(b bool, message string) {
	if b {
		panic(message)
	}
}

type BNode []byte

// For an on-disk B+tree, the database file is an array of pages (nodes) referenced by page numbers (pointers)
type BTree struct {
	// pointer (non-zero page number)
	root uint64
	// callbacks for managing on-disk pages
	get func(uint64) []byte // reads a page from disk
	new func([]byte) uint64 // Allocates and writes a new page (copy-on-write)
	del func(uint64)        // deallocates a page
}

const (
	BNODE_NODE = 1 // internal nodes without values
	BNODE_LEAF = 2 // leaf nodes with values
)

// | type       | number_of_keys | pointers             | offsets            | key-values | unused |
// | 2 Bytes   | 2 Bytes         | number_of_keys * 8B | number_of_keys * 2B | ...        |        |
// ---------------------------------
// | key_len | value_len | key | val |
// | 2 Bytes | 2 Bytes  | ... | ... |
func (node BNode) btype() uint16 {
	return binary.LittleEndian.Uint16(node[0:2])
}

func (node BNode) nkeys() uint16 {
	return binary.LittleEndian.Uint16(node[2:4])
}

func (node BNode) setHeader(btype uint16, nkeys uint16) {
	binary.LittleEndian.PutUint16(node[0:2], btype)
	binary.LittleEndian.PutUint16(node[2:4], nkeys)
}

func (node BNode) getPointer(index uint16) uint64 {
	assert(index < node.nkeys(), "Error: Index exceeds total keys in node")
	pos := HEADER + 8*index
	return binary.LittleEndian.Uint64(node[pos:])
}

func (node BNode) setPtr(index uint16, val uint64) {
	assert(index >= node.nkeys(), "Error: Index exceeds total keys in node")
	pos := HEADER + 8*index
	binary.LittleEndian.PutUint64(node[pos:], val)
}

func main() {
	node1max := HEADER + 8 + 2 + 4 + B_TREE_MAX_KEY_SIZE + B_TREE_MAX_VAL_SIZE
	fmt.Println(node1max)
	assert(node1max > B_TREE_PAGE_SIZE, "Error: Max node size is larger than Page size")
}
