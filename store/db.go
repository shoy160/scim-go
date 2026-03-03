package store

import (
	"errors"
	"log"
	"scim-go/model"
	"scim-go/util"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DBStore 关系型数据库存储实现（MySQL/PG通用）
type DBStore struct {
	db *gorm.DB
}

// NewDB 创建数据库存储实例
func NewDB(db *gorm.DB) Store {
	// 自动迁移表结构（含关联表）
	err := db.AutoMigrate(
		&model.User{}, &model.Email{}, &model.Role{},
		&model.Group{}, &model.Member{},
	)
	if err != nil {
		panic("auto migrate table failed: " + err.Error())
	}
	return &DBStore{db: db}
}

// ---------------------- User 实现 ----------------------
func (d *DBStore) CreateUser(u *model.User) error {
	// 检查用户名唯一性
	var count int64
	d.db.Model(&model.User{}).Where("user_name = ?", u.UserName).Count(&count)
	if count > 0 {
		return model.ErrUniqueness
	}
	return d.db.Create(u).Error
}

func (d *DBStore) GetUser(id string) (*model.User, error) {
	var u model.User
	err := d.db.Preload("Emails").Preload("Roles").First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	return &u, err
}

func (d *DBStore) ListUsers(q *model.ResourceQuery) ([]model.User, int64, error) {
	var list []model.User
	var total int64

	// 基础查询（预加载关联）
	query := d.db.Preload("Emails").Preload("Roles").Model(&model.User{})

	// 应用过滤器
	if q.Filter != "" {
		filterQuery, args, err := d.applyUserFilter(q.Filter)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where(filterQuery, args...)
	}

	// 统计总数
	query.Count(&total)

	// 分页
	offset := (q.StartIndex - 1) * q.Count
	query.Offset(offset).Limit(q.Count)

	// 排序
	if q.SortBy != "" {
		sort := d.toSnakeCase(q.SortBy)
		if q.SortOrder == "descending" {
			sort += " DESC"
		} else {
			sort += " ASC"
		}
		query.Order(sort)
	}

	// 执行查询
	err := query.Find(&list).Error
	return list, total, err
}

func (d *DBStore) UpdateUser(u *model.User) error {
	// 检查用户是否存在
	var existing model.User
	if err := d.db.First(&existing, "id = ?", u.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ErrNotFound
		}
		return err
	}

	// 检查用户名唯一性（排除自己）
	var count int64
	d.db.Model(&model.User{}).Where("user_name = ? AND id != ?", u.UserName, u.ID).Count(&count)
	if count > 0 {
		return model.ErrUniqueness
	}

	// 全量更新：包含关联表（Emails/Roles）
	return d.db.Session(&gorm.Session{FullSaveAssociations: true}).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Save(u).Error
}

func (d *DBStore) PatchUser(id string, ops []model.PatchOperation) error {
	// 检查用户是否存在
	u, err := d.GetUser(id)
	if err != nil {
		return err
	}

	err = PatchResource(d, id, u, ops)
	if err != nil {
		return err
	}

	// 保存更新
	return d.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(u).Error
}

func (d *DBStore) DeleteUser(id string) error {
	result := d.db.Delete(&model.User{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return model.ErrNotFound
	}
	return nil
}

// ---------------------- Group 实现 ----------------------
func (d *DBStore) CreateGroup(g *model.Group) error {
	// 检查组名唯一性
	var count int64
	d.db.Model(&model.Group{}).Where("display_name = ?", g.DisplayName).Count(&count)
	if count > 0 {
		return model.ErrUniqueness
	}
	return d.db.Create(g).Error
}

func (d *DBStore) GetGroup(id string, preloadMembers bool) (*model.Group, error) {
	var g model.Group
	query := d.db
	if preloadMembers {
		query = query.Preload("Members")
	}
	err := query.First(&g, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	return &g, err
}

func (d *DBStore) ListGroups(q *model.ResourceQuery, preloadMembers bool) ([]model.Group, int64, error) {
	var list []model.Group
	var total int64

	query := d.db
	if preloadMembers {
		query = query.Preload("Members")
	}
	query = query.Model(&model.Group{})

	// 应用过滤器
	if q.Filter != "" {
		filterQuery, args, err := d.applyGroupFilter(q.Filter)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where(filterQuery, args...)
	}

	query.Count(&total)

	offset := (q.StartIndex - 1) * q.Count
	query.Offset(offset).Limit(q.Count)

	if q.SortBy != "" {
		sort := d.toSnakeCase(q.SortBy)
		if q.SortOrder == "descending" {
			sort += " DESC"
		} else {
			sort += " ASC"
		}
		query.Order(sort)
	}

	err := query.Find(&list).Error
	return list, total, err
}

func (d *DBStore) UpdateGroup(g *model.Group) error {
	// 检查组是否存在
	var existing model.Group
	if err := d.db.First(&existing, "id = ?", g.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ErrNotFound
		}
		return err
	}

	// 检查组名唯一性（排除自己）
	var count int64
	d.db.Model(&model.Group{}).Where("display_name = ? AND id != ?", g.DisplayName, g.ID).Count(&count)
	if count > 0 {
		return model.ErrUniqueness
	}

	return d.db.Session(&gorm.Session{FullSaveAssociations: true}).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Save(g).Error
}

func (d *DBStore) PatchGroup(id string, ops []model.PatchOperation) error {
	g, err := d.GetGroup(id, false)
	if err != nil {
		return err
	}
	err = PatchResource(d, id, g, ops)
	if err != nil {
		return err
	}
	return d.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(g).Error
}

func (d *DBStore) DeleteGroup(id string) error {
	result := d.db.Delete(&model.Group{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return model.ErrNotFound
	}
	return nil
}

// ---------------------- Group 成员管理 ----------------------

// AddUserToGroup 添加用户到组
func (d *DBStore) AddUserToGroup(groupID, userID string) error {
	// 验证组是否存在
	var group model.Group
	if err := d.db.First(&group, "id = ?", groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ErrNotFound
		}
		return err
	}

	// 验证用户是否存在
	var user model.User
	if err := d.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ErrNotFound
		}
		return err
	}

	// 检查用户是否已在组中
	var count int64
	d.db.Table("scim_group_members").
		Where("group_id = ? AND value = ?", groupID, userID).
		Count(&count)
	if count > 0 {
		return errors.New("user already in group")
	}

	// 添加成员
	member := model.Member{
		GroupID: groupID,
		Value:   userID,
	}

	// 获取用户名用于显示
	if user.DisplayName != "" {
		member.Display = user.DisplayName
	} else {
		member.Display = user.UserName
	}

	return d.db.Create(&member).Error
}

// RemoveUserFromGroup 从组中移除用户
func (d *DBStore) RemoveUserFromGroup(groupID, userID string) error {
	log.Println("remove user from group", groupID, userID)
	// 验证组是否存在
	var group model.Group
	if err := d.db.First(&group, "id = ?", groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ErrNotFound
		}
		return err
	}

	log.Println("remove scim_group_members", groupID, userID)
	// 删除成员
	result := d.db.Table("scim_group_members").
		Where("group_id = ? AND value = ?", groupID, userID).
		Delete(&model.Member{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not in group")
	}

	return nil
}

// IsUserInGroup 检查用户是否在组中
func (d *DBStore) IsUserInGroup(groupID, userID string) (bool, error) {
	// 验证组是否存在
	var group model.Group
	if err := d.db.First(&group, "id = ?", groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, model.ErrNotFound
		}
		return false, err
	}

	var count int64
	d.db.Table("scim_group_members").
		Where("group_id = ? AND value = ?", groupID, userID).
		Count(&count)

	return count > 0, nil
}

// GetUserGroups 获取用户所属的所有组
func (d *DBStore) GetUserGroups(userID string) ([]model.UserGroup, error) {
	var groups []model.UserGroup
	err := d.db.Raw(`
		SELECT g.id as value, g.display_name as display
		FROM scim_groups g
		JOIN scim_group_members m ON g.id = m.group_id
		WHERE m.value = ?
	`, userID).Scan(&groups).Error

	return groups, err
}

// ---------------------- 过滤器辅助方法 ----------------------

// applyUserFilter 应用过滤器到用户查询
func (d *DBStore) applyUserFilter(filter string) (string, []interface{}, error) {
	node, err := util.ParseFilter(filter)
	if err != nil {
		return "", nil, err
	}

	columnMapping := map[string]string{
		"userName":        "user_name",
		"displayName":     "display_name",
		"nickName":        "nick_name",
		"profileUrl":      "profile_url",
		"active":          "active",
		"externalId":      "external_id",
		"name.givenName":  "given_name",
		"name.familyName": "family_name",
		"name.middleName": "middle_name",
	}

	return util.FilterToSQL(node, columnMapping)
}

// applyGroupFilter 应用过滤器到组查询
func (d *DBStore) applyGroupFilter(filter string) (string, []interface{}, error) {
	node, err := util.ParseFilter(filter)
	if err != nil {
		return "", nil, err
	}

	columnMapping := map[string]string{
		"displayName": "display_name",
		"externalId":  "external_id",
	}

	return util.FilterToSQL(node, columnMapping)
}

// toSnakeCase 转换为蛇形命名
func (d *DBStore) toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
