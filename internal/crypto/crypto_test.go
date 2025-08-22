package crypto

import (
	"testing"
)

func TestCrypto(t *testing.T) {
	passphrase := "earhjgu43ut3434j3k"
	data := []string{"some test data"}

	saltData := GenerateRandSalt()

	for _, d := range data {
		key, err := GenerateKey(passphrase, saltData)
		if err != nil {
			t.Errorf("Error when generating key: %s", err.Error())
		}

		plaintext := []byte(d)
		ciphertext, err := EncrpyptData(plaintext, key)
		if err != nil {
			t.Errorf("Error during encryption: %s", err.Error())
		}

		key, err = GenerateKey(passphrase, saltData)
		if err != nil {
			t.Errorf("Error when generating key: %s", err.Error())
		}
		plaintext, err = DecryptData(ciphertext, key)
		if err != nil {
			t.Errorf("Error during decryption: %s", err.Error())
		}

		if string(plaintext) != d {
			t.Errorf("plaintext after decryption %s != data %s", plaintext, d)
		}
	}
}

func TestDifferentSaltsProduceDifferentKeys(t *testing.T) {
	passphrase := "same_passphrase"

	salt1 := GenerateRandSalt()
	salt2 := GenerateRandSalt()

	key1, err := GenerateKey(passphrase, salt1)
	if err != nil {
		t.Fatal("Failed to generate key1:", err)
	}

	key2, err := GenerateKey(passphrase, salt2)
	if err != nil {
		t.Fatal("Failed to generate key2:", err)
	}

	if string(key1) == string(key2) {
		t.Error("Different salts should produce different keys")
	}
}

func TestWrongPassphraseFailsDecryption(t *testing.T) {
	data := "secret message"
	saltData := GenerateRandSalt()

	// Encrypt with correct passphrase
	key1, _ := GenerateKey("correct_password", saltData)
	ciphertext, _ := EncrpyptData([]byte(data), key1)

	// Try to decrypt with wrong passphrase
	key2, _ := GenerateKey("wrong_password", saltData)
	_, err := DecryptData(ciphertext, key2)

	if err == nil {
		t.Error("Decryption should fail with wrong passphrase")
	}
}

func TestEmptyData(t *testing.T) {
	passphrase := "test_password"
	saltData := GenerateRandSalt()

	key, _ := GenerateKey(passphrase, saltData)

	// Test empty data
	ciphertext, err := EncrpyptData([]byte(""), key)
	if err != nil {
		t.Error("Should handle empty data:", err)
	}

	plaintext, err := DecryptData(ciphertext, key)
	if err != nil {
		t.Error("Should decrypt empty data:", err)
	}

	if len(plaintext) != 0 {
		t.Error("Decrypted empty data should be empty")
	}
}

func TestLargeData(t *testing.T) {
	passphrase := "test_password"
	saltData := GenerateRandSalt()

	// Test with 1MB of data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	key, _ := GenerateKey(passphrase, saltData)

	ciphertext, err := EncrpyptData(largeData, key)
	if err != nil {
		t.Error("Should handle large data:", err)
	}

	plaintext, err := DecryptData(ciphertext, key)
	if err != nil {
		t.Error("Should decrypt large data:", err)
	}

	if len(plaintext) != len(largeData) {
		t.Errorf("Decrypted data length mismatch: got %d, want %d",
			len(plaintext), len(largeData))
	}
}
