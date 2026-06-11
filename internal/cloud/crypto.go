package cloud

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

func encryptAPIKey(keyHex, plaintext string) (string, error) {
	key, err := decodeKey(keyHex)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptAPIKey(keyHex, encoded string) (string, error) {
	key, err := decodeKey(keyHex)
	if err != nil {
		return "", err
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("密文无效")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func decodeKey(keyHex string) ([]byte, error) {
	if len(keyHex) == 64 {
		out := make([]byte, 32)
		for i := 0; i < 32; i++ {
			var b byte
			_, err := fmt.Sscanf(keyHex[i*2:i*2+2], "%02x", &b)
			if err != nil {
				return nil, fmt.Errorf("REGULUS_CLOUD_ENCRYPTION_KEY 须为 64 位十六进制")
			}
			out[i] = b
		}
		return out, nil
	}
	if len(keyHex) >= 32 {
		return []byte(keyHex[:32]), nil
	}
	return nil, fmt.Errorf("REGULUS_CLOUD_ENCRYPTION_KEY 长度不足")
}
