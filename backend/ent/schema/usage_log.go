package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UsageLog preserves the generic usage projection table used by admin/user
// dashboards and subscription accounting. SocialOps writes canonical execution
// details to social_task_logs and may project successful executions here.
type UsageLog struct {
	ent.Schema
}

func (UsageLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "usage_logs"},
	}
}

func (UsageLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("api_key_id").
			Optional().
			Nillable().
			Comment("Historical api_keys foreign key; SocialOps social task rows leave this empty"),
		field.Int64("account_id").
			Optional().
			Nillable().
			Comment("Historical account identifier column retained for table shape compatibility"),
		field.String("request_id").
			MaxLen(64).
			Optional().
			Nillable(),
		field.String("model").
			MaxLen(100).
			Default("social-action").
			Comment("Historical model column; SocialOps stores a generic social-action marker"),
		field.Int64("group_id").
			Optional().
			Nillable(),
		field.Int64("subscription_id").
			Optional().
			Nillable(),
		field.Int("input_tokens").
			Default(0).
			Comment("Historical AI token column; SocialOps social task rows keep this at zero"),
		field.Int("output_tokens").
			Default(0).
			Comment("Historical AI token column; SocialOps social task rows keep this at zero"),
		field.Int("cache_creation_tokens").
			Default(0).
			Comment("Historical AI token column; SocialOps social task rows keep this at zero"),
		field.Int("cache_read_tokens").
			Default(0).
			Comment("Historical AI token column; SocialOps social task rows keep this at zero"),
		field.Int("cache_creation_5m_tokens").
			Default(0).
			Comment("Historical AI token column; SocialOps social task rows keep this at zero"),
		field.Int("cache_creation_1h_tokens").
			Default(0).
			Comment("Historical AI token column; SocialOps social task rows keep this at zero"),
		field.Float("input_cost").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("output_cost").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("cache_creation_cost").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("cache_read_cost").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("total_cost").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("actual_cost").
			Default(0).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)"}),
		field.Float("rate_multiplier").
			Default(1).
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}),
		field.Int8("billing_type").
			Default(0),
		field.Bool("stream").
			Default(false).
			Comment("Historical AI request column; SocialOps social task rows keep this false"),
		field.Int("duration_ms").
			Optional().
			Nillable().
			Comment("Historical request timing column retained for table shape compatibility"),
		field.Int("first_token_ms").
			Optional().
			Nillable().
			Comment("Historical AI first-token timing column retained for table shape compatibility"),
		field.String("user_agent").
			MaxLen(512).
			Optional().
			Nillable(),
		field.String("ip_address").
			MaxLen(45).
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (UsageLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("usage_logs").
			Field("user_id").
			Required().
			Unique(),
		edge.From("api_key", APIKey.Type).
			Ref("usage_logs").
			Field("api_key_id").
			Unique(),
		edge.From("group", Group.Type).
			Ref("usage_logs").
			Field("group_id").
			Unique(),
		edge.From("subscription", UserSubscription.Type).
			Ref("usage_logs").
			Field("subscription_id").
			Unique(),
	}
}

func (UsageLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("api_key_id"),
		index.Fields("account_id"),
		index.Fields("group_id"),
		index.Fields("subscription_id"),
		index.Fields("created_at"),
		index.Fields("model"),
		index.Fields("request_id"),
		index.Fields("user_id", "created_at"),
		index.Fields("api_key_id", "created_at"),
		index.Fields("group_id", "created_at"),
	}
}
