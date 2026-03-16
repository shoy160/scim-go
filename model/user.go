package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// StringList 用于存储 JSON 数组的自定义类型
type StringList []string

// Value 实现 driver.Valuer 接口
func (s StringList) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *StringList) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// Manager SCIM 2.0 企业用户扩展中的manager字段
type Manager struct {
	Value string `json:"value,omitempty" gorm:"type:varchar(64);column:manager_value"`
	Ref   string `json:"$ref,omitempty" gorm:"-"` // 成员引用URI（动态生成）
}

// EnterpriseUserExtension SCIM 2.0 企业用户扩展模型（RFC 7643）
type EnterpriseUserExtension struct {
	EmployeeNumber string   `json:"employeeNumber,omitempty" gorm:"type:varchar(64);column:employee_number"`
	CostCenter     string   `json:"costCenter,omitempty" gorm:"type:varchar(64);column:cost_center"`
	Organization   string   `json:"organization,omitempty" gorm:"type:varchar(128);column:organization"`
	Division       string   `json:"division,omitempty" gorm:"type:varchar(128);column:division"`
	Department     string   `json:"department,omitempty" gorm:"type:varchar(128);column:department"`
	Manager        *Manager `json:"manager,omitempty" gorm:"embedded"`
}

// User SCIM 2.0标准用户模型（RFC 7644）
type User struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(64)"`
	Schemas     StringList `json:"schemas" gorm:"type:json;serializer:json"` // 存储SCIM schemas，包括自定义schemas
	ExternalID  string     `json:"externalId,omitempty" gorm:"type:varchar(64);index"`
	UserName    string     `json:"userName" gorm:"type:varchar(64);uniqueIndex;not null"`
	Active      bool       `json:"active" gorm:"default:true"`
	DisplayName string     `json:"displayName,omitempty" gorm:"type:varchar(128)"`
	NickName    string     `json:"nickName,omitempty" gorm:"type:varchar(64)"`
	ProfileUrl  string     `json:"profileUrl,omitempty" gorm:"type:varchar(255)"`
	Password    string     `json:"password,omitempty" gorm:"type:varchar(255);-"` // 密码，非必填，默认不返回
	Meta        Meta       `json:"meta,omitempty" gorm:"-"`                       // ResourceType由Meta.ResourceType动态生成，不持久化

	// SCIM Meta 数据字段（数据库存储）
	// ResourceType 不持久化，由API层根据资源类型动态生成
	CreatedAt time.Time `json:"-" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"-" gorm:"column:updated_at;autoUpdateTime"`
	Version   string    `json:"-" gorm:"column:version;type:varchar(64)"`

	// 姓名信息（嵌套）
	Name struct {
		Formatted  string `json:"formatted,omitempty" gorm:"column:formatted;type:varchar(255)"`
		GivenName  string `json:"givenName,omitempty" gorm:"column:given_name;type:varchar(64)"`
		FamilyName string `json:"familyName,omitempty" gorm:"column:family_name;type:varchar(64)"`
		MiddleName string `json:"middleName,omitempty" gorm:"column:middle_name;type:varchar(64)"`
	} `json:"name" gorm:"embedded"`

	// 企业扩展属性
	*EnterpriseUserExtension `json:"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User,omitempty" gorm:"embedded"`

	// 邮箱（多值属性，关联表）
	Emails []Email `json:"emails,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	// 电话号码（多值属性，关联表）
	PhoneNumbers []PhoneNumber `json:"phoneNumbers,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	// 地址（多值属性，关联表）
	Addresses []Address `json:"addresses,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	// 角色（SCIM标准）
	Roles []Role `json:"roles,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	// 用户所属的组（非数据库字段，动态填充）
	Groups []UserGroup `json:"groups,omitempty" gorm:"-"`
}

// UserGroup 用户所属的组信息
type UserGroup struct {
	Value   string `json:"value"`   // 组ID
	Ref     string `json:"$ref"`    // 组引用URI
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

// PhoneNumber 用户电话号码（多值属性）
type PhoneNumber struct {
	ID        uint      `json:"-" gorm:"primaryKey"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	UserID    string    `json:"-" gorm:"column:user_id;type:varchar(64);index"`
	Value     string    `json:"value" gorm:"type:varchar(32);not null"`
	Type      string    `json:"type" gorm:"type:varchar(32);default:'work'"` // work/home/mobile/other
	Primary   bool      `json:"primary" gorm:"default:true"`
}

// Address 用户地址（多值属性）
type Address struct {
	ID            uint      `json:"-" gorm:"primaryKey"`
	CreatedAt     time.Time `json:"-"`
	UpdatedAt     time.Time `json:"-"`
	UserID        string    `json:"-" gorm:"column:user_id;type:varchar(64);index"`
	Value         string    `json:"value,omitempty" gorm:"type:varchar(255)"`
	Display       string    `json:"display,omitempty" gorm:"type:varchar(128)"`
	StreetAddress string    `json:"streetAddress,omitempty" gorm:"type:varchar(128);column:street_address"`
	Locality      string    `json:"locality,omitempty" gorm:"type:varchar(64)"`
	Region        string    `json:"region,omitempty" gorm:"type:varchar(64)"`
	PostalCode    string    `json:"postalCode,omitempty" gorm:"type:varchar(32);column:postal_code"`
	Country       string    `json:"country,omitempty" gorm:"type:varchar(64)"`
	Type          string    `json:"type" gorm:"type:varchar(32);default:'work'"` // work/home/other
	Primary       bool      `json:"primary" gorm:"default:true"`
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
func (u *User) TableName() string        { return "scim_users" }
func (e *Email) TableName() string       { return "scim_user_emails" }
func (p *PhoneNumber) TableName() string { return "scim_user_phone_numbers" }
func (a *Address) TableName() string     { return "scim_user_addresses" }
func (r *Role) TableName() string        { return "scim_user_roles" }
