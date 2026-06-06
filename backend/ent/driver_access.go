package ent

import "entgo.io/ent/dialect"

// Driver exposes the underlying Ent driver for repository code that needs raw SQL.
func (c *Client) Driver() dialect.Driver {
	return c.driver
}
