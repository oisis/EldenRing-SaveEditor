package data

import (
	_ "embed"
	"strconv"
	"strings"
	"sync"
)

//go:embed eventflag_bst.txt
var eventflagBSTRaw string

// EventFlagBST maps block numbers (flag_id / 1000) to BST positions.
// Loaded from embedded eventflag_bst.txt on first access.
// BST formula: byte = bst_pos*125 + (flag_id%1000)/8, bit = 7 - (flag_id%1000)%8
var EventFlagBST map[uint32]uint32

var bstOnce sync.Once

// LoadBST initializes the EventFlagBST map from the embedded data.
// Safe to call multiple times; only the first call does actual work.
func LoadBST() {
	bstOnce.Do(func() {
		EventFlagBST = make(map[uint32]uint32, 12000)
		for _, line := range strings.Split(eventflagBSTRaw, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ",", 2)
			if len(parts) != 2 {
				continue
			}
			block, err1 := strconv.ParseUint(parts[0], 10, 32)
			pos, err2 := strconv.ParseUint(parts[1], 10, 32)
			if err1 != nil || err2 != nil {
				continue
			}
			EventFlagBST[uint32(block)] = uint32(pos)
		}
	})
}

const (
	// BSTBlockSize is the number of bytes per BST block (1000 flags / 8 = 125 bytes).
	BSTBlockSize = 125
	// BSTFlagDivisor is the number of flags per BST block.
	BSTFlagDivisor = 1000
)
