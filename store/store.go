package store

import (
	"log"
	"scim-go/model"
	"scim-go/util"
)

// Store 存储层通用接口（所有存储实现需实现此接口）
type Store interface {
	// ---------------------- User 相关 ----------------------
	CreateUser(u *model.User) error
	GetUser(id string) (*model.User, error)
	ListUsers(q *model.ResourceQuery) ([]model.User, int64, error)
	UpdateUser(u *model.User) error                        // 全量更新（PUT）
	PatchUser(id string, ops []model.PatchOperation) error // 补丁更新（PATCH）
	DeleteUser(id string) error

	// ---------------------- Group 相关 ----------------------
	CreateGroup(g *model.Group) error
	GetGroup(id string, preloadMembers bool) (*model.Group, error)
	ListGroups(q *model.ResourceQuery, preloadMembers bool) ([]model.Group, int64, error)
	UpdateGroup(g *model.Group) error                       // 全量更新（PUT）
	PatchGroup(id string, ops []model.PatchOperation) error // 补丁更新（PATCH）
	DeleteGroup(id string) error

	// ---------------------- Group 成员管理 ----------------------
	AddUserToGroup(groupID, userID string) error        // 添加用户到组
	RemoveUserFromGroup(groupID, userID string) error   // 从组中移除用户
	IsUserInGroup(groupID, userID string) (bool, error) // 检查用户是否在组中

	// ---------------------- 反向查询 ----------------------
	GetUserGroups(userID string) ([]model.UserGroup, error) // 获取用户所属的所有组
}

func PatchResource(s Store, id string, data any, ops []model.PatchOperation) error {
	if len(ops) == 0 {
		return nil
	}
	var isGroup bool
	if _, ok := data.(*model.Group); ok {
		isGroup = true
	}
	var isUser bool
	if _, ok := data.(*model.User); ok {
		isUser = true
	}
	log.Println("isGroup", isGroup, "isUser", isUser, data)

	for _, op := range ops {
		log.Println("op", op)
		switch op.Op {
		case "add", "replace":
			// 成员字段的处理
			if op.Path == "members" && isGroup {
				log.Println("members", op.Value)
				for _, member := range op.Value.([]any) {
					log.Println("add member", member)
					err := s.AddUserToGroup(id, member.(map[string]any)["value"].(string))
					if err != nil {
						return err
					}
				}
				continue
			}
			// 群组字段的处理
			if op.Path == "groups" && isUser {
				for _, group := range op.Value.([]any) {
					log.Println("add group", group)
					err := s.AddUserToGroup(group.(map[string]any)["value"].(string), id)
					if err != nil {
						return err
					}
				}
				continue
			}
			util.SetValueByPath(data, op.Path, op.Value)
		case "remove":
			// 成员字段的处理
			if op.Path == "members" && isGroup {
				for _, member := range op.Value.([]any) {
					log.Println("remove member", member)
					err := s.RemoveUserFromGroup(id, member.(map[string]any)["value"].(string))
					if err != nil {
						return err
					}
				}
				continue
			}
			// 群组字段的处理
			if op.Path == "groups" && isUser {
				for _, group := range op.Value.([]any) {
					log.Println("remove group", group)
					err := s.RemoveUserFromGroup(group.(map[string]any)["value"].(string), id)
					if err != nil {
						return err
					}
				}
				continue
			}
			util.RemoveByPath(data, op.Path)
		}
	}
	return nil
}
