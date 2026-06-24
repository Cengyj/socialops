package schema

import (
	"github.com/Wei-Shaw/socialops/internal/domain"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
)

// SocialTaskLog holds the schema definition for social account task execution logs.
type SocialTaskLog struct {
	ent.Schema
}

func (SocialTaskLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "social_task_logs"},
	}
}

func (SocialTaskLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (SocialTaskLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("social_account_id").
			Comment("关联的社交账号 ID"),
		field.Int64("user_id").
			Comment("执行任务的用户 ID"),

		// 任务信息
		field.String("action").
			MaxLen(50).
			Comment("Social task action: login / login_check / follow / post / like / retweet / update_profile / update_avatar / update_banner"),
		field.Text("target").
			Optional().
			Nillable().
			Comment("操作目标（如目标用户名、URL 等）"),
		field.Text("content").
			Optional().
			Nillable().
			Comment("操作内容（如发帖内容等）"),
		field.JSON("payload", domain.SocialTaskPayload{}).
			Default(func() domain.SocialTaskPayload { return domain.SocialTaskPayload{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Comment("Structured execution payload snapshot"),
		field.JSON("template_snapshot", domain.SocialTaskTemplateSnapshot{}).
			Default(func() domain.SocialTaskTemplateSnapshot { return domain.SocialTaskTemplateSnapshot{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Comment("Saved task template snapshot resolved at submission time"),

		// 执行结果
		field.String("status").
			MaxLen(20).
			Default("pending").
			Comment("执行状态：pending / running / success / failed"),
		field.Text("result_message").
			Optional().
			Nillable().
			Comment("执行结果描述"),
		field.Float("price").
			Default(0.1).
			Comment("本次真实社交动作单价"),
		field.Float("charged_amount").
			Default(0).
			Comment("最终扣费金额；失败或未执行为 0"),
		field.String("charge_status").
			MaxLen(30).
			Default("not_charged").
			Comment("扣费状态：not_charged / charged / refunded"),
		field.String("charge_source").
			MaxLen(30).
			Optional().
			Nillable().
			Comment("扣费来源：subscription / wallet / mixed"),
		field.Int64("proxy_id").
			Optional().
			Nillable().
			Comment("本次执行使用的用户代理 ID"),
		field.Text("proxy_snapshot").
			Optional().
			Nillable().
			Comment("执行提交时的代理快照"),
		field.String("billing_request_id").
			MaxLen(255).
			Optional().
			Nillable().
			Comment("扣费/预扣请求 ID"),
		field.String("idempotency_key").
			MaxLen(255).
			Optional().
			Nillable().
			Comment("用户侧幂等键"),

		// 时间
		field.Time("executed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("实际执行时间"),
	}
}

func (SocialTaskLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("social_account", SocialAccount.Type).
			Ref("task_logs").
			Field("social_account_id").
			Unique().
			Required(),
		edge.From("user", User.Type).
			Ref("social_task_logs").
			Field("user_id").
			Unique().
			Required(),
	}
}

func (SocialTaskLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("social_account_id"),
		index.Fields("user_id"),
		index.Fields("action"),
		index.Fields("status"),
		index.Fields("user_id", "action"),
		index.Fields("social_account_id", "status"),
		index.Fields("social_account_id").
			StorageKey("idx_social_task_logs_one_active_per_account").
			Unique().
			Annotations(entsql.IndexWhere("status IN ('pending', 'running')")),
		index.Fields("user_id", "social_account_id", "action", "idempotency_key").
			StorageKey("idx_social_task_logs_user_account_action_idem_unique").
			Unique().
			Annotations(entsql.IndexWhere("idempotency_key IS NOT NULL")),
		index.Fields("charge_status"),
		index.Fields("proxy_id"),
	}
}
