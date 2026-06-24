package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
)

// SocialIP holds the schema definition for user IP/proxy bindings.
type SocialIP struct {
	ent.Schema
}

func (SocialIP) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "social_ips"},
	}
}

func (SocialIP) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (SocialIP) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Comment("所属用户 ID"),

		// IP/代理信息
		field.String("name").
			MaxLen(100).
			Comment("IP/代理标签名称，如 Tokyo Residential 01"),
		field.String("ip_type").
			MaxLen(30).
			Default("residential").
			Comment("类型：residential / static / mobile / datacenter / dynamic"),
		field.String("endpoint").
			MaxLen(500).
			Optional().
			Nillable().
			Comment("代理端点地址，如 socks5://user:pass@host:port"),

		// 状态
		field.String("status").
			MaxLen(20).
			Default("unknown").
			Comment("连通状态：online / offline / unknown"),
		field.Int("latency_ms").
			Optional().
			Nillable().
			Comment("最近一次检测延迟（毫秒）"),
		field.Time("last_check_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("最近一次检测时间"),

		// Compatibility-only historical field. It is not account ownership and
		// must not be used for default proxy authorization; one user-owned proxy
		// may be used by multiple accounts owned by that user.
		field.Int64("bound_social_account_id").
			Optional().
			Nillable().
			Comment("兼容字段：不要作为账号归属或默认代理授权依据"),

		// 备注
		field.Text("remark").
			Optional().
			Nillable().
			Comment("备注"),
	}
}

func (SocialIP) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("social_ips").
			Field("user_id").
			Unique().
			Required(),
	}
}

func (SocialIP) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("status"),
		index.Fields("ip_type"),
		index.Fields("deleted_at"),
		index.Fields("user_id", "status"),
	}
}
