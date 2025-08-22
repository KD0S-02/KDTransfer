package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

func GenerateID() string {
	letters := []rune("abcdefghijlmnopqrstuvwxyz1234567890")

	randID := make([]rune, 8)

	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)

	for i := 0; i < 8; i++ {
		randID[i] = letters[random.Intn(len(letters))]
	}

	return string(randID)
}

func GenerateRandSalt() string {
	timestamp := time.Now().UTC().String()
	code := GenerateID()
	return code + timestamp
}

func GenerateKey(passphrase string,
	saltData string) (key []byte, err error) {
	saltArr := sha256.Sum256([]byte(saltData))
	salt := saltArr[:]

	key, err = pbkdf2.Key(
		sha256.New,
		passphrase,
		salt,
		100000,
		32,
	)

	if err != nil {
		return nil, fmt.Errorf("error while keygen: %s",
			err)
	}

	return key, nil
}

func generateAEAD(key []byte) (aead cipher.AEAD, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err = cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return aead, nil
}

func EncrpyptData(plaintext []byte, key []byte) (
	data []byte, err error) {
	aead, err := generateAEAD(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	return append(nonce, ciphertext...), nil
}

func DecryptData(ciphertextWithNonce []byte, key []byte) (
	plaintext []byte, err error) {
	aead, err := generateAEAD(key)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()
	if len(ciphertextWithNonce) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertextWithNonce[:nonceSize],
		ciphertextWithNonce[nonceSize:]

	return aead.Open(nil, nonce, ciphertext, nil)
}
