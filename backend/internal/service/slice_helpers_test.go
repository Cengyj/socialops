//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniqueInt64sPreserveOrder(t *testing.T) {
	require.Nil(t, UniqueInt64sPreserveOrder(nil))
	require.Equal(t, []int64{3, 1, 2}, UniqueInt64sPreserveOrder([]int64{3, 1, 3, 2, 1}))
	require.Equal(t, []int64{0, 4, -1, 2}, UniqueInt64sPreserveOrder([]int64{0, 4, -1, 4, 2}))
}
