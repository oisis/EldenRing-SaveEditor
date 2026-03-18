import os
import shutil
from datetime import datetime
from .structures import SAVE_SLOT, PLAYER_GAME_DATA
from .crypto import decrypt_pc_save, encrypt_pc_save, calculate_sha256

class SaveManager:
    # PS4 Decrypted Save Wizard offsets
    # These are heuristics based on the provided save file
    PS4_SLOT_SIZE = 0x280000
    PS4_FIRST_SLOT_OFFSET = 0x310 # Common offset for PS4 decrypted saves
    
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
        # For PS4, we scan 10 potential slots
        for i in range(10):
            offset = self.PS4_FIRST_SLOT_OFFSET + (i * self.PS4_SLOT_SIZE)
            if offset + self.PS4_SLOT_SIZE > len(self.data):
                break
            
            # Check if slot is active by looking for a non-empty name
            # Name is at offset + 0x280000 - some_offset... 
            # Actually, let's use the offset we found for 'Nowy' to calibrate
            # Nowy was at 0xe5c5 (name). 
            # If Slot 0 starts at 0x310, then name is at 0xe5c5 - 0x310 = 0xe2b5 from slot start.
            name_offset_in_slot = 0xe2b5
            name_bytes = self.data[offset + name_offset_in_slot : offset + name_offset_in_slot + 32]
            name = name_bytes.decode('utf-16le').strip('\x00')
            
            if name:
                self.slots.append({
                    "id": i,
                    "offset": offset,
                    "name": name,
                    "active": True
                })
            else:
                self.slots.append({
                    "id": i,
                    "offset": offset,
                    "name": "Empty Slot",
                    "active": False
                })

    def backup(self):
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        backup_path = f"{self.file_path}.{timestamp}.bak"
        shutil.copy2(self.file_path, backup_path)
        return backup_path

    def clone_character(self, source_id, target_id, new_name=None):
        """Clones character from source_id slot to target_id slot."""
        if source_id >= len(self.slots) or target_id >= len(self.slots):
            return False
        
        src_off = self.slots[source_id]["offset"]
        dst_off = self.slots[target_id]["offset"]
        
        # Copy entire slot data
        self.data[dst_off : dst_off + self.PS4_SLOT_SIZE] = self.data[src_off : src_off + self.PS4_SLOT_SIZE]
        
        # Update name if provided
        if new_name:
            name_offset = dst_off + 0xe2b5
            name_bytes = new_name.encode('utf-16le').ljust(32, b'\x00')
            self.data[name_offset : name_offset + 32] = name_bytes
            
        print(f"Cloned {self.slots[source_id]['name']} to Slot {target_id}")
        self._scan_slots() # Refresh slot info
        return True

    def save(self, output_path=None):
        target = output_path or self.file_path
        self.backup()
        with open(target, 'wb') as f:
            f.write(self.data)
        print(f"File saved to: {target}")
