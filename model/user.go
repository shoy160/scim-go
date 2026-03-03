package model

import "time"

// User SCIM 2.0标准用户模型（RFC 7644）
type User struct {
	ID          string   `json:"id" gorm:"primaryKey;type:varchar(64)"`
	Schemas     []string `json:"schemas" gorm:"-"`
	ExternalID  string   `json:"externalId,omitempty" gorm:"type:varchar(64);index"`
	UserName    string   `json:"userName" gorm:"type:varchar(64);uniqueIndex;not null"`
	Active      bool     `json:"active" gorm:"default:true"`
	DisplayName string   `json:"displayName,omitempty" gorm:"type:varchar(128)"`
	NickName    string   `json:"nickName,omitempty" gorm:"type:varchar(64)"`
	ProfileUrl  string   `json:"profileUrl,omitempty" gorm:"type:varchar(255)"`

	// 姓名信息（嵌套）
	Name struct {
		GivenName  string `json:"givenName" gorm:"column:given_name;type:varchar(64);not null"`
		FamilyName string `json:"familyName" gorm:"column:family_name;type:varchar(64);not null"`
		MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
	} `json:"name" gorm:"embedded"`

	// 邮箱（多值属性，关联表）
	Emails []Email `json:"emails" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	// 角色（SCIM标准）
	Roles []Role `json:"roles,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	// 用户所属的组（非数据库字段，动态填充）
	Groups []UserGroup `json:"groups,omitempty" gorm:"-"`
}

// UserGroup 用户所属的组信息
type UserGroup struct {
	Value   string `json:"value"`   // 组ID
	Display string `json:"display"` // 组显示名称
}

// Email 用户邮箱（多值属性）
type Email struct {
	ID        uint      `json:"-" gorm:"primaryKey"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	UserID    string    `json:"-" gorm:"column:user_id;type:varchar(64);index"`
	Value     string    `json:"value" gorm:"type:varchar(128);not null"`
	Type      string    `json:"type" gorm:"type:varchar(32);default:'work'"` // work/home/other
	Primary   bool      `json:"primary" gorm:"default:true"`
}

// Role 用户角色（SCIM标准）
type Role struct {
	ID        uint      `json:"-" gorm:"primaryKey"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	UserID    string    `json:"-" gorm:"column:user_id;type:varchar(64);index"`
	Value     string    `json:"value" gorm:"type:varchar(64);not null"` // 角色值
	Type      string    `json:"type,omitempty" gorm:"type:varchar(32)"`
	Display   string    `json:"display,omitempty" gorm:"type:varchar(64)"`
	Primary   bool      `json:"primary,omitempty" gorm:"default:true"`
}

// TableName 表名映射
func (u *User) TableName() string  { return "scim_users" }
func (e *Email) TableName() string { return "scim_user_emails" }
func (r *Role) TableName() string  { return "scim_user_roles" }
