package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/Wei-Shaw/socialops/ent/schema/mixins"
)

// GlobalProxy holds administrator-managed proxies available for platform execution fallback.
type GlobalProxy struct {
	ent.Schema
}

func (GlobalProxy) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "global_proxies"},
	}
}

func (GlobalProxy) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (GlobalProxy) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			Comment("全局代理标签名称"),
		field.String("ip_type").
			MaxLen(30).
			Default("residential").
			Comment("类型：residential / static / mobile / datacenter / dynamic"),
		field.String("endpoint").
			MaxLen(500).
			Optional().
			Nillable().
			Comment("代理端点或动态提取链接"),
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
		field.Time("last_used_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("最近一次被执行兜底使用时间"),
		field.Text("remark").
			Optional().
			Nillable().
			Comment("备注"),
	}
}

func (GlobalProxy) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("ip_type"),
		index.Fields("deleted_at"),
		index.Fields("status", "last_used_at"),
	}
}
