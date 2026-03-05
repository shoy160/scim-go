package model

// MemberType 成员类型枚举
type MemberType string

// 成员类型常量
const (
	MemberTypeUser  MemberType = "User"  // 用户类型
	MemberTypeGroup MemberType = "Group" // 组类型
)

// String 返回成员类型的字符串表示
func (mt MemberType) String() string {
	return string(mt)
}

// IsValid 检查成员类型是否有效
func (mt MemberType) IsValid() bool {
	return mt == MemberTypeUser || mt == MemberTypeGroup
}

// ParseMemberType 解析字符串为成员类型
func ParseMemberType(s string) (MemberType, bool) {
	mt := MemberType(s)
	return mt, mt.IsValid()
}
