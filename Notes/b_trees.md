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
| Byte 0       | 4                          | 4 + (8 * N)                 | 4 + (8 * N) + (2 * N) |       Byte 4096 |
| ------------ | -------------------------- | --------------------------- | --------------------------- | ---------------------- | 
|   HEADER   |   CHILD POINTER ARRAY    |       OFFSET ARRAY        |       ... FREE SPACE ...  |  KV DATA             |
|  4 Bytes   |   8 Bytes  ×  node.NKeys()  |   2 Bytes  ×  node.NKeys()  |   Shrinks as you add KVs  | Packed KVs      |

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

# Inserting data into a single Page
1.Find the Insertion Index:
- The high-level function calls `NodeLookupLE(old_node, new_key)`
- Let's say it returns `index` = 2
2.Allocate a New Page: 
- It creates a brand new, empty 4096-byte slice in memory: `new_p := make([]byte, 4096)`
3.Copy the Left Side:
- It loops from 0 to 1 (everything before the insertion point). 
- For each existing key in `old_node`, it reads it and calls `NodeAppendKV` to write it exactly as-is into `new_p`
4.Drop in the New Key:
- It reaches `index` = 2. 
- It calls NodeAppendKV to write your brand new KV pair into `new_p`. 
- This is where `SetOffset` updates the header so the next step knows where to resume writing.
5.Copy the Right Side:
- It loops through the rest of the keys in old_node (from index 2 to the end). 
- For each one, it reads it from the old node and calls `NodeAppendKV` to append it into `new_p`. 
- Because we inserted our new key in the previous step, all these older keys are naturally shifted one index to the right in the new page

# A Real Insert Example
Imagine your Old Page has 4 keys: `[10, 20, 30, 40]` (at slots `0, 1, 2, 3`).
You want to insert the key `25`. Binary search tells us `25` belongs at `slot #2`.
Here is how we call `NodeAppendRange` to build the New Page:
1. Copy the Left Side (Keys smaller than 25)
We want to copy keys 10 and 20 (slots 0 and 1 from the old page) into slots 0 and 1 of the new page.
   - NodeAppendRange(new, old, dstNew=0, srcOld=0, n=2)
   - Translation: "Start at slot 0 of the old page, start at slot 0 of the new page, and copy 2 items."
2. Insert the New Key
   - We call `NodeAppendKV(new, index=2, ...)` to drop our new key 25 directly into slot #2 of the new page.
3. Copy the Right Side (Keys larger than 25)
- We still need to copy keys 30 and 40 over. In the old page, they were at slots 2 and 3. 
- But in our new page, slot #2 is now taken by our new key
- Therefore, they must be pasted into slots 3 and 4.
  - `NodeAppendRange(new, old, dstNew=3, srcOld=2, n=2)`
  - Translation: "Start reading from slot 2 of the old page, but paste them starting at slot 3 of the new page. Copy 2 items."

- Notice how dstNew=3 and srcOld=2 are different in that final step
- That difference of 1 is exactly what shifts the older keys to the right in the new page, making room for the key we just inserted without ever modifying the old page in place.

# Difference between an internal node and a leaf node
| Attribute |	Internal Node (Branch / Routing) |	Leaf Node (Data Warehouse)|
| -------- | -------- | -------- |
| Tree Position | Top and middle of the B-tree | Very bottom level of the B-tree |
| What it stores | Keys + Child Page Pointers | Keys + Actual Data Values |
| Purpose |	Directs searches down to the right child | Holds the actual data payload you want to retrieve |
| Search Action |	"Your key is smaller than X, go to Page #4" |	"Here is the value for your key: 'my_value'" |

# Rules of the Tree Levels
## Are ALL leaf nodes at the lowest level?
Yes. By definition, a B-tree (and specifically a B+ tree, which is what most databases use) is perfectly balanced. Every single leaf node is exactly the same distance from the root. The actual data values only exist on this bottom floor.

## Is there only one level of internal nodes?
No. Depending on how much data you have, there can be zero, one, or multiple levels of internal nodes. A small database might have Root -> Leaves (1 internal level). A massive 100 GB database might have Root -> Internal L1 -> Internal L2 -> Internal L3 -> Leaves.

## Is the root an internal node?
Usually yes, BUT not always. The Root is a chameleon.
When you create a brand-new database and insert your first row, the database only has one single page (Page #0).
In a 1-page database, the Root is the Leaf Node. It holds both the routing keys and the actual data values.

# How Trees Grow: Page Splitting
Because our nodes are strictly locked to a 4096-byte limit, what happens when you try to insert a row into a Leaf Node that is already full?
   - The database performs a **Page Split**.
1.The Leaf Node Splits:
  - When Leaf Page A is full (4096 bytes), the database allocates a brand-new 4KB page (Leaf Page B). 
  - It takes half the KV pairs from Page A and moves them to Page B.
2.The Parent gets Updated:The database must tell the parent Internal Node about this new Page B
  - It takes the middle key from the split and pushes it up into the parent Internal Node, along with a pointer to Page B.
3.The Ripple Effect (Internal Splits):
  - If that parent Internal Node is also completely full of child pointers, it must split too
  - It splits into two Internal Nodes, and pushes its middle key up to the next parent.

# Adding a New Level: The Root Split
- The only way a B-tree gets taller is when the Root Node gets full and splits.
- Imagine your database has grown to three levels: Root (Internal) -> Internal -> Leaves.
- Eventually, you insert a row at the bottom. 
  - That leaf splits. 
  - It pushes a key up. 
  - The parent internal node is full, so it splits. 
  - It pushes a key up to the Root.
- But what if the Root is full?
  - The Root page (Page #0) splits its contents into two brand-new pages (e.g., Page #50 and Page #51). 
  - These two new pages are now on the level below the root.Page #0 (the Root) is completely wiped clean.
  - The database writes exactly one key and two pointers into Page #0: pointing to Page #50 and Page #51.
- The tree just got one level taller. The root didn't move downward; it split its contents, pushed them down, and stayed at the top to act as the new boss of those two nodes.

**Because the tree always grows by splitting the root and adding a level above the rest of the data, all leaf nodes remain perfectly aligned at the exact same depth!**

Imagine our 4KB pages can only hold 4 keys before filling up. 
1.Initial State: A Full Root Page: 
- Database currently contains only 1 page (Page #0).
- When your database starts, Page #0 acts as both the Root and a Leaf node. 
- It holds both keys and values. Right now, it is completely full with 4 keys:
  
```
Page #0 (Type: LEAF | Pointers: None)
+-------------------------------------------------------+
|  [Key: 10]  |  [Key: 20]  |  [Key: 30]  |  [Key: 40]  |
+-------------------------------------------------------+
```

- You attempt to run INSERT [Key: 25]. Because there is no room left in the 4096-byte array of Page #0, the database must trigger a root split

2.Allocate Two Brand New Pages on Disk: Creating the new lower level (Level 0)
- Instead of growing downward, the database allocates two brand-new blank 4KB pages at the end of your database file: Page #1 and Page #2
- It copies the left half of the sorted data into Page #1, and the right half (including our new key 25) into Page #2
- Both of these new pages are formatted as Leaf Nodes:

```
Page #1 (Type: LEAF)                Page #2 (Type: LEAF)
+---------------------------+       +-----------------------------------------+
|  [Key: 10]  |  [Key: 20]  |       |  [Key: 25]  |  [Key: 30]  |  [Key: 40]  |
+---------------------------+       +-----------------------------------------+
```

3.Wipe Page #0 and Convert to Internal Node: The tree adds a level above the leaves
- Now that the data is safely copied into Pages #1 and #2, the database completely wipes Page #0 clean.
- It flips Page #0's header flag from LEAF to INTERNAL. 
- It takes the first key of the right child (25) and promotes it into Page #0 as a routing boundary, along with pointers to our two new leaf pages:            
```       
                      Page #0 (Type: INTERNAL / ROOT)
                   +---------------------------------------+
                   | Ptr: Page #1 | Key: 25 | Ptr: Page #2 |
                   +---------------------------------------+
                                  /              \
           Key < 25 goes Left    /                \    Key >= 25 goes Right
                                v                  v
              Page #1 (Type: LEAF)                  Page #2 (Type: LEAF)
        +---------------------------+             +-----------------------------------------+
        |  [Key: 10]  |  [Key: 20]  |             |  [Key: 25]  |  [Key: 30]  |  [Key: 40]  |
        +---------------------------+             +-----------------------------------------+
```

- Notice how Page #0 never moved. It stayed right at the beginning of your database file (byte offset 0), but its role changed from storing data to routing traffic. 
- That is how database trees grow upward while keeping the root locked at a fixed location on disk.