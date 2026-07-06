- A B-tree is roughly a balanced n-ary tree
- Can be queried and updated in O(log(n))
  - Can also be range-queried
## Advantages of using b-trees over binary trees
### Less space overhead
- Every leaf in a binary tree is reached via a pointer from the parent node and the parent may also have a parent -> each leaf node requires 1-2 pointers on average
- In a b-tree multiple leaf nodes can share one parent and makes it shorter and use less space
### Faster in memory
- b-trees are faster than binary trees 
### Less disk I/O
- Shorter than binary trees -> Less disk ops
- The min disk I/O is the memory page size -> usually 4096 B
  - The OS will fill up the entire page even if the info is smaller than 4K
  - It's ideal to fill up the entire page with a single Node because of this

## The Intuitions of the B-Tree and BST
- Keeping a tree in good shape after inserting or removing keys is what “balancing” means. 
- Each node of a B-tree contains multiple keys and multiple links to its children
- When looking up a key in a node, all keys are used to decide the next child node
- The height of all B-tree leaf nodes is the same -> a B-tree is balanced by the size of the nodes:
  - If a node is too large to fit on one page, it is split into two nodes
    - This will increase the size of the parent node and possibly increase the height of the tree if the root node was split
  - If a node is too small, try merging it with a sibling
  [  1,         4,       9]
    /           |        \
    v           v         v
  [1, 2, 3] [4, 6]   [9, 11, 12]

### B-tree and Nested Arrays
[[1,2,3], [4,6], [9,11,12]]
- Queries can be done by bisection
- Updating is `O(n)` which is not great
  - We can split the array into smaller sub-arrays to update
  - Usually split into `sqrt(n)` parts
  - Each part contains `sqrt(n)` keys on average
- To query a key, we must first determine which part contains the key
  - Bisecting on the `sqrt(n)` parts is `O(log(n))`
  - This can be done in `O(log(n))` any number of times
  - This improves querying by `O(sqrt(n))`

## B-Tree Operations
- Assume we're using a variant of a B-tree called a B| tree
- **Querying** a B-tree is the same as querying a BST.
### Updating
- Key insertion starts at a leaf
- A leaf is just a **sorted** list of keys

- If inserting a key exceeds the page size, we need to split the leaf node into 2 nodes
  - Each split node contains half the keys
  - The parent node replaces the old pointer and key with the new pointers and keys
     parent           parent
     / | \     =>    / | | \
    L1 L2 L6       L1 L3 L4 L6
  - After the root node is split, a new root node is added. This is how a B-tree grows
                          new_root
                            / \
      root                 N1 N2
      / | \       =>       / | | \
    L1 L2 L6             L1 L3 L4 L6
- A node consists of
  - A list of pointers to its children
  - A list of keys paired with the pointer list
### Deleting
- Opposite of insertion
- Nodes are never empty coz a small node will get merged into its left or right sibling
- When a non-leaf root is reduced to a single key, the root can be replaced by its sole child
  - This is how a B-tree shrinks

- B-trees are also **immutable**
  - When inserting a key into a leaf node, do not modify the node in place, instead, create a new node with all the keys from the to-be-updated node and the new key
    - Now the parent node must also be updated to point to the new node
    - Likewise, the parent node is duplicated with the new pointer
  - Avoids data corruption
  - Readers can operate concurrently with writers 

# The Complete Master Layout of a B+ Tree Page (4096 Bytes)
Byte 0       4                          4 + (8 * N)                 4 + (8 * N) + (2 * N)       Byte 4096
+------------+--------------------------+---------------------------+---------------------------+----------------------+
|   HEADER   |   CHILD POINTER ARRAY    |       OFFSET ARRAY        |       ... FREE SPACE ...  |  KV DATA             |
+------------+--------------------------+---------------------------+---------------------------+----------------------+
|  4 Bytes   |   8 Bytes  ×  node.NKeys()  |   2 Bytes  ×  node.NKeys()  |   Shrinks as you add KVs  | Packed KVs      |
+------------+--------------------------+---------------------------+--------------------------------+-----------------+
1. The Header (Fixed: 4 Bytes)
   - This is the identity card of the page. 
   - It always occupies bytes 0, 1, 2, and 3.
   - Bytes 0–1 (2 Bytes): The type of node (BNODE_NODE or BNODE_LEAF).
   - Bytes 2–3 (2 Bytes): The number of keys currently stored on this page (node.NKeys()).
2. The Child Pointer Array (Variable Size)
   - This section only matters if the node is an internal node (BNODE_NODE). 
   - It stores 8-byte numbers (uint64) pointing to other page IDs.
   - Starts at: Byte 4 (right after the header).
   - Ends at: 4 + (8 * node.NKeys()).
3. The Offset Array (Variable Size)
   - Stores 2-byte numbers (uint16).
   - Starts at: Exactly where the Child Pointers end.
   - Math Formula: OffsetPos() calculates positions inside this zone.
   - What it stores: It doesn't store starting positions; it stores the end position of each KV item relative to the beginning of the KV data section.
4. The Key-Value Data Section (Variable Size)
   - This is where your GetKey and GetVal methods do their work. 
   - It is located at the very end of the page.
   - Every time a new KV item is added, it is appended to the left of the previous item.
   - **The structure of ONE item**:[ 2 Bytes Key Len ] [ 2 Bytes Val Len ] [ Raw Key Bytes ] [ Raw Value Bytes ]