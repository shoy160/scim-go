package model

import "time"

// Group SCIM 2.0标准组模型（RFC 7644）
type Group struct {
	Schemas     StringList `json:"schemas" gorm:"type:json;serializer:json"` // 存储SCIM schemas，包括自定义schemas
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(64)"`
	ExternalID  string     `json:"externalId,omitempty" gorm:"type:varchar(64);index"`
	DisplayName string     `json:"displayName" gorm:"type:varchar(128);uniqueIndex;not null"`
	Meta        Meta       `json:"meta,omitempty" gorm:"-"` // ResourceType由Meta.ResourceType动态生成，不持久化
	// 组成员（多值属性，关联用户ID）
	Members []Member `json:"members,omitempty" gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE"`

	// SCIM Meta 数据字段（数据库存储）
	// ResourceType 不持久化，由API层根据资源类型动态生成
	CreatedAt time.Time `json:"-" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"-" gorm:"column:updated_at;autoUpdateTime"`
	Version   string    `json:"-" gorm:"column:version;type:varchar(64)"`
}

// Member 组成员（关联用户或组）
type Member struct {
	GroupID string     `json:"-" gorm:"column:group_id;type:varchar(64);index"`
	Value   string     `json:"value" gorm:"type:varchar(64);not null"`              // 成员ID（用户或组）
	Display string     `json:"display,omitempty" gorm:"type:varchar(128)"`          // 成员显示名称
	Type    MemberType `json:"type,omitempty" gorm:"type:varchar(32);default:User"` // 成员类型（User或Group）
	Ref     string     `json:"$ref,omitempty" gorm:"-"`                             // 成员引用URI（动态生成）
}

// TableName 表名映射
func (g *Group) TableName() string  { return "scim_groups" }
func (m *Member) TableName() string { return "scim_group_members" }
