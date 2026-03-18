from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend
import hashlib

# Elden Ring PC Save Key (AES-128-CBC)
# Note: In ER, the IV is the first 16 bytes of the encrypted block.
PC_SAVE_KEY = bytes(
    [
        0x42,
        0x03,
        0xB2,
        0xEF,
        0xCC,
        0x74,
        0x61,
        0x37,
        0x45,
        0x63,
        0x51,
        0x72,
        0x67,
        0xAD,
        0x27,
        0x33,
    ]
)


def decrypt_pc_save(data: bytes) -> bytes:
    """Decrypts an Elden Ring PC save (.sl2)."""
    # The first 16 bytes are the IV
    iv = data[:16]
    encrypted_payload = data[16:]

    cipher = Cipher(
        algorithms.AES(PC_SAVE_KEY), modes.CBC(iv), backend=default_backend()
    )
    decryptor = cipher.decryptor()
    return decryptor.update(encrypted_payload) + decryptor.finalize()


def encrypt_pc_save(data: bytes, iv: bytes) -> bytes:
    """Encrypts an Elden Ring PC save (.sl2)."""
    cipher = Cipher(
        algorithms.AES(PC_SAVE_KEY), modes.CBC(iv), backend=default_backend()
    )
    encryptor = cipher.encryptor()
    return iv + encryptor.update(data) + encryptor.finalize()


def calculate_sha256(data: bytes) -> bytes:
    """Calculates SHA256 checksum for PC save integrity."""
    return hashlib.sha256(data).digest()


def calculate_md5(data: bytes) -> bytes:
    """Calculates MD5 checksum for PlayStation save integrity."""
    return hashlib.md5(data).digest()
