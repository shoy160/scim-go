package store

import (
	"encoding/json"
	"errors"
	"reflect"
	"scim-go/model"
	"scim-go/util"
	"sort"
	"time"
)

// Resource 通用资源接口
type Resource interface {
	GetID() string
	GetSchemas() []string
	SetSchemas([]string)
	GetMeta() *model.Meta
	SetMeta(*model.Meta)
	GetCreatedAt() time.Time
	SetCreatedAt(time.Time)
	GetUpdatedAt() time.Time
	SetUpdatedAt(time.Time)
	GetVersion() string
	SetVersion(string)
}

// ResourceStore 通用资源存储接口
type ResourceStore interface {
	Get(id string) (Resource, error)
	Create(resource Resource) error
	Update(resource Resource) error
	Delete(id string) error
	List(query *model.ResourceQuery) ([]Resource, int64, error)
}

// BaseResource 基础资源结构
type BaseResource struct {
	ID        string      `json:"id"`
	Schemas   []string    `json:"schemas"`
	Meta      *model.Meta `json:"meta,omitempty"`
	CreatedAt time.Time   `json:"createdAt,omitempty"`
	UpdatedAt time.Time   `json:"updatedAt,omitempty"`
	Version   string      `json:"version,omitempty"`
}

// GetID 获取资源ID
func (r *BaseResource) GetID() string {
	return r.ID
}

// GetSchemas 获取资源schemas
func (r *BaseResource) GetSchemas() []string {
	return r.Schemas
}

// SetSchemas 设置资源schemas
func (r *BaseResource) SetSchemas(schemas []string) {
	r.Schemas = schemas
}

// GetMeta 获取资源meta
func (r *BaseResource) GetMeta() *model.Meta {
	return r.Meta
}

// SetMeta 设置资源meta
func (r *BaseResource) SetMeta(meta *model.Meta) {
	r.Meta = meta
}

// GetCreatedAt 获取资源创建时间
func (r *BaseResource) GetCreatedAt() time.Time {
	return r.CreatedAt
}

// SetCreatedAt 设置资源创建时间
func (r *BaseResource) SetCreatedAt(t time.Time) {
	r.CreatedAt = t
}

// GetUpdatedAt 获取资源更新时间
func (r *BaseResource) GetUpdatedAt() time.Time {
	return r.UpdatedAt
}

// SetUpdatedAt 设置资源更新时间
func (r *BaseResource) SetUpdatedAt(t time.Time) {
	r.UpdatedAt = t
}

// GetVersion 获取资源版本
func (r *BaseResource) GetVersion() string {
	return r.Version
}

// SetVersion 设置资源版本
func (r *BaseResource) SetVersion(version string) {
	r.Version = version
}

// GenerateMeta 生成资源的meta属性
func GenerateMeta(resourceType string) (model.Meta, time.Time, time.Time) {
	now := time.Now()
	createdAt := now
	updatedAt := now
	created := createdAt.Format(time.RFC3339Nano)
	lastModified := updatedAt.Format(time.RFC3339Nano)
	version := util.GenerateVersion()

	meta := model.Meta{
		ResourceType: resourceType,
		Created:      created,
		LastModified: lastModified,
		Location:     "", // 由API层动态生成
		Version:      version,
	}

	return meta, createdAt, updatedAt
}

// UpdateMeta 更新资源的meta属性
func UpdateMeta(meta *model.Meta) string {
	now := time.Now()
	meta.LastModified = now.Format(time.RFC3339Nano)
	meta.Version = util.GenerateVersion()
	return meta.Version
}

// Paginate 通用分页函数
func Paginate(items interface{}, startIndex, count int) interface{} {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil
	}

	start := startIndex - 1
	if start < 0 {
		start = 0
	}
	if start >= v.Len() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}

	end := start + count
	if end > v.Len() {
		end = v.Len()
	}

	return v.Slice(start, end).Interface()
}

// Filter 通用过滤函数
func Filter(items interface{}, filter string, toMap func(interface{}) (map[string]interface{}, error)) (interface{}, error) {
	if filter == "" {
		return items, nil
	}

	node, err := util.ParseFilter(filter)
	if err != nil {
		return nil, err
	}

	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil, errors.New("items must be a slice")
	}

	// 预分配切片容量
	result := reflect.MakeSlice(v.Type(), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		obj, err := toMap(item)
		if err != nil {
			return nil, err
		}

		match, err := util.MatchFilter(node, obj)
		if err != nil {
			return nil, err
		}

		if match {
			result = reflect.Append(result, v.Index(i))
		}
	}

	return result.Interface(), nil
}

// Sort 通用排序函数
func Sort(items interface{}, sortBy, sortOrder string, compare func(i, j int) bool) {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice || !v.CanAddr() {
		return
	}

	less := func(i, j int) bool {
		if sortOrder == "descending" {
			return !compare(i, j)
		}
		return compare(i, j)
	}

	sort.Slice(items, less)
}

// ToMap 通用转换为map函数
func ToMap(obj interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// HandleAddOrReplace 处理 add 和 replace 操作的通用函数
func HandleAddOrReplace(s Store, id string, data any, op model.PatchOperation, isGroup, isUser bool) error {
	// 成员字段的处理（Group 类型）
	if op.Path == "members" && isGroup {
		return handleAddMembers(s, id, op.Value)
	}
	// 群组字段的处理（User 类型）
	if op.Path == "groups" && isUser {
		return handleAddGroups(s, id, op.Value)
	}
	// emails 字段的处理（User 类型）
	if op.Path == "emails" && isUser {
		if op.Op == "add" {
			return handleAddEmails(data, op.Value)
		}
		return handleReplaceEmails(data, op.Value)
	}
	// roles 字段的处理（User 类型）
	if op.Path == "roles" && isUser {
		if op.Op == "add" {
			return handleAddRoles(data, op.Value)
		}
		return handleReplaceRoles(data, op.Value)
	}
	// 通用字段处理
	if op.Path != "" {
		return util.SetValueByPath(data, op.Path, op.Value)
	}
	// 如果 Path 为空，Value 应该是一个对象，需要合并到 data 中
	if op.Value != nil {
		return util.MergeValue(data, op.Value)
	}
	return nil
}

// HandleRemove 处理 remove 操作的通用函数
func HandleRemove(s Store, id string, data any, op model.PatchOperation, isGroup, isUser bool) error {
	// 成员字段的处理（Group 类型）
	if op.Path == "members" && isGroup {
		return handleRemoveMembers(s, id, op.Value)
	}
	// 群组字段的处理（User 类型）
	if op.Path == "groups" && isUser {
		return handleRemoveGroups(s, id, op.Value)
	}
	// emails 字段的处理（User 类型）
	if op.Path == "emails" && isUser {
		return handleRemoveEmails(s, id, data, op.Value)
	}
	// roles 字段的处理（User 类型）
	if op.Path == "roles" && isUser {
		return handleRemoveRoles(s, id, data, op.Value)
	}
	// 通用字段处理
	if op.Path != "" {
		return util.RemoveByPath(data, op.Path)
	}
	return nil
}
