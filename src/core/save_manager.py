import shutil
from datetime import datetime
from .structures import PLAYER_GAME_DATA, GA_ITEM
from .crypto import decrypt_pc_save, encrypt_pc_save, calculate_sha256

class SaveManager:
    # PS4 Constants
    PS4_SLOT_SIZE = 0x280000
    PS4_FIRST_SLOT_OFFSET = 0x310
    
    # PC Constants
    PC_SLOT_SIZE = 0x280000
    PC_STEAM_ID_OFFSET = 0x19003B4 # Offset in decrypted payload
    
    # Common Offsets (relative to slot start)
    NAME_OFFSET_IN_SLOT = 0xe2b5
    STATS_OFFSET_IN_SLOT = NAME_OFFSET_IN_SLOT - 88
    EVENT_FLAGS_OFFSET_IN_SLOT = 0x1bfaf0 
    GA_ITEMS_OFFSET_IN_SLOT = 0x20

    def __init__(self, file_path):
        self.file_path = file_path
        self.raw_data = None
        self.decrypted_data = None
        self.is_pc = False
        self.slots = []
        self.steam_id = None
        
    def load(self):
        with open(self.file_path, 'rb') as f:
            self.raw_data = bytearray(f.read())
        
        if self.raw_data.startswith(b"BND4"):
            self.is_pc = True
            self.decrypted_data = bytearray(decrypt_pc_save(self.raw_data[0x30:]))
            self._detect_steam_id()
        else:
            self.is_pc = False
            self.decrypted_data = self.raw_data
            
        self._scan_slots()

    def _detect_steam_id(self):
        if self.is_pc and len(self.decrypted_data) > self.PC_STEAM_ID_OFFSET + 8:
            steam_id_bytes = self.decrypted_data[self.PC_STEAM_ID_OFFSET : self.PC_STEAM_ID_OFFSET + 8]
            self.steam_id = int.from_bytes(steam_id_bytes, 'little')

    def set_steam_id(self, new_steam_id_int):
        """Updates SteamID in the decrypted payload (PC only)."""
        if not self.is_pc:
            return False
        
        self.steam_id = new_steam_id_int
        steam_id_bytes = new_steam_id_int.to_bytes(8, 'little')
        self.decrypted_data[self.PC_STEAM_ID_OFFSET : self.PC_STEAM_ID_OFFSET + 8] = steam_id_bytes
        
        # After changing SteamID, we MUST recalculate SHA256 for ALL slots
        # because the slot checksum includes the SteamID in some versions of BND4
        # or at least it's good practice to refresh all checksums.
        for slot in self.slots:
            if slot['active']:
                slot_offset = slot['offset']
                new_checksum = calculate_sha256(self.decrypted_data[slot_offset : slot_offset + self.PC_SLOT_SIZE])
                self.decrypted_data[slot_offset - 32 : slot_offset] = new_checksum
                
        return True

    def _scan_slots(self):
        self.slots = []
        first_slot_off = 0x310
        
        for i in range(10):
            offset = first_slot_off + (i * (self.PS4_SLOT_SIZE + (32 if self.is_pc else 0)))
            slot_data_offset = offset + (32 if self.is_pc else 0)

            if slot_data_offset + self.PS4_SLOT_SIZE > len(self.decrypted_data):
                break
            
            name_pos = slot_data_offset + self.NAME_OFFSET_IN_SLOT
            name_bytes = self.decrypted_data[name_pos : name_pos + 32]
            name = name_bytes.decode('utf-16le').strip('\x00')
            
            self.slots.append({
                "id": i,
                "offset": slot_data_offset,
                "name": name if name else "Empty Slot",
                "active": bool(name)
            })

    def get_character_stats(self, slot_id):
        if slot_id >= len(self.slots):
            return None
        slot_offset = self.slots[slot_id]["offset"]
        stats_pos = slot_offset + self.STATS_OFFSET_IN_SLOT
        stats_data = self.decrypted_data[stats_pos : stats_pos + 0x200]
        return PLAYER_GAME_DATA.parse(stats_data)

    def update_character_stats(self, slot_id, new_stats_dict):
        if slot_id >= len(self.slots):
            return False
        slot_offset = self.slots[slot_id]["offset"]
        stats_pos = slot_offset + self.STATS_OFFSET_IN_SLOT
        current_stats = self.get_character_stats(slot_id)
        for key, value in new_stats_dict.items():
            if hasattr(current_stats, key):
                setattr(current_stats, key, value)
        updated_bytes = PLAYER_GAME_DATA.build(current_stats)
        self.decrypted_data[stats_pos : stats_pos + len(updated_bytes)] = updated_bytes
        
        if self.is_pc:
            checksum_pos = slot_offset - 32
            new_checksum = calculate_sha256(self.decrypted_data[slot_offset : slot_offset + self.PC_SLOT_SIZE])
            self.decrypted_data[checksum_pos : checksum_pos + 32] = new_checksum
            
        self._scan_slots()
        return True

    def import_character(self, source_manager, source_slot_id, target_slot_id):
        if source_slot_id >= len(source_manager.slots) or target_slot_id >= len(self.slots):
            return False
        src_off = source_manager.slots[source_slot_id]["offset"]
        dst_off = self.slots[target_slot_id]["offset"]
        self.decrypted_data[dst_off : dst_off + self.PS4_SLOT_SIZE] = source_manager.decrypted_data[src_off : src_off + self.PS4_SLOT_SIZE]
        
        if self.is_pc:
            new_checksum = calculate_sha256(self.decrypted_data[dst_off : dst_off + self.PC_SLOT_SIZE])
            self.decrypted_data[dst_off - 32 : dst_off] = new_checksum
            
        self._scan_slots()
        return True

    def delete_character(self, slot_id):
        if slot_id >= len(self.slots):
            return False
        offset = self.slots[slot_id]["offset"]
        self.decrypted_data[offset : offset + self.PS4_SLOT_SIZE] = b'\x00' * self.PS4_SLOT_SIZE
        
        if self.is_pc:
            new_checksum = calculate_sha256(self.decrypted_data[offset : offset + self.PC_SLOT_SIZE])
            self.decrypted_data[offset - 32 : offset] = new_checksum
            
        self._scan_slots()
        return True

    def add_item(self, slot_id, item_id, quantity=1):
        if slot_id >= len(self.slots):
            return False
        slot_offset = self.slots[slot_id]["offset"]
        ga_items_pos = slot_offset + self.GA_ITEMS_OFFSET_IN_SLOT
        found_idx = -1
        max_handle = 0x80000000
        for i in range(0x1400):
            item_pos = ga_items_pos + (i * 17)
            current_id = int.from_bytes(self.decrypted_data[item_pos+4 : item_pos+8], 'little')
            current_handle = int.from_bytes(self.decrypted_data[item_pos : item_pos+4], 'little')
            if current_id == 0 and found_idx == -1:
                found_idx = i
            if current_handle > max_handle and current_handle < 0xFFFFFFFF:
                max_handle = current_handle
        if found_idx == -1:
            return False
        new_handle = max_handle + 1
        new_item = {"handle": new_handle, "id": item_id, "data": {"unk2": -1, "unk3": -1, "aow_handle": 0xFFFFFFFF, "unk5": 0}}
        self.decrypted_data[ga_items_pos + (found_idx * 17) : ga_items_pos + (found_idx * 17) + 17] = GA_ITEM.build(new_item)
        
        if self.is_pc:
            new_checksum = calculate_sha256(self.decrypted_data[slot_offset : slot_offset + self.PC_SLOT_SIZE])
            self.decrypted_data[slot_offset - 32 : slot_offset] = new_checksum
            
        return True

    def get_event_flag(self, slot_id, flag_id):
        if slot_id >= len(self.slots):
            return False
        byte_offset = flag_id // 8
        bit_mask = 1 << (flag_id % 8)
        slot_offset = self.slots[slot_id]["offset"]
        flag_pos = slot_offset + self.EVENT_FLAGS_OFFSET_IN_SLOT + byte_offset
        if flag_pos >= len(self.decrypted_data):
            return False
        return bool(self.decrypted_data[flag_pos] & bit_mask)

    def set_event_flag(self, slot_id, flag_id, active):
        if slot_id >= len(self.slots):
            return False
        byte_offset = flag_id // 8
        bit_mask = 1 << (flag_id % 8)
        slot_offset = self.slots[slot_id]["offset"]
        flag_pos = slot_offset + self.EVENT_FLAGS_OFFSET_IN_SLOT + byte_offset
        if flag_pos >= len(self.decrypted_data):
            return False
        if active:
            self.decrypted_data[flag_pos] |= bit_mask
        else:
            self.decrypted_data[flag_pos] &= ~bit_mask
            
        if self.is_pc:
            new_checksum = calculate_sha256(self.decrypted_data[slot_offset : slot_offset + self.PC_SLOT_SIZE])
            self.decrypted_data[slot_offset - 32 : slot_offset] = new_checksum
            
        return True

    def backup(self):
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        backup_path = f"{self.file_path}.{timestamp}.bak"
        shutil.copy2(self.file_path, backup_path)
        return backup_path

    def save(self, output_path=None):
        target = output_path or self.file_path
        self.backup()
        
        if self.is_pc:
            iv = self.raw_data[0x30:0x40]
            encrypted_payload = encrypt_pc_save(self.decrypted_data, iv)
            self.raw_data[0x30:] = encrypted_payload
            final_data = self.raw_data
        else:
            final_data = self.decrypted_data
            
        with open(target, 'wb') as f:
            f.write(final_data)
        print(f"File saved to: {target}")
