package core

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

var pcSaveKey = []byte{
	0x40, 0x17, 0x81, 0x30, 0xDF, 0x0A, 0x24, 0x54,
	0x72, 0x09, 0x5C, 0x71, 0x0C, 0x25, 0x4B, 0xDD,
}

func DecryptSave(data []byte) ([]byte, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("data too short")
	}

	iv := data[:16]
	encryptedData := data[16:]

	block, err := aes.NewCipher(pcSaveKey)
	if err != nil {
		return nil, err
	}

	if len(encryptedData)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	decrypted := make([]byte, len(encryptedData))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, encryptedData)

	return decrypted, nil
}

func EncryptSave(data []byte, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(pcSaveKey)
	if err != nil {
		return nil, err
	}

	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("plaintext is not a multiple of the block size")
	}

	encrypted := make([]byte, len(data))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, data)

	// Result is IV + Encrypted Data
	result := append(iv, encrypted...)
	return result, nil
}
