package store

import (
	"context"
	"encoding/json"
	"errors"
	"scim-go/model"

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

// AddUserToGroup 添加用户到组
func (r *RedisStore) AddUserToGroup(groupID, userID string) error {
	// 验证组是否存在
	group, err := r.GetGroup(groupID, false)
	if err != nil {
		return err
	}

	// 验证用户是否存在
	_, err = r.GetUser(userID)
	if err != nil {
		return err
	}

	// 检查用户是否已在组中
	for _, member := range group.Members {
		if member.Value == userID {
			return errors.New("user already in group")
		}
	}

	// 添加成员
	group.Members = append(group.Members, model.Member{
		GroupID: groupID,
		Value:   userID,
	})

	return r.CreateGroup(group)
}

// RemoveUserFromGroup 从组中移除用户
func (r *RedisStore) RemoveUserFromGroup(groupID, userID string) error {
	// 验证组是否存在
	group, err := r.GetGroup(groupID, false)
	if err != nil {
		return err
	}

	// 查找并移除成员
	found := false
	var newMembers []model.Member
	for _, member := range group.Members {
		if member.Value == userID {
			found = true
		} else {
			newMembers = append(newMembers, member)
		}
	}

	if !found {
		return errors.New("user not in group")
	}

	group.Members = newMembers
	return r.CreateGroup(group)
}

// IsUserInGroup 检查用户是否在组中
func (r *RedisStore) IsUserInGroup(groupID, userID string) (bool, error) {
	group, err := r.GetGroup(groupID, false)
	if err != nil {
		return false, err
	}

	for _, member := range group.Members {
		if member.Value == userID {
			return true, nil
		}
	}

	return false, nil
}

// GetUserGroups 获取用户所属的所有组
func (r *RedisStore) GetUserGroups(userID string) ([]model.UserGroup, error) {
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
			if member.Value == userID {
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
