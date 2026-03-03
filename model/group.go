package model

// Group SCIM 2.0标准组模型（RFC 7644）
type Group struct {
	Schemas     []string `json:"schemas" gorm:"-"`
	ID          string   `json:"id" gorm:"primaryKey;type:varchar(64)"`
	ExternalID  string   `json:"externalId,omitempty" gorm:"type:varchar(64);index"`
	DisplayName string   `json:"displayName" gorm:"type:varchar(128);uniqueIndex;not null"`
	// 组成员（多值属性，关联用户ID）
	Members []Member `json:"members,omitempty" gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE"`
}

// Member 组成员（关联用户）
type Member struct {
	GroupID string `json:"-" gorm:"column:group_id;type:varchar(64);index"`
	Value   string `json:"value" gorm:"type:varchar(64);not null"`     // 用户ID
	Display string `json:"display,omitempty" gorm:"type:varchar(128)"` // 用户名
}

// TableName 表名映射
func (g *Group) TableName() string  { return "scim_groups" }
func (m *Member) TableName() string { return "scim_group_members" }
