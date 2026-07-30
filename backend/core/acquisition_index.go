package core

// InventoryIndexCollisionKey identifies one collision group. Native records at
// or below InvEquipReservedMax use their exact raw Index unless the same bucket
// also contains an editor-range record. Buckets containing any Index above the
// native range use Index>>1 for every member, including the 432/433 boundary.
//
// Fields stay private so callers cannot reimplement the range policy.
type InventoryIndexCollisionKey struct {
	bucketed bool
	value    uint32
}

// Matches reports whether index belongs to this collision group.
func (k InventoryIndexCollisionKey) Matches(index uint32) bool {
	if k.bucketed {
		return index>>1 == k.value
	}
	return index == k.value
}

func (k InventoryIndexCollisionKey) issueField() (field string, value uint32) {
	if k.bucketed {
		return "bucket", k.value
	}
	return "index", k.value
}

// InventoryIndexCollisionSet is the single source of truth for classifying
// acquisition-index collisions across Inventory.CommonItems and KeyItems.
//
// The complete set of existing indices is supplied up front so a boundary
// bucket is classified consistently regardless of record order. Add returns
// the first conflicting raw index when a claim already exists.
type InventoryIndexCollisionSet struct {
	bucketed      map[uint32]bool
	firstByRaw    map[uint32]uint32
	firstByBucket map[uint32]uint32
}

func NewInventoryIndexCollisionSet(indices []uint32) *InventoryIndexCollisionSet {
	set := &InventoryIndexCollisionSet{
		bucketed:      make(map[uint32]bool),
		firstByRaw:    make(map[uint32]uint32),
		firstByBucket: make(map[uint32]uint32),
	}
	for _, index := range indices {
		if index > InvEquipReservedMax {
			set.bucketed[index>>1] = true
		}
	}
	return set
}

func (s *InventoryIndexCollisionSet) collisionKey(index uint32) InventoryIndexCollisionKey {
	bucket := index >> 1
	if index > InvEquipReservedMax || s.bucketed[bucket] {
		return InventoryIndexCollisionKey{bucketed: true, value: bucket}
	}
	return InventoryIndexCollisionKey{value: index}
}

// Add claims index and reports whether it conflicts with an earlier claim.
func (s *InventoryIndexCollisionSet) Add(index uint32) (InventoryIndexCollisionKey, uint32, bool) {
	key := s.collisionKey(index)
	if key.bucketed {
		if first, exists := s.firstByBucket[key.value]; exists {
			return key, first, true
		}
	} else if first, exists := s.firstByRaw[index]; exists {
		return key, first, true
	}

	if _, exists := s.firstByRaw[index]; !exists {
		s.firstByRaw[index] = index
	}
	bucket := index >> 1
	if _, exists := s.firstByBucket[bucket]; !exists {
		s.firstByBucket[bucket] = index
	}
	if index > InvEquipReservedMax {
		s.bucketed[bucket] = true
	}
	return key, 0, false
}

// Conflicts reports whether adding index would collide with an existing claim.
// A new editor-range index always checks the complete occupied bucket set, which
// prevents 433 from sharing bucket 216 with a native Index 432.
func (s *InventoryIndexCollisionSet) Conflicts(index uint32) bool {
	if index > InvEquipReservedMax {
		_, exists := s.firstByBucket[index>>1]
		return exists
	}
	key := s.collisionKey(index)
	if key.bucketed {
		_, exists := s.firstByBucket[key.value]
		return exists
	}
	_, exists := s.firstByRaw[index]
	return exists
}
