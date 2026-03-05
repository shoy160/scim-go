package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"scim-go/model"
	"strings"

	"github.com/go-redis/redis/v8"
)

// RedisStore Redis存储实现
type RedisStore struct {
	cli *redis.Client
	ctx context.Context
}

// 缓存key前缀
const (
	prefixUser  = "scim:user:"
	prefixGroup = "scim:group:"
)

// NewRedis 创建Redis存储实例
func NewRedis(uri string) Store {
	opt, err := redis.ParseURL(uri)
	if err != nil {
		panic("redis parse url failed: " + err.Error())
	}
	cli := redis.NewClient(opt)
	// 测试连接
	_, err = cli.Ping(context.Background()).Result()
	if err != nil {
		panic("redis connect failed: " + err.Error())
	}
	return &RedisStore{
		cli: cli,
		ctx: context.Background(),
	}
}

// ---------------------- User 实现 ----------------------
func (r *RedisStore) CreateUser(u *model.User) error {
	b, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return r.cli.Set(r.ctx, prefixUser+u.ID, b, 0).Err()
}

func (r *RedisStore) GetUser(id string) (*model.User, error) {
	b, err := r.cli.Get(r.ctx, prefixUser+id).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	var u model.User
	err = json.Unmarshal(b, &u)
	return &u, err
}

func (r *RedisStore) ListUsers(_ *model.ResourceQuery) ([]model.User, int64, error) {
	keys, err := r.cli.Keys(r.ctx, prefixUser+"*").Result()
	if err != nil {
		return nil, 0, err
	}
	var list []model.User
	for _, k := range keys {
		b, _ := r.cli.Get(r.ctx, k).Bytes()
		var u model.User
		json.Unmarshal(b, &u)
		list = append(list, u)
	}
	return list, int64(len(list)), nil
}

func (r *RedisStore) UpdateUser(u *model.User) error {
	return r.CreateUser(u)
}

func (r *RedisStore) PatchUser(id string, ops []model.PatchOperation) error {
	u, err := r.GetUser(id)
	if err != nil {
		return err
	}
	err = PatchResource(r, id, u, ops)
	if err != nil {
		return err
	}
	return r.CreateUser(u)
}

func (r *RedisStore) DeleteUser(id string) error {
	return r.cli.Del(r.ctx, prefixUser+id).Err()
}

// ---------------------- Group 实现 ----------------------
func (r *RedisStore) CreateGroup(g *model.Group) error {
	b, err := json.Marshal(g)
	if err != nil {
		return err
	}
	return r.cli.Set(r.ctx, prefixGroup+g.ID, b, 0).Err()
}

func (r *RedisStore) GetGroup(id string, preloadMembers bool) (*model.Group, error) {
	b, err := r.cli.Get(r.ctx, prefixGroup+id).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("group not found")
		}
		return nil, err
	}
	var g model.Group
	err = json.Unmarshal(b, &g)
	return &g, err
}

func (r *RedisStore) ListGroups(_ *model.ResourceQuery, preloadMembers bool) ([]model.Group, int64, error) {
	keys, err := r.cli.Keys(r.ctx, prefixGroup+"*").Result()
	if err != nil {
		return nil, 0, err
	}
	var list []model.Group
	for _, k := range keys {
		b, _ := r.cli.Get(r.ctx, k).Bytes()
		var g model.Group
		json.Unmarshal(b, &g)
		list = append(list, g)
	}
	return list, int64(len(list)), nil
}

func (r *RedisStore) UpdateGroup(g *model.Group) error {
	return r.CreateGroup(g)
}

func (r *RedisStore) PatchGroup(id string, ops []model.PatchOperation) error {
	g, err := r.GetGroup(id, false)
	if err != nil {
		return err
	}
	err = PatchResource(r, id, g, ops)
	if err != nil {
		return err
	}
	return r.CreateGroup(g)
}

func (r *RedisStore) DeleteGroup(id string) error {
	return r.cli.Del(r.ctx, prefixGroup+id).Err()
}

// ---------------------- Group 成员管理 ----------------------

// AddMemberToGroup 添加成员到组（支持用户和组）
func (r *RedisStore) AddMemberToGroup(groupID, memberID string, memberType ...model.MemberType) error {
	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	// 验证组是否存在
	group, err := r.GetGroup(groupID, false)
	if err != nil {
		return err
	}

	// 验证成员是否存在
	if mt == "User" {
		_, err = r.GetUser(memberID)
		if err != nil {
			return err
		}
	} else if mt == "Group" {
		_, err = r.GetGroup(memberID, false)
		if err != nil {
			return err
		}
	} else {
		return model.ErrInvalidValue
	}

	// 检查成员是否已在组中
	for _, member := range group.Members {
		if member.Value == memberID {
			return model.ErrMemberAlreadyInGroup
		}
	}

	// 添加成员
	group.Members = append(group.Members, model.Member{
		GroupID: groupID,
		Value:   memberID,
		Type:    mt,
	})

	return r.CreateGroup(group)
}

// RemoveMemberFromGroup 从组中移除成员（支持用户和组）
func (r *RedisStore) RemoveMemberFromGroup(groupID, memberID string, memberType ...model.MemberType) error {
	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	// 验证组是否存在
	group, err := r.GetGroup(groupID, false)
	if err != nil {
		return err
	}

	// 查找并移除成员
	found := false
	var newMembers []model.Member
	for _, member := range group.Members {
		memberType := member.Type
		if memberType == "" {
			memberType = model.MemberTypeUser
		}
		if member.Value == memberID && (mt == "" || memberType == mt) {
			found = true
		} else {
			newMembers = append(newMembers, member)
		}
	}

	if !found {
		return model.ErrMemberNotInGroup
	}

	group.Members = newMembers
	return r.CreateGroup(group)
}

// GetGroupMembers 获取组成员（支持分页和类型过滤）
func (r *RedisStore) GetGroupMembers(groupID string, memberType model.MemberType, q *model.ResourceQuery) ([]model.Member, int64, error) {
	group, err := r.GetGroup(groupID, false)
	if err != nil {
		return nil, 0, err
	}

	var filteredMembers []model.Member
	for _, member := range group.Members {
		if memberType == "" || member.Type == memberType {
			filteredMembers = append(filteredMembers, member)
		}
	}

	total := int64(len(filteredMembers))

	startIndex := q.StartIndex
	count := q.Count

	if startIndex < 1 {
		startIndex = 1
	}
	if count <= 0 {
		count = len(filteredMembers)
	}

	start := startIndex - 1
	end := start + count
	if start > len(filteredMembers) {
		return []model.Member{}, total, nil
	}
	if end > len(filteredMembers) {
		end = len(filteredMembers)
	}

	return filteredMembers[start:end], total, nil
}

// IsMemberInGroup 检查成员是否在组中（支持用户和组）
func (r *RedisStore) IsMemberInGroup(groupID, memberID string, memberType ...model.MemberType) (bool, error) {
	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	group, err := r.GetGroup(groupID, false)
	if err != nil {
		return false, err
	}

	for _, member := range group.Members {
		memberType := member.Type
		if memberType == "" {
			memberType = model.MemberTypeUser
		}
		if member.Value == memberID && (mt == "" || memberType == mt) {
			return true, nil
		}
	}

	return false, nil
}

// GetMemberGroups 获取成员所属的所有组（支持用户和组）
func (r *RedisStore) GetMemberGroups(memberID string, memberType ...model.MemberType) ([]model.UserGroup, error) {
	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	keys, err := r.cli.Keys(r.ctx, prefixGroup+"*").Result()
	if err != nil {
		return nil, err
	}

	var groups []model.UserGroup
	for _, k := range keys {
		b, err := r.cli.Get(r.ctx, k).Bytes()
		if err != nil {
			continue
		}
		var g model.Group
		if err := json.Unmarshal(b, &g); err != nil {
			continue
		}

		for _, member := range g.Members {
			memberType := member.Type
			if memberType == "" {
				memberType = "User"
			}
			if member.Value == memberID && (mt == "" || memberType == mt) {
				groups = append(groups, model.UserGroup{
					Value:   g.ID,
					Display: g.DisplayName,
				})
				break
			}
		}
	}

	return groups, nil
}

// RemoveEmailFromUser 从用户中移除指定邮箱
func (r *RedisStore) RemoveEmailFromUser(userID, emailValue string) error {
	key := fmt.Sprintf("user:%s:emails", userID)
	_, err := r.cli.HDel(r.ctx, key, strings.ToLower(emailValue)).Result()
	if err != nil {
		return err
	}
	// 如果找不到记录，返回 nil 而不是错误（可能已经被删除）
	return nil
}

// RemoveRoleFromUser 从用户中移除指定角色
func (r *RedisStore) RemoveRoleFromUser(userID, roleValue string) error {
	key := fmt.Sprintf("user:%s:roles", userID)
	_, err := r.cli.HDel(r.ctx, key, strings.ToLower(roleValue)).Result()
	if err != nil {
		return err
	}
	// 如果找不到记录，返回 nil 而不是错误（可能已经被删除）
	return nil
}
