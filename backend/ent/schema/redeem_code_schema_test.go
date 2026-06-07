package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedeemCodeCodeFieldMatchesGeneratedAndFixedCodeContracts(t *testing.T) {
	descriptor := requireEntFieldDescriptor(t, RedeemCode{}.Fields(), "code")
	generatedCodeLength := len("12345678-12345678-12345678-12345678")

	require.GreaterOrEqual(t, descriptor.Size, generatedCodeLength, "generated redeem codes must fit the schema")
	require.Equal(t, 128, descriptor.Size, "admin fixed redeem-code endpoint accepts codes up to 128 characters")
}
