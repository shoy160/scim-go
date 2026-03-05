package util

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"scim-go/model"
	"time"
)

// GenerateVersion 生成 SCIM 版本标识符（ETag 格式）
func GenerateVersion() string {
	// 使用当前时间戳生成唯一版本标识
	timestamp := time.Now().UnixNano()
	hash := sha1.New()
	hash.Write([]byte(fmt.Sprintf("%d", timestamp)))
	return fmt.Sprintf("W/\"%s\"", hex.EncodeToString(hash.Sum(nil)))
}

// PopulateMeta 填充通用meta数据
func PopulateMeta(resourceType string, id string, createdAt, updatedAt time.Time, version string, baseURL string, apiPath string) model.Meta {
	meta := model.Meta{
		ResourceType: resourceType,
		Version:      version,
	}

	// 从数据库时间戳生成 ISO 8601 格式时间（使用纳秒精度）
	if !createdAt.IsZero() {
		meta.Created = createdAt.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	if !updatedAt.IsZero() {
		meta.LastModified = updatedAt.Format("2006-01-02T15:04:05.999999999Z07:00")
	}

	// 动态生成 location
	meta.Location = baseURL + apiPath + "/" + resourceType + "s/" + id

	return meta
}

// GenerateUserMeta 生成用户meta数据
func GenerateUserMeta(baseURL string, userID string, createdAt, updatedAt time.Time) model.Meta {
	return PopulateMeta("User", userID, createdAt, updatedAt, GenerateVersion(), baseURL, "/scim/v2")
}

// GenerateGroupMeta 生成组meta数据
func GenerateGroupMeta(baseURL string, groupID string, createdAt, updatedAt time.Time) model.Meta {
	return PopulateMeta("Group", groupID, createdAt, updatedAt, GenerateVersion(), baseURL, "/scim/v2")
}

// UpdateMeta 更新meta数据
func UpdateMeta(meta *model.Meta, baseURL string, id string, resourceType string) {
	meta.LastModified = time.Now().Format("2006-01-02T15:04:05.999999999Z07:00")
	meta.Version = GenerateVersion()
	meta.Location = baseURL + "/scim/v2/" + resourceType + "s/" + id
}

// GenerateMemberRef 生成成员的$ref属性
func GenerateMemberRef(member *model.Member, baseURL string, apiPath string) {
	if member.Type == "Group" {
		member.Ref = baseURL + apiPath + "/Groups/" + member.Value
	} else {
		// 默认类型为 User
		member.Type = "User"
		member.Ref = baseURL + apiPath + "/Users/" + member.Value
	}
}

// GenerateMembersRef 生成多个成员的$ref属性
func GenerateMembersRef(members []model.Member, baseURL string, apiPath string) {
	for i := range members {
		GenerateMemberRef(&members[i], baseURL, apiPath)
	}
}
