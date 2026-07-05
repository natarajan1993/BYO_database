package storage

import (
	. "BYO_database/internal/btree"
	. "BYO_database/internal/utils"
	"encoding/binary"
)

type Page struct {
	data []byte
}

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

// Get the offset poisition of an index
// Each offset is the end of the KV pair relative to the start of the 1st KV.
// The start offset of the 1st KV is just 0, so we use the end offset instead, which is the start offset of the next KV.
func OffsetPos(node Page, index uint16) uint16 {
	Assert(index >= 1 && index <= node.NKeys(), "Error: invalid index for position offset")
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

}
