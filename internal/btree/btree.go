package btree

import (
	"BYO_database/internal/storage"
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

// All this function is doing is to insert the KV at the right spot inside a single Page
// When you want to insert a brand-new KV pair into a leaf page:
// 1. You run NodeLookupLE on that page to find the exact slot where your new key belongs.
// 2. If the slot is already taken by a larger key, the database shifts all the existing bytes to the right to open up a gap.
// 3. Finally, it calls PageAppendKV to drop your new key into that open gap, preserving the sorted order.
func PageAppendKV(new_p Page, index uint16, pointer uint64, key []byte, val []byte) {
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

// PageAppendRange copies a contiguous block of n Key-Value pairs (along with their child pointers) from an old Page into a new Page.
// PageAppendRange works identically for both Leaf Nodes and Internal Nodes
// new_p Page (Destination Page): The 4096-byte page slice we are currently constructing. This is where the data is being pasted into.
// old Page (Source Page): The existing 4096-byte page slice we are reading from.
// dstNew uint16 (Destination Start Index): The logical slot number in the new page where we should begin pasting.
// For example, if dstNew = 0, we start writing at the very first slot of the new page.
// srcOld uint16 (Source Start Index): The logical slot number in the old page where we should begin reading.
// If srcOld = 2, we skip slots 0 and 1 of the old page and start grabbing data from slot 2.
// n uint16 (Count / Number of Items): The total number of sequential Key-Value slots to copy over.
// If n = 3, the loop will execute 3 times, copying slots srcOld, srcOld+1, and srcOld+2.
func PageAppendRange(new_p Page, old Page, dstNew uint16, srcOld uint16, n uint16) {
	for i := uint16(0); i < n; i++ {
		srcIndex := srcOld + i
		dstIndex := dstNew + i

		ptr := old.GetPointer(srcIndex)
		key := old.GetKey(srcIndex)
		val := old.GetVal(srcIndex)

		PageAppendKV(new_p, dstIndex, ptr, key, val) // We let the underlying PageAppendKV function do the heavy lifting of figuring out where the raw bytes actually live in each page
	}
}

// creates a new leaf page containing the original data plus the new key-value pair, preserving sorted order
// It implements a copy-on-write strategy by returning a new page rather than modifying the existing one.
func LeafInsert(new_p Page, old_p Page, index uint16, key []byte, val []byte) {
	new_p.SetHeader(BNODE_LEAF, old_p.NKeys()+1) // add 1 to the total keys for the key we are about to insert, and record that total as our new key count.
	PageAppendRange(new_p, old_p, 0, 0, index)   // copy over all existing keys that are strictly smaller than our new key
	// leaf nodes are at the very bottom of the tree—they don't have child pages, so the child pointer is always zero
	PageAppendKV(new_p, index, 0, key, val)                            // Append the new key to the new page.
	PageAppendRange(new_p, old_p, index+1, index, old_p.NKeys()-index) // copy over all existing keys that are larger than our new key
}

// Purpose: We are creating a new version of a parent node (new_p) that contains updated child pointers.
// In a B-tree, when a node (let's call it Node A) gets too full and splits into two new nodes (Node B and Node C), the Parent of Node A needs to be updated.
// The Parent is an Internal Node - its job is to route traffic.
// Right now, it has a pointer to Node A. After the split, it needs to get rid of the pointer to Node A and add two new pointers: one for B and one for C.
// Deletion: It removes the old, "full" child page pointer from the parent.
// Insertion: It inserts the new, "split" child pages into that same spot.
// Promotion: It adds the necessary "separator keys" so the parent knows how to route traffic between the new children.
func ReplaceChildNode(tree *BTree, new_p Page, old_p Page, index uint16, children ...Page) {
	child_count := uint16(len(children))                     // Get the new number of keys the parent will have
	new_p.SetHeader(BNODE_NODE, old_p.NKeys()+child_count-1) // Take the old key count, add the number of new children (child_count), and subtract 1 because we are removing the one child we are replacing.
	PageAppendRange(new_p, old_p, 0, 0, index)
	for i, child_page := range children {
		pointer := tree.new(child_page.Data()) // Save the child page to disk using the callback (tree.new). The disk address it returns becomes the new pointer in our internal node.
		key := child_page.GetKey(0)            // Take the first key of that child page. In B-trees, the first key of a child is the "separator key" used to decide if a search goes left or right.
		// index+uint16(i): We place these new pointers starting exactly where the old, bad pointer was.
		PageAppendKV(new_p, index+uint16(i), pointer, key, nil)
	}

	// index+inc: This is the new starting position in the destination page. If we inserted 2 children where 1 used to be, the remaining pointers have to be shifted "to the right."
	// index+1: This is the starting position in the source page (skipping the pointer we just replaced).
	// old_p.NKeys()-(index+1): This counts how many items are left to copy.
	PageAppendRange(new_p, old_p, index+child_count, index+1, old_p.NKeys()-(index+1)) // Copies everything in the parent that comes after the index we modified.
}

func FindSplitIndex(page Page) uint16 {
	totalBytes := uint16(HEADER)

	for i := uint16(0); i < page.NKeys(); i++ {
		kvSize := page.GetKVSize(i)

		if totalBytes+kvSize > B_TREE_PAGE_SIZE/2 {
			// Return the index of the KV where it exceeds 2048 bytes
			return i
		}

		totalBytes += kvSize
	}

	// Fallback: if the page isn't full enough to split,
	// or we reached the end, return the total count.
	return page.NKeys()
}

func PageSplitInTwo(left Page, right Page, old Page) []byte {
	// Determine the split point
	splitIndex := FindSplitIndex(old)

	// Initialize the headers for the new nodes
	left.SetHeader(old.BType(), splitIndex)
	right.SetHeader(old.BType(), old.NKeys()-splitIndex)

	// Copy the first half to the "left"
	PageAppendRange(left, old, 0, 0, splitIndex)
	// Copy the second half to the "right"
	PageAppendRange(right, old, 0, splitIndex, old.NKeys()-splitIndex)

	// If internal node, return the first key of the right node
	// to be pushed up to the parent.
	if old.BType() == BNODE_NODE {
		return right.GetKey(0)
	}
	return nil
}

// split a node if it's too big. the results are 1~3 nodes.
func PageSplitInThree(old Page) (uint16, [3]Page) {
	// 1. Initial check. If we don't need to split, then return the slice of 3 Page objects immediately
	if len(old.Data()) <= B_TREE_PAGE_SIZE {
		return 1, [3]Page{old}
	}

	// 2. Prepare for the first split
	left := storage.NewPage(make([]byte, 2*B_TREE_PAGE_SIZE))
	right := storage.NewPage(make([]byte, B_TREE_PAGE_SIZE))

	PageSplitInTwo(left, right, old) // Split

	// Return 2 Nodes
	if len(left.Data()) <= B_TREE_PAGE_SIZE {
		return 2, [3]Page{
			storage.NewPage(left.Data()[:B_TREE_PAGE_SIZE]),
			right,
		}
	}

	// 4. Second split logic: Split 'left' again
	secondLeftSplit := storage.NewPage(make([]byte, B_TREE_PAGE_SIZE))
	middle := storage.NewPage(make([]byte, B_TREE_PAGE_SIZE))

	// We treat the current 'left' as the new 'old' page
	PageSplitInTwo(secondLeftSplit, middle, left)

	// If it's still too big, we can't do anything
	Assert(len(secondLeftSplit.Data()) > B_TREE_PAGE_SIZE, "PageSplitInThree: secondLeftSplit page still too large!")

	// Return 3 pages: (secondLeftSplit, middle, right)
	return 3, [3]Page{secondLeftSplit, middle, right}
}

func main() {
	node1max := HEADER + B_TREE_POINTER_SIZE + B_TREE_OFFSET_SIZE + 4 + B_TREE_MAX_KEY_SIZE + B_TREE_MAX_VAL_SIZE
	fmt.Println(node1max)
	Assert(node1max > B_TREE_PAGE_SIZE, "Error: Max node size is larger than Page size")
}
