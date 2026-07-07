package storage

import (
	. "BYO_database/internal/utils"
	"encoding/binary"
)

type Page struct {
	data []byte
}

// What is the Offset Array Used For?
// In a B+Tree page layout, the Offset Array is a table of contents for the data inside that specific node.
// A node stores variable-length items (like keys and values) packed tightly at the end of the byte page.
// Because keys are different sizes (e.g., "apple" is 5 bytes, "hippopotamus" is 12 bytes), you cannot predict exactly where the 2nd or 3rd key starts just by doing math.
// It is storing a location number (a map coordinate), not the user's actual keys or values.
// The offset array solves this:
// The Index: Refers to the item number (Item 0, Item 1, Item 2).
// The Value (Offset): Stores the exact byte position where that item's data actually begins inside the page.

// | type       | number_of_keys | pointers             | offsets            | key-values | unused |
// | 2 Bytes   | 2 Bytes         | number_of_keys * 8B | number_of_keys * 2B | ...        |        |
// ---------------------------------
// | key_len | value_len | key | val |
// | 2 Bytes | 2 Bytes  | ... | ... |
// The format packs everything back to back
func (node Page) BType() uint16 {
	return binary.LittleEndian.Uint16(node.data[0:2])
}

func (node Page) NKeys() uint16 {
	return binary.LittleEndian.Uint16(node.data[2:4])
}

func (node Page) SetHeader(BType uint16, NKeys uint16) {
	binary.LittleEndian.PutUint16(node.data[0:2], BType)
	binary.LittleEndian.PutUint16(node.data[2:4], NKeys)
}

func (node Page) GetPointer(index uint16) uint64 {
	Assert(index < node.NKeys(), "Error: Index exceeds total keys in node")
	pos := HEADER + B_TREE_POINTER_SIZE*index
	return binary.LittleEndian.Uint64(node.data[pos:])
}

func (node Page) SetPtr(index uint16, val uint64) {
	Assert(index >= node.NKeys(), "Error: Index exceeds total keys in node")
	pos := HEADER + B_TREE_POINTER_SIZE*index
	binary.LittleEndian.PutUint64(node.data[pos:], val)
}

// Get the offset position of an index
// Each offset is the end of the KV pair relative to the start of the 1st KV.
// The start offset of the 1st KV is just 0, so we use the end offset instead, which is the start offset of the next KV.
func OffsetPos(node Page, index uint16) uint16 {
	Assert(index >= 1 && index <= node.NKeys(), "Error: invalid index for position offset")
	// HEADER: Skip the first 4 bytes (node type, number of keys).
	// B_TREE_POINTER_SIZE * node.NKeys(): Skip the Child Pointer Array
	// In an internal B+Tree node, every key has a corresponding pointer (8 bytes each) to a child page. The formula skips past all of them first
	// B_TREE_OFFSET_SIZE * ...: Puts us in Offset Array section and jumps to the exact slot for your specific index
	return HEADER + B_TREE_POINTER_SIZE*node.NKeys() + B_TREE_OFFSET_SIZE*(index+1)
}

func (node Page) GetOffset(index uint16) uint16 {
	if index == 0 {
		return 0
	}

	return binary.LittleEndian.Uint16(node.data[OffsetPos(node, index):])
}

// SetOffset writes a 2-byte offset into the page metadata section
func (node Page) SetOffset(index uint16, offset uint16) {
	pos := HEADER + (index * B_TREE_OFFSET_SIZE)
	binary.LittleEndian.PutUint16(node.data[pos:pos+2], offset)
}

// To find where KV Pair #1 starts: You ask for the end offset of KV Pair #0.
// Therefore, you call node.GetOffset(0).
// To find where KV Pair #2 starts: You ask for the end offset of KV Pair #1.
// Therefore, you call node.GetOffset(1).
func (node Page) KVPos(index uint16) uint16 {
	Assert(index <= node.NKeys(), "Error KVPos: Index out of bounds: Exceeds total keys in Page")
	// HEADER: Skips the first 4 bytes of metadata
	// B_TREE_POINTER_SIZE * node.NKeys(): Skips past the entire array of Child Pointers
	// B_TREE_OFFSET_SIZE * node.NKeys(): Skips past the entire Offset Array
	// node.GetOffset(index): Now that the math has reached the exact starting boundary of the KV Data section, it adds the relative offset for your specific item.
	return HEADER + B_TREE_POINTER_SIZE*node.NKeys() + B_TREE_OFFSET_SIZE*node.NKeys() + node.GetOffset(index)
}

//nolint:all Byte Position:  [ pos ]   [ pos+2 ]  [ pos+4 ]             [ pos+4+klen ]
//nolint:all                +---------+---------+---------------------+---------------------+
//nolint:all Data Layout:   | Key Len | Val Len |     ACTUAL KEY      |    ACTUAL VALUE     |
//nolint:all                +---------+---------+---------------------+---------------------+
//nolint:all Size in Bytes: | 2 bytes | 2 bytes |     'klen' bytes    |    'vlen' bytes     |
//nolint:all                +---------+---------+---------------------+---------------------+
//nolint:all                                    ^                     ^
//nolint:all                                    |                     |
//nolint:all                                    [--- node.GetKey() ---]
//nolint:all                                     Extracts just this!
func (node Page) GetKey(index uint16) []byte {
	Assert(index < node.NKeys(), "Error GetKey: Index out of bounds: Exceeds total keys in Page")
	pos := node.KVPos(index)                            //  get the exact starting byte index of the KV item in memory
	klen := binary.LittleEndian.Uint16(node.data[pos:]) //  the first 2 bytes always store the length of the key. Next 2 bytes store the length of the value
	// node.data[pos+4:] means get to the start of the key position and get the data all the way to the end of the page
	// immediately applying [:klen] will cut it off at the length of the key
	return node.data[pos+4:][:klen]
}

func (node Page) GetVal(index uint16) []byte {
	Assert(index < node.NKeys(), "Error GetVal: Index out of bounds: Exceeds total keys in Page")
	pos := node.KVPos(index) //  get the exact starting byte index of the KV item in memory
	klen := binary.LittleEndian.Uint16(node.data[pos:])
	vlen := binary.LittleEndian.Uint16(node.data[pos+2:])
	return node.data[pos+4+klen:][:vlen]
}

// returns the node size (used space) with an off-by-one lookup
func (node Page) NBytes() uint16 {
	return node.KVPos(node.NKeys())
}
