package schema

import (
	"testing"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"github.com/stretchr/testify/require"
)

func TestPaymentOrderMonetaryFieldsPreserveSupportedCurrencyPrecision(t *testing.T) {
	fields := PaymentOrder{}.Fields()

	for _, name := range []string{"amount", "pay_amount", "refund_amount"} {
		t.Run(name, func(t *testing.T) {
			descriptor := requireEntFieldDescriptor(t, fields, name)
			require.Equal(t, "decimal(20,8)", descriptor.SchemaType[dialect.Postgres])
		})
	}
}

func requireEntFieldDescriptor(t *testing.T, fields []ent.Field, name string) *field.Descriptor {
	t.Helper()

	for _, entField := range fields {
		descriptor := entField.Descriptor()
		if descriptor.Name == name {
			return descriptor
		}
	}

	require.Failf(t, "missing field descriptor", "schema should include field %s", name)
	return nil
}
