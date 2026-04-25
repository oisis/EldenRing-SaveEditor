# 29 — DLC Black Tiles (Map Cover Layer)

## Overview

DLC "Shadow of the Erdtree" uses a "Hard Blackout" system — black tiles that completely
cover the DLC map area until the player physically explores it. This is separate from
Fog of War (FoW) and map piece visibility flags.

## Three-Layer Map System

| Layer | Name | Mechanism | Editor Support |
|-------|------|-----------|----------------|
| 1. Cover Layer | Black tiles | **BloodStain section** (afterRegs+0x0088..0x0110) | WIP — this spec |
| 2. Basic Topography | Grey sketch | Automatic (under layer 1) | n/a |
| 3. Detailed Bitmap | Colored map | Event flags 62xxx + Map Fragment items | ✅ working |

FoW (afterRegs+0x087E..0x10B0) is a separate system that adds grey fog overlay.
Base game has Cover Layer mostly transparent by default. DLC has full blackout.

## Binary Search Results (2025-04-25)

Systematically narrowed down the 117,342-byte section [afterRegs..EventFlags]:

| Test | Range (from afterRegs) | Size | Black tiles? |
|------|------------------------|------|-------------|
| 7 | Full section copy | 117,342 B | ✅ NO tiles |
| 8 | 0x10B1..0x1C639 (menuProfile) | 112,008 B | ❌ tiles present |
| 9 | 0x0000..0x087E + 0x1C639..end (A+D) | 3,235 B | ✅ NO tiles |
| 10 | 0x0000..0x087E (A only) | 2,174 B | ✅ NO tiles |
| 11 | 0x0000..0x0440 (A1) | 1,088 B | ✅ NO tiles |
| 12 | 0x0000..0x0220 (A1a) | 544 B | ✅ NO tiles |
| 13 | 0x0000..0x0110 | 272 B | ✅ NO tiles |
| 14 | 0x0000..0x0088 | 136 B | ❌ tiles present |
| 15 | 0x0088..0x0110 | 136 B | ✅ NO tiles |
| 16 | 0x0088..0x00CC | 68 B | ❌ tiles in DLC only |
| 17 | 0x00CC..0x0110 | 68 B | ❌ partial tiles + FoW artifact |
| 18 | 0x0085..0x0110 zeroed | 139 B | ❌ tiles present (zeroing doesn't work) |

**Conclusion:** The critical data is at **afterRegs+0x0088..0x0110** (136 bytes),
within the BloodStain section. Both halves (0x0088..0x00CC and 0x00CC..0x0110) are
needed together — neither alone fully removes black tiles.

Zeroing the range does NOT work — the game needs **specific values** (coordinates/state)
from a save that has explored the DLC.

## Data Structure at afterRegs+0x0088..0x0110

This 136-byte range sits inside the BloodStain section (afterRegs+0x0075..0x10B1).
It appears to contain **two position/state records**:

### Record 1 (afterRegs+0x0085..0x00C4)
```
+0x0085: u32  — unknown (ref: 0x00000000)
+0x0089: u32  — unknown (ref: 0x00000000)  
+0x008D: f32  — X coordinate (ref: 9648.0 = DLC area)
+0x0091: f32  — Y coordinate (ref: 9123.8 = DLC area)
+0x0095: u8   — flag (ref: 0x01)
+0x0096..0x00C4: padding/extra data (mostly zeros in ref)
```

### Record 2 (afterRegs+0x00C5..0x00D5)
```
+0x00C5: f32  — X coordinate (ref: 3037.0)
+0x00C9: f32  — Y coordinate (ref: 1869.0)
+0x00CD: f32  — Z coordinate (ref: 7880.0)
+0x00D1: f32  — W coordinate (ref: 7803.0)
+0x00D5: u8   — flag (ref: 0x01, clean: 0x00)
```

**Key difference:** Ref has DLC-area coordinates; clean has base-game coordinates.
When ref values are copied, the game renders the map as if the player explored DLC.

## What Does NOT Control Black Tiles

Exhaustively tested and confirmed NOT responsible:

| Element | Tested | Result |
|---------|--------|--------|
| Event flags 62080-62084 | ✅ | Survive game load, no effect on tiles |
| Event flags 62xxx (269 flags) | ✅ | All survive, no effect |
| Discovery flags 60xxx, 61xxx | ✅ | Survive, no effect |
| FoW bitfield 0xFF | ✅ | Removes FoW only, not tiles |
| FoW bitfield 0x00 | ✅ | Adds FoW back, no tile effect |
| FoW bitfield from ref | ✅ | Same pattern, no tile effect |
| Map Fragment items | ✅ | Survive in inventory, no effect |
| DLC grace flags | ✅ | Survive, no effect |
| CsDlc byte[1] (0x30, 0x80) | ✅ | Game resets or modifies, no effect |
| Unlocked regions (395) | ✅ | Survive, no effect alone |
| menuProfile (112KB) | ✅ | No effect |
| gaItemsOther + tutorialData + ingameTimer (1KB) | ✅ | No effect |

## Section Layout Reference

```
afterRegs + 0x0000                = start (after unlocked regions)
afterRegs + 0x0029                = horse section end
afterRegs + 0x006D                = clearCount  
afterRegs + 0x0075                = bloodStain start
afterRegs + 0x0088..0x0110        = *** BLACK TILE DATA *** (136 bytes)
afterRegs + 0x087E                = FoW bitfield start
afterRegs + 0x10B0                = FoW bitfield end
afterRegs + 0x10B1                = menuProfile start
afterRegs + 0x10B1 + 0x1B588     = gaItemsOther
afterRegs + gaItemsOther + 0x40B  = tutorialData
afterRegs + tutorialData + 0x1A   = ingameTimer  
afterRegs + ingameTimer + 0       = EventFlags
```

## SOLUTION (Test 19 — confirmed working)

Synthetic values that remove DLC black tiles. Zero out 0x0085..0x0110, then write:

### Record 1 (afterRegs+0x008D)
```
+0x008D: f32 = 9648.0   (X — DLC map center)
+0x0091: f32 = 9124.0   (Y — DLC map center)
+0x0095: u8  = 0x01     (flag — "visited")
```

### Record 2 (afterRegs+0x00C5)
```
+0x00C5: f32 = 3037.0   (X)
+0x00C9: f32 = 1869.0   (Y)
+0x00CD: f32 = 7880.0   (Z)
+0x00D1: f32 = 7803.0   (W)
+0x00D5: u8  = 0x01     (flag — "visited")
```

These coordinates match the DLC overworld area. The game uses them to determine
which map tiles to render as "discovered". Slots 0 and 1 (both with DLC completed)
have identical values.

### Implementation

```go
func removeDLCBlackTiles(slot *SaveSlot) {
    storageEnd := slot.StorageBoxOffset + DynStorageBox
    gesturesOff := storageEnd + DynStorageToGestures
    regCount := readU32(slot.Data, gesturesOff)
    afterRegs := gesturesOff + 4 + regCount*4

    // Zero out bloodstain position data
    for i := afterRegs + 0x0085; i < afterRegs + 0x0110; i++ {
        slot.Data[i] = 0x00
    }

    // Record 1: DLC map center coordinates
    putF32(slot.Data, afterRegs+0x008D, 9648.0)
    putF32(slot.Data, afterRegs+0x0091, 9124.0)
    slot.Data[afterRegs+0x0095] = 0x01

    // Record 2: DLC area coordinates
    putF32(slot.Data, afterRegs+0x00C5, 3037.0)
    putF32(slot.Data, afterRegs+0x00C9, 1869.0)
    putF32(slot.Data, afterRegs+0x00CD, 7880.0)
    putF32(slot.Data, afterRegs+0x00D1, 7803.0)
    slot.Data[afterRegs+0x00D5] = 0x01
}
```

### Cross-slot verification

Slot 0 (base game completed, DLC completed) and slot 1 (DLC fully explored)
have **identical** values in this range. Fresh slot 4 has base-game coordinates
and -1.0 sentinel values.

## Remaining work

1. Fix partial FoW — zeroing 0x0085..0x0088 may overlap with nearby data
2. Determine if these coordinates affect base game map (they shouldn't — base game
   Cover Layer is transparent by default)
3. Investigate what the coordinates represent precisely (last bloodstain? respawn point?
   or map discovery anchor?)
4. Test with different DLC coordinates to see if specific values matter or just
   "any DLC-area coordinate" works
