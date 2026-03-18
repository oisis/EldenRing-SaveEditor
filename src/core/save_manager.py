import os
import shutil
from datetime import datetime
from .structures import SAVE_SLOT, PLAYER_GAME_DATA
from .crypto import decrypt_pc_save, encrypt_pc_save, calculate_sha256

class SaveManager:
    PS4_SLOT_SIZE = 0x280000
    PS4_FIRST_SLOT_OFFSET = 0x310
    NAME_OFFSET_IN_SLOT = 0xe2b5
    STATS_OFFSET_IN_SLOT = NAME_OFFSET_IN_SLOT - 88 # Based on PLAYER_GAME_DATA structure

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
        """Parses and returns stats for a specific slot."""
        if slot_id >= len(self.slots): return None
        
        slot_offset = self.slots[slot_id]["offset"]
        stats_pos = slot_offset + self.STATS_OFFSET_IN_SLOT
        
        # Read enough bytes for PLAYER_GAME_DATA
        stats_data = self.data[stats_pos : stats_pos + 0x200]
        return PLAYER_GAME_DATA.parse(stats_data)

    def update_character_stats(self, slot_id, new_stats_dict):
        """Updates character stats in the bytearray."""
        if slot_id >= len(self.slots): return False
        
        slot_offset = self.slots[slot_id]["offset"]
        stats_pos = slot_offset + self.STATS_OFFSET_IN_SLOT
        
        # 1. Get current stats object to preserve unknown fields
        current_stats = self.get_character_stats(slot_id)
        
        # 2. Update fields from dictionary
        for key, value in new_stats_dict.items():
            if hasattr(current_stats, key):
                setattr(current_stats, key, value)
        
        # 3. Serialize back to bytes
        updated_bytes = PLAYER_GAME_DATA.build(current_stats)
        
        # 4. Patch the main data array
        self.data[stats_pos : stats_pos + len(updated_bytes)] = updated_bytes
        
        # 5. Refresh slot info (in case name changed)
        self._scan_slots()
        return True

    def backup(self):
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        backup_path = f"{self.file_path}.{timestamp}.bak"
        shutil.copy2(self.file_path, backup_path)
        return backup_path

    def clone_character(self, source_id, target_id, new_name=None):
        if source_id >= len(self.slots) or target_id >= len(self.slots):
            return False
        src_off = self.slots[source_id]["offset"]
        dst_off = self.slots[target_id]["offset"]
        self.data[dst_off : dst_off + self.PS4_SLOT_SIZE] = self.data[src_off : src_off + self.PS4_SLOT_SIZE]
        if new_name:
            self.update_character_stats(target_id, {"name": new_name})
        self._scan_slots()
        return True

    def save(self, output_path=None):
        target = output_path or self.file_path
        self.backup()
        with open(target, 'wb') as f:
            f.write(self.data)
        print(f"File saved to: {target}")
