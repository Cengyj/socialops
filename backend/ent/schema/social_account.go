package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
)

// SocialAccount holds the schema definition for social media accounts.
type SocialAccount struct {
	ent.Schema
}

func (SocialAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "social_accounts"},
	}
}

func (SocialAccount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (SocialAccount) Fields() []ent.Field {
	return []ent.Field{
		// 账号基本信息
		field.String("name").
			MaxLen(100).
			Comment("账号名称 / 用户名，如 @northwind_ops"),
		field.String("platform").
			MaxLen(50).
			Comment("平台：x_twitter / instagram / tiktok / facebook"),
		field.String("platform_key").
			MaxLen(50).
			Comment("Normalized platform key for total-pool uniqueness"),
		field.String("name_key").
			MaxLen(100).
			Comment("Normalized username key for total-pool uniqueness"),
		field.String("account_id").
			MaxLen(100).
			Optional().
			Nillable().
			Comment("平台账号 ID"),
		field.String("password").
			MaxLen(1024).
			Optional().
			Nillable().
			Comment("账号密码（按业务要求原样存储和返回）"),
		field.String("phone").
			MaxLen(50).
			Optional().
			Nillable().
			Comment("绑定手机号"),
		field.String("email").
			MaxLen(255).
			Optional().
			Nillable().
			Comment("绑定邮箱"),
		field.String("email_password").
			MaxLen(1024).
			Optional().
			Nillable().
			Comment("邮箱密码（按业务要求原样存储和返回）"),

		// 账号状态
		field.String("account_status").
			MaxLen(30).
			Default("pending_check").
			Comment("账号状态：pending_check / available / limited / invalid / not_stored"),
		field.String("task_status").
			MaxLen(30).
			Default("pending").
			Comment("任务状态：pending / registering / importing / parsing / stored / register_failed / risk_rejected / duplicate / ip_unavailable / manual_review"),
		field.Text("task_message").
			Optional().
			Nillable().
			Comment("任务状态描述"),

		// 来源
		field.String("source").
			MaxLen(30).
			Default("manual_import").
			Comment("账号来源：registered / manual_import / file_upload"),

		// 默认执行代理快照（过渡字段，后续可收敛为 default_proxy_snapshot）
		field.Text("bound_ip").
			Optional().
			Nillable().
			Comment("默认执行代理快照"),

		// 分配信息
		field.Int64("assigned_user_id").
			Optional().
			Nillable().
			Comment("分配给的用户 ID"),

		// 备注
		field.Text("remark").
			Optional().
			Nillable().
			Comment("备注"),
	}
}

func (SocialAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("assigned_user", User.Type).
			Ref("social_accounts").
			Field("assigned_user_id").
			Unique(),
		edge.To("task_logs", SocialTaskLog.Type),
	}
}

func (SocialAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("platform"),
		index.Fields("account_status"),
		index.Fields("task_status"),
		index.Fields("assigned_user_id"),
		index.Fields("source"),
		index.Fields("deleted_at"),
		index.Fields("platform", "account_status"),
		index.Fields("platform_key", "name_key").
			Unique().
			Annotations(entsql.IndexWhere("deleted_at IS NULL")),
	}
}
