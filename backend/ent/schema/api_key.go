package schema

import (
	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
	"github.com/Wei-Shaw/socialops/internal/domain"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// APIKey maps historical api_keys rows retained for migration and accounting
// consistency. SocialOps no longer exposes user API key management.
type APIKey struct {
	ent.Schema
}

func (APIKey) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "api_keys"},
	}
}

func (APIKey) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (APIKey) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("key").
			MaxLen(128).
			NotEmpty().
			Unique(),
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.Int64("group_id").
			Optional().
			Nillable(),
		field.String("status").
			MaxLen(20).
			Default(domain.StatusActive),
		field.Time("last_used_at").
			Optional().
			Nillable().
			Comment("Historical last usage time for removed user API key records"),
		field.JSON("ip_whitelist", []string{}).
			Optional().
			Comment("Historical allowed IP/CIDR list for removed user API key records"),
		field.JSON("ip_blacklist", []string{}).
			Optional().
			Comment("Historical blocked IP/CIDR list for removed user API key records"),

		// Historical quota fields retained only so existing api_keys rows can be read.
		field.Float("quota").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("Historical quota limit in USD for removed user API key records"),
		field.Float("quota_used").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("Historical used quota amount in USD for removed user API key records"),
		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("Historical expiration time for removed user API key records"),

		// Historical rate-limit fields retained only for existing api_keys rows.
		field.Float("rate_limit_5h").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("Historical USD rate limit per 5 hours for removed user API key records"),
		field.Float("rate_limit_1d").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("Historical USD rate limit per day for removed user API key records"),
		field.Float("rate_limit_7d").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("Historical USD rate limit per 7 days for removed user API key records"),
		field.Float("usage_5h").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("Historical used amount in USD for the 5h window"),
		field.Float("usage_1d").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("Historical used amount in USD for the 1d window"),
		field.Float("usage_7d").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("Historical used amount in USD for the 7d window"),
		field.Time("window_5h_start").
			Optional().
			Nillable().
			Comment("Historical start time of the 5h rate limit window"),
		field.Time("window_1d_start").
			Optional().
			Nillable().
			Comment("Historical start time of the 1d rate limit window"),
		field.Time("window_7d_start").
			Optional().
			Nillable().
			Comment("Historical start time of the 7d rate limit window"),
	}
}

func (APIKey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("api_keys").
			Field("user_id").
			Unique().
			Required(),
		edge.From("group", Group.Type).
			Ref("api_keys").
			Field("group_id").
			Unique(),
		edge.To("usage_logs", UsageLog.Type),
	}
}

func (APIKey) Indexes() []ent.Index {
	return []ent.Index{
		// key 字段已在 Fields() 中声明 Unique()，无需重复索引
		index.Fields("user_id"),
		index.Fields("group_id"),
		index.Fields("status"),
		index.Fields("deleted_at"),
		index.Fields("last_used_at"),
		// Historical quota lookup index retained for existing rows.
		index.Fields("quota", "quota_used"),
		index.Fields("expires_at"),
	}
}
