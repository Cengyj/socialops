//go:build unit

package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTwitterExecutionAuthEncryptorUsesLegacyDeterministicFormat(t *testing.T) {
	encryptor := NewTwitterExecutionAuthEncryptor()
	payload := `{"access_token":"access","token_secret":"secret","screen_name":"northwind_ops"}`
	flyingBirdCiphertext := "pmHDN+xTzhmYcYhKpf5x37xU5+4W0LDwM3yC6YmPZVBIawhzfTR60dA6cB7Ts7HwCfJK0+6x9ebOOuP9tSgDy/TSSMuF4FvcngUX2Jrozn5ot2FvtqrSMuU2MvaGVj87sxXVfso1JOsWjkPx5WoU2A=="

	stored, err := encryptor.Encrypt(payload)
	require.NoError(t, err)
	require.NotEmpty(t, stored)
	require.Equal(t, flyingBirdCiphertext, stored)
	require.NotEqual(t, payload, stored)
	require.NotContains(t, stored, "access")
	require.NotContains(t, stored, "token_secret")

	storedAgain, err := encryptor.Encrypt(payload)
	require.NoError(t, err)
	require.Equal(t, stored, storedAgain)

	decrypted, err := encryptor.Decrypt(stored)
	require.NoError(t, err)
	require.Equal(t, payload, decrypted)
}

func TestTwitterExecutionAuthEncryptorRejectsAESGCMCiphertext(t *testing.T) {
	aesEncryptor, err := NewAESEncryptor(aesTestCfg(aesHexKey(32, 0x42)))
	require.NoError(t, err)
	encryptor := NewTwitterExecutionAuthEncryptor()
	payload := `{"access_token":"legacy-access","token_secret":"legacy-secret","screen_name":"northwind_ops"}`

	existingAESGCM, err := aesEncryptor.Encrypt(payload)
	require.NoError(t, err)
	require.NotEqual(t, payload, existingAESGCM)

	_, err = encryptor.Decrypt(existingAESGCM)
	require.Error(t, err)

	newLegacyCiphertext, err := encryptor.Encrypt(payload)
	require.NoError(t, err)
	_, err = aesEncryptor.Decrypt(newLegacyCiphertext)
	require.Error(t, err)
}
