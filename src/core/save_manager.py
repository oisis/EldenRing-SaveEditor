import os
import shutil
from datetime import datetime
from .structures import SAVE_SLOT, PLAYER_GAME_DATA, GA_ITEM
from .crypto import decrypt_pc_save, encrypt_pc_save, calculate_sha256

class SaveManager:
    PS4_SLOT_SIZE = 0x280000
    PS4_FIRST_SLOT_OFFSET = 0x310
    NAME_OFFSET_IN_SLOT = 0xe2b5
    STATS_OFFSET_IN_SLOT = NAME_OFFSET_IN_SLOT - 88
    EVENT_FLAGS_OFFSET_IN_SLOT = 0x1bfaf0 
    
    # Inventory offsets
    GA_ITEMS_OFFSET_IN_SLOT = 0x20 # After ver (4) and map_id (4) and unk0 (0x18)
    GA_ITEM_DATA_OFFSET_IN_SLOT = 0x1bf190 # Offset for GaItemData (counts)
    
    def __init__(self, file_path):
        self.file_path = file_path
        self.data = None
        self.is_pc = False
        self.slots = []
        
    def load(self):
        with open(self.file_path, 'rb') as f:
            self.data = bytearray(f.read())
        self.is_pc = self.data.startswith(b"BND4")
        self._scan_slots()

    def _scan_slots(self):
        self.slots = []
        for i in range(10):
            offset = self.PS4_FIRST_SLOT_OFFSET + (i * self.PS4_SLOT_SIZE)
            if offset + self.PS4_SLOT_SIZE > len(self.data):
                break
            name_pos = offset + self.NAME_OFFSET_IN_SLOT
            name_bytes = self.data[name_pos : name_pos + 32]
            name = name_bytes.decode('utf-16le').strip('\x00')
            self.slots.append({
                "id": i,
                "offset": offset,
                "name": name if name else "Empty Slot",
                "active": bool(name)
            })

    def get_character_stats(self, slot_id):
        if slot_id >= len(self.slots): return None
        slot_offset = self.slots[slot_id]["offset"]
        stats_pos = slot_offset + self.STATS_OFFSET_IN_SLOT
        stats_data = self.data[stats_pos : stats_pos + 0x200]
        return PLAYER_GAME_DATA.parse(stats_data)

    def update_character_stats(self, slot_id, new_stats_dict):
        if slot_id >= len(self.slots): return False
        slot_offset = self.slots[slot_id]["offset"]
        stats_pos = slot_offset + self.STATS_OFFSET_IN_SLOT
        current_stats = self.get_character_stats(slot_id)
        for key, value in new_stats_dict.items():
            if hasattr(current_stats, key):
                setattr(current_stats, key, value)
        updated_bytes = PLAYER_GAME_DATA.build(current_stats)
        self.data[stats_pos : stats_pos + len(updated_bytes)] = updated_bytes
        self._scan_slots()
        return True

    def add_item(self, slot_id, item_id, quantity=1):
        """Adds a new item to the character's inventory."""
        if slot_id >= len(self.slots): return False
        
        slot_offset = self.slots[slot_id]["offset"]
        ga_items_pos = slot_offset + self.GA_ITEMS_OFFSET_IN_SLOT
        
        # 1. Find an empty slot in GA_ITEMS (5120 slots)
        # An empty slot has id 0
        found_idx = -1
        max_handle = 0x80000000 # Base handle for items
        
        for i in range(0x1400):
            item_pos = ga_items_pos + (i * 17) # GA_ITEM size is 17 bytes
            current_id = int.from_bytes(self.data[item_pos+4 : item_pos+8], 'little')
            current_handle = int.from_bytes(self.data[item_pos : item_pos+4], 'little')
            
            if current_id == 0 and found_idx == -1:
                found_idx = i
            if current_handle > max_handle and current_handle < 0xFFFFFFFF:
                max_handle = current_handle

        if found_idx == -1:
            print("Inventory full!")
            return False

        # 2. Create new item data
        new_handle = max_handle + 1
        new_item = {
            "handle": new_handle,
            "id": item_id,
            "data": {
                "unk2": -1,
                "unk3": -1,
                "aow_handle": 0xFFFFFFFF,
                "unk5": 0
            }
        }
        
        # 3. Write to data array
        item_bytes = GA_ITEM.build(new_item)
        write_pos = ga_items_pos + (found_idx * 17)
        self.data[write_pos : write_pos + 17] = item_bytes
        
        print(f"Added item {hex(item_id)} to slot {found_idx} with handle {hex(new_handle)}")
        return True

    def get_event_flag(self, slot_id, flag_id):
        if slot_id >= len(self.slots): return False
        byte_offset = flag_id // 8
        bit_mask = 1 << (flag_id % 8)
        slot_offset = self.slots[slot_id]["offset"]
        flag_pos = slot_offset + self.EVENT_FLAGS_OFFSET_IN_SLOT + byte_offset
        if flag_pos >= len(self.data): return False
        return bool(self.data[flag_pos] & bit_mask)

    def set_event_flag(self, slot_id, flag_id, active):
        if slot_id >= len(self.slots): return False
        byte_offset = flag_id // 8
        bit_mask = 1 << (flag_id % 8)
        slot_offset = self.slots[slot_id]["offset"]
        flag_pos = slot_offset + self.EVENT_FLAGS_OFFSET_IN_SLOT + byte_offset
        if flag_pos >= len(self.data): return False
        if active:
            self.data[flag_pos] |= bit_mask
        else:
            self.data[flag_pos] &= ~bit_mask
        return True

    def backup(self):
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        backup_path = f"{self.file_path}.{timestamp}.bak"
        shutil.copy2(self.file_path, backup_path)
        return backup_path

    def save(self, output_path=None):
        target = output_path or self.file_path
        self.backup()
        with open(target, 'wb') as f:
            f.write(self.data)
        print(f"File saved to: {target}")
