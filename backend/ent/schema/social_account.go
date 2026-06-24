package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
	"github.com/Wei-Shaw/socialops/internal/pkg/socialidentity"
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
			Comment("Normalized username key for display/search and username fallback identity"),
		field.String("identity_kind").
			MaxLen(30).
			Default("username").
			Comment("Business identity type. SocialOps account identity is username-based."),
		field.String("identity_key").
			MaxLen(100).
			Comment("Normalized username identity key for pool uniqueness"),
		field.String("platform_user_id").
			MaxLen(100).
			Optional().
			Nillable().
			Comment("Platform user ID / rest_id, e.g. Twitter/X numeric user ID"),
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
		field.String("two_factor").
			MaxLen(1024).
			Optional().
			Nillable().
			Comment("Account two-factor secret or code"),
		field.String("backup_code").
			MaxLen(1024).
			Optional().
			Nillable().
			Comment("Account backup/recovery code"),
		field.String("email_client_id").
			MaxLen(1024).
			Optional().
			Nillable().
			Comment("Email OAuth/client identifier used for account support workflows"),
		field.Text("email_token").
			Optional().
			Nillable().
			Comment("Email token used for account support workflows"),
		field.String("registration_ip").
			MaxLen(255).
			Optional().
			Nillable().
			Comment("IP used when the social account was registered"),

		// 执行凭证与账号状态
		field.Text("auth_cookie").
			Optional().
			Nillable().
			Comment("Platform auth cookie captured for account operations"),
		field.Text("execution_auth").
			Optional().
			Nillable().
			Comment("Encrypted platform execution authentication"),
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

		// 默认执行代理快照
		field.Text("default_proxy_snapshot").
			Optional().
			Nillable().
			Comment("Default execution proxy snapshot JSON"),

		// 分配信息
		field.Int64("assigned_user_id").
			Optional().
			Nillable().
			Comment("分配给的用户 ID"),

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

func (SocialAccount) Hooks() []ent.Hook {
	return []ent.Hook{
		func(next ent.Mutator) ent.Mutator {
			return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
				if err := applySocialAccountIdentityMutation(ctx, m); err != nil {
					return nil, err
				}
				return next.Mutate(ctx, m)
			})
		},
	}
}

func (SocialAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("platform"),
		index.Fields("account_status"),
		index.Fields("task_status"),
		index.Fields("assigned_user_id"),
		index.Fields("platform", "account_status"),
		index.Fields("platform_user_id").
			StorageKey("idx_social_accounts_platform_user_id"),
		index.Fields("platform_key", "name_key").
			StorageKey("idx_social_accounts_platform_name_key_unique").
			Unique(),
	}
}

type socialAccountIdentityMutation interface {
	Name() (string, bool)
	Platform() (string, bool)
	PlatformKey() (string, bool)
	NameKey() (string, bool)
	IdentityKind() (string, bool)
	IdentityKey() (string, bool)
	SetPlatformKey(string)
	SetNameKey(string)
	SetIdentityKind(string)
	SetIdentityKey(string)
	Op() ent.Op
}

type socialAccountIdentityUpdateMutation interface {
	socialAccountIdentityMutation
	OldName(context.Context) (string, error)
	OldPlatform(context.Context) (string, error)
	OldPlatformKey(context.Context) (string, error)
	OldNameKey(context.Context) (string, error)
	OldIdentityKind(context.Context) (string, error)
	OldIdentityKey(context.Context) (string, error)
}

func applySocialAccountIdentityMutation(ctx context.Context, m ent.Mutation) error {
	mx, ok := m.(socialAccountIdentityMutation)
	if !ok || !m.Op().Is(ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne) {
		return nil
	}
	platform, _ := mx.Platform()
	name, _ := mx.Name()
	if update, ok := mx.(socialAccountIdentityUpdateMutation); ok && !m.Op().Is(ent.OpCreate) {
		if platform == "" {
			if oldPlatform, err := update.OldPlatform(ctx); err == nil {
				platform = oldPlatform
			}
		}
		if name == "" {
			if oldName, err := update.OldName(ctx); err == nil {
				name = oldName
			}
		}
	}
	identity := socialidentity.Build(platform, name)
	if identity.PlatformKey != "" {
		mx.SetPlatformKey(identity.PlatformKey)
	} else if current, ok := mx.PlatformKey(); ok && current != "" {
		identity.PlatformKey = current
	}
	if identity.NameKey != "" {
		mx.SetNameKey(identity.NameKey)
	} else if current, ok := mx.NameKey(); ok && current != "" {
		identity.NameKey = current
	}
	if identity.IdentityKey == "" {
		identity.IdentityKind, identity.IdentityKey = fallbackSocialAccountIdentity(ctx, mx)
	}
	if identity.IdentityKey != "" {
		mx.SetIdentityKind(identity.IdentityKind)
		mx.SetIdentityKey(identity.IdentityKey)
	}
	return nil
}

func fallbackSocialAccountIdentity(ctx context.Context, mx socialAccountIdentityMutation) (string, string) {
	if currentKind, ok := mx.IdentityKind(); ok {
		if currentKey, ok := mx.IdentityKey(); ok && currentKey != "" {
			return currentKind, currentKey
		}
	}
	if update, ok := mx.(socialAccountIdentityUpdateMutation); ok && !mx.Op().Is(ent.OpCreate) {
		if oldKind, err := update.OldIdentityKind(ctx); err == nil {
			if oldKey, err := update.OldIdentityKey(ctx); err == nil && oldKey != "" {
				return oldKind, oldKey
			}
		}
		if oldNameKey, err := update.OldNameKey(ctx); err == nil && oldNameKey != "" {
			return socialidentity.KindUsername, oldNameKey
		}
	}
	if nameKey, ok := mx.NameKey(); ok && nameKey != "" {
		return socialidentity.KindUsername, nameKey
	}
	return socialidentity.KindUsername, ""
}
