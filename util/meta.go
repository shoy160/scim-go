package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"scim-go/model"
	"time"
)

// GenerateUserMeta 生成用户的 meta 属性
func GenerateUserMeta(baseURL, userID string, createdAt, updatedAt time.Time) model.Meta {
	location := fmt.Sprintf("%s/Users/%s", baseURL, userID)
	created := createdAt.Format(time.RFC3339)
	lastModified := updatedAt.Format(time.RFC3339)
	version := generateVersion(userID, lastModified)

	return model.Meta{
		Created:      created,
		LastModified: lastModified,
		Location:     location,
		ResourceType: "User",
		Version:      version,
	}
}

// GenerateGroupMeta 生成组的 meta 属性
func GenerateGroupMeta(baseURL, groupID string, createdAt, updatedAt time.Time) model.Meta {
	location := fmt.Sprintf("%s/Groups/%s", baseURL, groupID)
	created := createdAt.Format(time.RFC3339)
	lastModified := updatedAt.Format(time.RFC3339)
	version := generateVersion(groupID, lastModified)

	return model.Meta{
		Created:      created,
		LastModified: lastModified,
		Location:     location,
		ResourceType: "Group",
		Version:      version,
	}
}

// UpdateMeta 更新 meta 属性的最后修改时间和版本
func UpdateMeta(meta *model.Meta, baseURL, resourceID, resourceType string) {
	now := time.Now()
	lastModified := now.Format(time.RFC3339)
	location := fmt.Sprintf("%s/%s/%s", baseURL, resourceType, resourceID)
	version := generateVersion(resourceID, lastModified)

	meta.LastModified = lastModified
	meta.Location = location
	meta.Version = version
	if meta.ResourceType == "" {
		meta.ResourceType = resourceType
	}
}

// GenerateVersion 生成版本标识
func GenerateVersion() string {
	// 使用当前时间生成版本标识（使用纳秒精度确保唯一性）
	now := time.Now().Format(time.RFC3339Nano)
	hash := md5.Sum([]byte(now))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("W/\"%s\"", hashStr)
}

// generateVersion 生成版本标识（内部使用）
func generateVersion(resourceID, lastModified string) string {
	// 使用资源ID和最后修改时间生成MD5哈希作为版本标识
	data := resourceID + lastModified
	hash := md5.Sum([]byte(data))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("W/\"%s\"", hashStr)
}
