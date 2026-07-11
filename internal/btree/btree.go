package btree

import (
	. "BYO_database/internal/storage"
	. "BYO_database/internal/utils"
	"bytes"
	"fmt"
)

// For an on-disk B+tree, the database file is an array of pages (nodes) referenced by page numbers (pointers)
// In a B+Tree, nodes are split into two categories: Leaf Nodes (which hold actual data) and Internal Nodes (which act as directional signposts)
// Internal Nodes: The pointer represents the exact page ID of a child node.
// When navigating the tree, the database looks at a key, finds its index, and uses that exact same index in the pointer array to know which page to jump to next.
// Leaf Nodes: Depending on how you implement your engine, leaf nodes often leave this pointer empty (0) or use it to point to a sibling leaf page for ultra-fast range scans.
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

// NodeLookupLE searches a flat 4KB Page to find the index of the first key that is greater than or equal to (>=) our search key.
func NodeLookupLE(node Page, key []byte) uint16 {
	// Read the total number of keys currently stored in this 4096-byte page.
	nkeys := node.NKeys()

	// 'found' keeps track of the best candidate index we've seen so far.
	// If all keys in this node are smaller than our search key, this defaults to 0
	// (the caller code will handle this edge case, usually by taking the rightmost child or appending to the end).
	found := uint16(0)

	// Set up the boundaries for our binary search.
	// Because a Page is small (4KB), the number of keys will easily fit inside a 16-bit integer (max 65,535).
	var low uint16 = 0
	var high uint16 = nkeys - 1

	// Standard binary search loop: divide and conquer the sorted keys inside this page
	for low <= high {
		// Calculate the middle index.
		// Using 'low + (high-low)/2' instead of '(low+high)/2' to prevent integer overflow when dealing with massive numbers
		mid := low + (high-low)/2

		// Extract the raw []byte key stored at slot 'mid' in our flat page.
		// Under the hood, node.GetKey(mid) reads the slot header, jumps to that byte offset
		// inside the 4096-byte slice, reads the key length, and returns the actual key bytes.
		midKey := node.GetKey(mid)

		// Compare the raw bytes of the key we just pulled from the disk page against our search key.
		// bytes.Compare returns:
		//    0 if midKey == key (exact match!)
		//   -1 if midKey < key  (midpoint key comes alphabetically/lexicographically BEFORE our target)
		//   +1 if midKey > key  (midpoint key comes AFTER our target)
		cmp := bytes.Compare(midKey, key)

		// If the midpoint key is greater than OR equal to our target key:
		if cmp >= 0 {
			// This is a valid candidate! We record this slot index because it is either an
			// exact match, or it's the first key that is larger than our search key.
			found = mid

			// CRITICAL BUG PREVENTER: Because 'high' and 'mid' are unsigned integers (uint16),
			// they CANNOT be negative! In math, 0 - 1 = -1. But in uint16, 0 - 1 underflows
			// and wraps around to 65,535! That would cause an infinite loop or an out-of-bounds crash.
			// If we are already at index 0, we can't search any further to the left, so we break immediately.
			if mid == 0 {
				break
			}

			// Even though we found a valid candidate, we keep searching the LEFT half of our remaining range
			// to see if there is an even earlier slot that is also >= our search key.
			// We want to guarantee we find the *first* occurrence!
			high = mid - 1
		} else {
			// cmp < 0 means midKey < key: The midpoint key is too small!
			// Therefore, our target key MUST be in the RIGHT half of our remaining range.
			// We discard the current midpoint and everything to its left.
			low = mid + 1
		}
	}

	return found
}

func LeafInsert(new_p Page, old_p Page, index uint16, key []byte, val []byte) {
	new_p.SetHeader(BNODE_LEAF, old_p.NKeys())
	// TODO
}

// All this function is doing is to insert the KV at the right spot inside a single Page
// When you want to insert a brand-new KV pair into a leaf page:
// 1. You run NodeLookupLE on that page to find the exact slot where your new key belongs.
// 2. If the slot is already taken by a larger key, the database shifts all the existing bytes to the right to open up a gap.
// 3. Finally, it calls NodeAppendKV to drop your new key into that open gap, preserving the sorted order.
func NodeAppendKV(new_p Page, index uint16, pointer uint64, key []byte, val []byte) {
	new_p.SetPtr(index, pointer) // Set the page pointer to the index

	pos := new_p.KVPos(index) // get the KV position at the index

	keyLen := uint16(len(key))
	valLen := uint16(len(val))

	new_p.WriteUint16(pos, keyLen)   // Write the key length to the first 2 bytes at the KV starting position
	new_p.WriteUint16(pos+2, valLen) // Write the value length to the next 2 bytes at the KV starting position

	new_p.Write(pos+4, key)        // Write the key to the starting position of the data section in the page which would be kv start -> 2 bytes (key len) -> 2 bytes (val len) -> actual key
	new_p.Write(pos+4+keyLen, val) // Write the value to the starting position of the data section in the page which would be kv start -> 2 bytes (key len) -> 2 bytes (val len) -> actual key length -> actual value

	new_p.SetOffset(index+1, new_p.GetOffset(index)+4+keyLen+valLen) // Set the offset of the page to the end of this value
}

func main() {
	node1max := HEADER + B_TREE_POINTER_SIZE + B_TREE_OFFSET_SIZE + 4 + B_TREE_MAX_KEY_SIZE + B_TREE_MAX_VAL_SIZE
	fmt.Println(node1max)
	Assert(node1max > B_TREE_PAGE_SIZE, "Error: Max node size is larger than Page size")
}
