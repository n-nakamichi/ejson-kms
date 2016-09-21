package crypto

import (
	"testing"

	crypto_mock "github.com/adrienkohlbecker/ejson-kms/crypto/mock"
	kms_mock "github.com/adrienkohlbecker/ejson-kms/kms/mock"
	"github.com/stretchr/testify/assert"
)

var testContext = map[string]*string{"ABC": nil}

const testKeyID = "my-key-id"

const (
	testKeyPlaintext  = "-abcdefabcdefabcdefabcdefabcdef-"
	testKeyCiphertext = "ciphertextblob"
	testConstantNonce = "abcdefabcdefabcdefabcdef"
	testPlaintext     = "abcdef"
	testCiphertext    = "EJK1];Y2lwaGVydGV4dGJsb2I=;YWJjZGVmYWJjZGVmYWJjZGVmYWJjZGVmlPmP6IWfK7WJMuXVi8aQ7TZu8vCkVA=="
)

func TestEncrypt(t *testing.T) {

	t.Run("working", func(t *testing.T) {

		client := kms_mock.New(t, testKeyID, testContext, testKeyCiphertext, testKeyPlaintext)

		crypto_mock.WithConstRandReader(testConstantNonce, func() {

			cipher := NewCipher(client, testKeyID, testContext)
			encoded, err := cipher.Encrypt(testPlaintext)
			assert.NoError(t, err)
			assert.Equal(t, encoded, testCiphertext)

		})

	})

	t.Run("with aws error", func(t *testing.T) {

		client := kms_mock.NewWithError("testing errors")

		cipher := NewCipher(client, testKeyID, testContext)
		_, err := cipher.Encrypt(testPlaintext)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Unable to generate data key")

	})

	t.Run("with encrypt error", func(t *testing.T) {

		client := kms_mock.New(t, testKeyID, testContext, testKeyCiphertext, testKeyPlaintext)

		crypto_mock.WithErrorRandReader("testing error", func() {

			cipher := NewCipher(client, testKeyID, testContext)
			_, err := cipher.Encrypt(testPlaintext)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Unable to generate nonce")

		})

	})
}

func TestDecrypt(t *testing.T) {

	t.Run("working", func(t *testing.T) {

		client := kms_mock.New(t, testKeyID, testContext, testKeyCiphertext, testKeyPlaintext)

		cipher := NewCipher(client, testKeyID, testContext)
		plaintext, err := cipher.Decrypt(testCiphertext)
		if assert.NoError(t, err) {
			assert.Equal(t, plaintext, testPlaintext)
		}

	})

	t.Run("with decode error", func(t *testing.T) {

		client := &kms_mock.Client{}
		cipher := NewCipher(client, testKeyID, testContext)
		_, err := cipher.Decrypt("abc")
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Invalid format for encoded string")
		}

	})

	t.Run("with aws error", func(t *testing.T) {

		client := kms_mock.NewWithError("testing errors")

		cipher := NewCipher(client, testKeyID, testContext)
		_, err := cipher.Decrypt(testCiphertext)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Unable to decrypt key ciphertext")
		}

	})

	t.Run("with decrypt error", func(t *testing.T) {

		client := kms_mock.New(t, testKeyID, testContext, testKeyCiphertext, "notlongenough")

		cipher := NewCipher(client, testKeyID, testContext)
		_, err := cipher.Decrypt(testCiphertext)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Expected key size of 32, got 13")
		}

	})

}
