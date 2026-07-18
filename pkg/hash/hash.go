// Package hash provides the hash table implementation used throughout GNU Make.
// Port of src/hash.c and src/hash.h
package hash

import (
	"fmt"
	"os"
	"sort"
)

// HashFunc is a primary or secondary hash function
type HashFunc func(key interface{}) uint64

// HashCmpFunc compares two keys for equality
type HashCmpFunc func(a, b interface{}) int

// HashMapFunc maps over items
type HashMapFunc func(item interface{})

// HashMapArgFunc maps over items with an argument
type HashMapArgFunc func(item interface{}, arg interface{})

// QsortCmp is a comparison function for sorting
type QsortCmp func(a, b interface{}) int

// HashTable is the hash table structure.
// Port of struct hash_table from hash.h
type HashTable struct {
	Vec        []unsafePointer
	HashFunc1  HashFunc
	HashFunc2  HashFunc
	Compare    HashCmpFunc
	Size       uint64 // total slots (power of 2)
	Capacity   uint64 // usable slots (loading-factor limited)
	Fill       uint64 // items in table
	EmptySlots uint64 // empty slots (not including deleted)
	Collisions uint64 // failed comparisons
	Lookups    uint64 // queries
	Rehashes   uint   // expansions
}

// unsafePointer wraps interface{} for the hash table entries
type unsafePointer struct {
	ptr interface{}
}

var hashDeletedItem = &struct{}{}

func IsVacant(item interface{}) bool {
	return item == nil || item == hashDeletedItem
}

// ComputePrime finds the largest prime <= n for hash table sizing.
func ComputePrime(n uint64) uint64 {
	// Use a simple approach: go to next power of 2
	// Make uses specific primes, but for Go we can use power-of-2 sizing
	if n < 3 {
		return 3
	}
	// Round up to next power of 2
	p := uint64(1)
	for p < n {
		p <<= 1
	}
	return p
}

// Init initializes a hash table.
// Port of hash_init() from hash.c
func Init(ht *HashTable, size uint64,
	hashFunc1 HashFunc, hashFunc2 HashFunc,
	hashCmp HashCmpFunc) {

	if ht == nil {
		panic("hash_init: HashTable is nil")
	}

	// enforce user-defined size
	if size < 3 {
		size = 3
	}

	ht.Size = ComputePrime(size)
	ht.Capacity = ht.Size * 3 / 4 // max loading factor ~75%
	ht.Vec = make([]unsafePointer, ht.Size)
	for i := range ht.Vec {
		ht.Vec[i] = unsafePointer{}
	}

	ht.HashFunc1 = hashFunc1
	ht.HashFunc2 = hashFunc2
	ht.Compare = hashCmp
	ht.Fill = 0
	ht.Collisions = 0
	ht.Lookups = 0
	ht.Rehashes = 0
	ht.EmptySlots = ht.Size
}

// roundUpToPowerOf2 rounds n up to the nearest power of 2
func roundUpToPowerOf2(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	p := n - 1
	p |= p >> 1
	p |= p >> 2
	p |= p >> 4
	p |= p >> 8
	p |= p >> 16
	p |= p >> 32
	return p + 1
}

// FindSlot returns a pointer to the slot for the given key.
// Port of hash_find_slot() from hash.c
func FindSlot(ht *HashTable, key interface{}) *unsafePointer {
	hash1 := ht.HashFunc1(key)
	hash2 := ht.HashFunc2(key)

	if ht.Size == 0 {
		return nil
	}

	ht.Lookups++

	step0 := hash1 & (ht.Size - 1)
	step1 := hash2 & (ht.Size - 1)

	if step1 == 0 {
		step1 = 1
	}

	for {
		slot := &ht.Vec[step0]
		if IsVacant(slot.ptr) {
			return slot
		}
		if ht.Compare(slot.ptr, key) == 0 {
			return slot
		}
		ht.Collisions++
		step0 += step1
		step0 &= (ht.Size - 1)
	}
}

// FindItem returns the item matching the key, or nil.
// Port of hash_find_item() from hash.c
func FindItem(ht *HashTable, key interface{}) interface{} {
	slot := FindSlot(ht, key)
	if slot == nil || IsVacant(slot.ptr) {
		return nil
	}
	return slot.ptr
}

// Insert inserts an item into the hash table.
// Port of hash_insert() from hash.c
func Insert(ht *HashTable, item interface{}) interface{} {
	if ht.Fill >= ht.Capacity {
		grow(ht)
	}

	slot := FindSlot(ht, item)
	if slot == nil || !IsVacant(slot.ptr) {
		return slot.ptr // already exists
	}

	ht.Fill++
	slot.ptr = item
	if IsVacant(slot.ptr) {
		ht.EmptySlots--
	}

	return item
}

// InsertAt inserts an item at a specific slot.
// Port of hash_insert_at() from hash.c
func InsertAt(ht *HashTable, item interface{}, slot *unsafePointer) interface{} {
	existing := slot.ptr
	if !IsVacant(existing) {
		return existing
	}

	if ht.Fill >= ht.Capacity {
		grow(ht)
		slot = FindSlot(ht, item)
	}

	ht.Fill++
	slot.ptr = item
	if IsVacant(existing) {
		ht.EmptySlots--
	}

	return item
}

// Delete removes an item from the hash table.
// Port of hash_delete() from hash.c
func Delete(ht *HashTable, item interface{}) interface{} {
	slot := FindSlot(ht, item)
	return DeleteAt(ht, slot)
}

// DeleteAt removes the item at a slot.
// Port of hash_delete_at() from hash.c
func DeleteAt(ht *HashTable, slot *unsafePointer) interface{} {
	if slot == nil || IsVacant(slot.ptr) {
		return nil
	}

	item := slot.ptr
	slot.ptr = hashDeletedItem
	ht.Fill--
	ht.EmptySlots-- // deleted-item occupies space
	return item
}

// DeleteItems frees all items from the hash table.
// Port of hash_delete_items from hash.c
func DeleteItems(ht *HashTable) {
	for i := range ht.Vec {
		ht.Vec[i] = unsafePointer{}
	}
	ht.Fill = 0
	ht.EmptySlots = ht.Size
}

// FreeItems frees items and the table.
// Port of hash_free_items / hash_free from hash.c
func FreeItems(ht *HashTable) {
	DeleteItems(ht)
}

// Free frees the hash table.
func Free(ht *HashTable, freeItems bool) {
	if freeItems {
		FreeItems(ht)
	}
	ht.Vec = nil
	ht.Size = 0
	ht.Capacity = 0
	ht.Fill = 0
	ht.EmptySlots = 0
}

// Map calls fn for each item in the table.
// Port of hash_map() from hash.c
func Map(ht *HashTable, fn HashMapFunc) {
	if ht == nil {
		return
	}
	for _, slot := range ht.Vec {
		if !IsVacant(slot.ptr) {
			fn(slot.ptr)
		}
	}
}

// MapArg calls fn with an argument for each item.
// Port of hash_map_arg() from hash.c
func MapArg(ht *HashTable, fn HashMapArgFunc, arg interface{}) {
	if ht == nil {
		return
	}
	for _, slot := range ht.Vec {
		if !IsVacant(slot.ptr) {
			fn(slot.ptr, arg)
		}
	}
}

// PrintStats prints hash table statistics.
// Port of hash_print_stats() from hash.c
func PrintStats(ht *HashTable, name string) {
	if ht == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%s: size=%d, fill=%d, capacity=%d, empty=%d, collisions=%d, lookups=%d, rehashes=%d\n",
		name, ht.Size, ht.Fill, ht.Capacity, ht.EmptySlots, ht.Collisions, ht.Lookups, ht.Rehashes)
}

// Dump returns all items in the table as a sorted slice.
// Port of hash_dump() from hash.c
func Dump(ht *HashTable, cmp QsortCmp) []interface{} {
	if ht == nil || ht.Fill == 0 {
		return nil
	}

	result := make([]interface{}, 0, ht.Fill)
	for _, slot := range ht.Vec {
		if !IsVacant(slot.ptr) {
			result = append(result, slot.ptr)
		}
	}

	if cmp != nil {
		sort.Slice(result, func(i, j int) bool {
			return cmp(result[i], result[j]) < 0
		})
	}

	return result
}

// grow expands the hash table when it becomes too full.
// Port of hash_rehash() from hash.c
func grow(ht *HashTable) {
	oldVec := ht.Vec
	oldSize := ht.Size

	newSize := oldSize * 2
	if newSize < 3 {
		newSize = 3
	}

	ht.Vec = make([]unsafePointer, newSize)
	for i := range ht.Vec {
		ht.Vec[i] = unsafePointer{}
	}
	ht.Size = newSize
	ht.Capacity = newSize * 3 / 4
	ht.EmptySlots = newSize
	ht.Fill = 0
	ht.Collisions = 0
	ht.Rehashes++

	for _, slot := range oldVec {
		if !IsVacant(slot.ptr) {
			Insert(ht, slot.ptr)
		}
	}
}

// ——————————————————— jhash ———————————————————
// Port of jhash() from hash.c - Jenkins hash functions

func Jhash(key []byte, n int) uint {
	// Jenkins one-at-a-time hash
	hash := uint(0)
	for i := 0; i < n; i++ {
		hash += uint(key[i])
		hash += hash << 10
		hash ^= hash >> 6
	}
	hash += hash << 3
	hash ^= hash >> 11
	hash += hash << 15
	return hash
}

func JhashString(key string) uint {
	return Jhash([]byte(key), len(key))
}

// ——————————————————— String hash helpers ———————————————————
// These match the macros in hash.h

func StringHash1(key string) uint64 {
	return uint64(JhashString(key))
}

func StringHash2(key string) uint64 {
	_ = key // no second hash needed
	return 0
}

func StringCompare(a, b string) int {
	if a == b {
		return 0
	}
	if a < b {
		return -1
	}
	return 1
}
