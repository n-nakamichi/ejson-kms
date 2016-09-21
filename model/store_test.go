package model

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	crypto_mock "github.com/adrienkohlbecker/ejson-kms/crypto/mock"
	kms_mock "github.com/adrienkohlbecker/ejson-kms/kms/mock"
	"github.com/stretchr/testify/assert"
)

var testContext = map[string]*string{"ABC": nil}

const (
	testKeyID         = "my-key-id"
	testKeyPlaintext  = "-abcdefabcdefabcdefabcdefabcdef-"
	testKeyCiphertext = "ciphertextblob"
	testConstantNonce = "abcdefabcdefabcdefabcdef"
	testPlaintext     = "abcdef"
	testCiphertext    = "EJK1];Y2lwaGVydGV4dGJsb2I=;YWJjZGVmYWJjZGVmYWJjZGVmYWJjZGVmlPmP6IWfK7WJMuXVi8aQ7TZu8vCkVA=="
	testName          = "my_cred"
	testDescription   = "Some description."
	testPlaintext2    = "ghijklm"
	testName2         = "my_other_cred"
	testDescription2  = "Some other description."
)

func TestNewStore(t *testing.T) {

	store := NewStore(testKeyID, testContext)
	assert.Equal(t, testKeyID, store.KMSKeyID)
	assert.Equal(t, 1, store.Version)
	assert.Equal(t, testContext, store.EncryptionContext)
	assert.Equal(t, []*Credential{}, store.Credentials)

}

func TestLoad(t *testing.T) {

	t.Run("valid", func(t *testing.T) {

		j, err := Load("./testdata/empty.json")
		assert.NoError(t, err)
		assert.NotNil(t, j)

		assert.NotEmpty(t, j.KMSKeyID)
		assert.NotEmpty(t, j.EncryptionContext)
		assert.Equal(t, *j.EncryptionContext["KEY"], "VALUE")
		if assert.Equal(t, len(j.Credentials), 1) {

			cred := j.Credentials[0]
			assert.Equal(t, cred.Name, "test_cred")
			assert.Equal(t, cred.Description, "Some Description")
			assert.Equal(t, time.Date(2016, 9, 19, 14, 21, 22, 0, time.UTC), cred.AddedAt)
			assert.Nil(t, cred.RotatedAt)
			assert.NotEmpty(t, cred.Ciphertext)

		}

	})

	t.Run("invalid json", func(t *testing.T) {

		_, err := Load("./testdata/invalid.json")
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Unable to decode Store")
		}

	})

	t.Run("no file", func(t *testing.T) {

		_, err := Load("does-not-exist")
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Unable to read file")
		}

	})

}

func TestContains(t *testing.T) {

	j := &Store{Credentials: []*Credential{
		&Credential{Name: "test_cred"},
	}}

	assert.True(t, j.Contains("test_cred"))
	assert.False(t, j.Contains("other"))

}

func TestSave(t *testing.T) {

	t.Run("valid", func(t *testing.T) {

		j, err := Load("./testdata/empty.json")
		assert.NoError(t, err)
		assert.NotNil(t, j)

		tmpfile, goErr := ioutil.TempFile(os.TempDir(), "read-from-file")
		assert.NoError(t, goErr)
		goErr = tmpfile.Close()
		assert.NoError(t, goErr)

		err = j.Save(tmpfile.Name())
		assert.NoError(t, err)

		goErr = os.Remove(tmpfile.Name())
		assert.NoError(t, goErr)

	})

	t.Run("write error", func(t *testing.T) {

		dir, goErr := ioutil.TempDir(os.TempDir(), "read-from-file")
		assert.NoError(t, goErr)

		j := &Store{}
		err := j.Save(dir)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Unable to write file")
		}

		goErr = os.Remove(dir)
		assert.NoError(t, goErr)

	})

}

func TestAdd(t *testing.T) {

	t.Run("working", func(t *testing.T) {

		client := kms_mock.GenerateDataKey(t, testKeyID, testContext, testKeyCiphertext, testKeyPlaintext)
		store := NewStore(testKeyID, testContext)

		crypto_mock.WithConstRandReader(testConstantNonce, func() {
			err := store.Add(client, testPlaintext, testName, testDescription)
			assert.NoError(t, err)
		})

		if assert.Len(t, store.Credentials, 1) {
			cred := store.Credentials[0]
			assert.Equal(t, cred.Name, testName)
			assert.Equal(t, cred.Description, testDescription)
			assert.Equal(t, cred.Ciphertext, testCiphertext)
			assert.WithinDuration(t, time.Now(), cred.AddedAt, 2*time.Second)
			assert.Nil(t, cred.RotatedAt)
		}

	})

	t.Run("fails", func(t *testing.T) {

		client := kms_mock.GenerateDataKey(t, testKeyID, testContext, testKeyCiphertext, testKeyPlaintext)
		store := NewStore(testKeyID, testContext)

		crypto_mock.WithErrorRandReader("testing errors", func() {
			err := store.Add(client, testPlaintext, testName, testDescription)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), "Unable to encrypt credential")
			}
		})

	})

}

func TestExportPlaintext(t *testing.T) {

	store := NewStore(testKeyID, testContext)

	t.Run("empty", func(t *testing.T) {

		client := kms_mock.Decrypt(t, testKeyID, testContext, testKeyCiphertext, testKeyPlaintext)
		items, err := store.ExportPlaintext(client)
		assert.NoError(t, err)

		_, open := <-items
		assert.False(t, open)

	})

	t.Run("working", func(t *testing.T) {

		client := kms_mock.GenerateDataKey(t, testKeyID, testContext, testKeyCiphertext, testKeyPlaintext)

		err := store.Add(client, testPlaintext, testName, testDescription)
		assert.NoError(t, err)

		err = store.Add(client, testPlaintext2, testName2, testDescription2)
		assert.NoError(t, err)

		client = kms_mock.Decrypt(t, testKeyID, testContext, testKeyCiphertext, testKeyPlaintext)
		items, err := store.ExportPlaintext(client)
		assert.NoError(t, err)

		item, open := <-items
		assert.True(t, open)
		assert.Equal(t, item.Name, testName)
		assert.Equal(t, item.Plaintext, testPlaintext)

		item, open = <-items
		assert.True(t, open)
		assert.Equal(t, item.Name, testName2)
		assert.Equal(t, item.Plaintext, testPlaintext2)

		_, open = <-items
		assert.False(t, open)

	})

	t.Run("fails", func(t *testing.T) {

		client := kms_mock.GenerateDataKey(t, testKeyID, testContext, testKeyCiphertext, testKeyPlaintext)

		err := store.Add(client, testPlaintext, testName, testDescription)
		assert.NoError(t, err)

		client = kms_mock.DecryptWithError("testing error")
		items, err := store.ExportPlaintext(client)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Unable to decrypt credential")
		}

		_, open := <-items
		assert.False(t, open)

	})

}
