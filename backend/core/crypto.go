package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
)

var (
	// SaveKey is the standard Elden Ring PC Save AES-128-CBC key.
	SaveKey = []byte{0x99, 0xad, 0x2d, 0x50, 0xed, 0xf2, 0xfb, 0x01, 0xc5, 0xf3, 0xec, 0x3a, 0x2b, 0xca, 0xb6, 0x9d}
)

// DecryptSave decrypts the AES-128-CBC encrypted payload from a PC save.
// The first 16 bytes of the encrypted data are used as the IV.
func DecryptSave(data []byte) ([]byte, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("data too short for decryption")
	}

	block, err := aes.NewCipher(SaveKey)
	if err != nil {
		return nil, err
	}

	iv := data[:16]
	encrypted := data[16:]
	
	// CBC mode works on blocks of 16 bytes
	if len(encrypted)%16 != 0 {
		return nil, fmt.Errorf("encrypted data is not a multiple of block size")
	}

	decrypted := make([]byte, len(encrypted))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, encrypted)

	return decrypted, nil
}

// EncryptSave encrypts the payload using AES-128-CBC.
// It returns the IV prepended to the encrypted data.
func EncryptSave(data []byte, iv []byte) ([]byte, error) {
	if len(iv) != 16 {
		return nil, fmt.Errorf("invalid IV length")
	}

	block, err := aes.NewCipher(SaveKey)
	if err != nil {
		return nil, err
	}

	if len(data)%16 != 0 {
		return nil, fmt.Errorf("data length is not a multiple of block size")
	}

	encrypted := make([]byte, len(data))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, data)

	result := make([]byte, 0, len(iv)+len(encrypted))
	result = append(result, iv...)
	result = append(result, encrypted...)
	
	return result, nil
}

// ComputeMD5 calculates the MD5 checksum of the given data.
func ComputeMD5(data []byte) [16]byte {
	return md5.Sum(data)
}

// ComputeSHA256 calculates the SHA256 checksum of the given data.
func ComputeSHA256(data []byte) [32]byte {
	return sha256.Sum256(data)
}
