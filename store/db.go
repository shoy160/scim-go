package store

import (
	"context"
	"errors"
	"fmt"
	"scim-go/model"
	"scim-go/util"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DBStore 关系型数据库存储实现（MySQL/PG通用）
type DBStore struct {
	db    *gorm.DB
	cache util.Cache
}

// NewDB 创建数据库存储实例
func NewDB(db *gorm.DB, cache util.Cache) Store {
	// 自动迁移表结构（含关联表）
	err := db.AutoMigrate(
		&model.User{}, &model.Email{}, &model.Role{},
		&model.Group{}, &model.Member{},
	)
	if err != nil {
		panic("auto migrate table failed: " + err.Error())
	}
	return &DBStore{db: db, cache: cache}
}

// NewDBWithMemoryCache 创建带内存缓存的数据库存储实例
func NewDBWithMemoryCache(db *gorm.DB) Store {
	return NewDB(db, util.NewMemoryCache())
}

// NewDBWithRedisCache 创建带Redis缓存的数据库存储实例
func NewDBWithRedisCache(db *gorm.DB, redisAddr, redisPassword string, redisDB int) Store {
	return NewDB(db, util.NewRedisCache(redisAddr, redisPassword, redisDB))
}

// ---------------------- User 实现 ----------------------
// CreateUser 创建用户
func (d *DBStore) CreateUser(u *model.User) error {
	// 检查用户名唯一性
	var count int64
	d.db.Model(&model.User{}).Where("user_name = ?", u.UserName).Count(&count)
	if count > 0 {
		return model.ErrUniqueness
	}
	// 设置 SCIM Meta 字段（ResourceType 不持久化，由API层动态生成）
	u.Version = generateVersion()
	// 设置默认 schemas（如果未提供）
	if len(u.Schemas) == 0 {
		u.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:User"}
	}
	// 去重处理：检查 emails 和 roles 是否有重复
	u.Emails = deduplicateEmails(u.Emails)
	u.Roles = deduplicateRoles(u.Roles)
	// 创建用户
	if err := d.db.Create(u).Error; err != nil {
		return err
	}
	// 清除相关缓存
	if d.cache != nil {
		ctx := context.Background()
		d.cache.Delete(ctx, "users:list")
		d.cache.Delete(ctx, "user:"+u.ID)
	}
	return nil
}

// GetUser 获取用户
func (d *DBStore) GetUser(id string) (*model.User, error) {
	// 尝试从缓存获取
	if d.cache != nil {
		ctx := context.Background()
		var u model.User
		err := d.cache.Get(ctx, "user:"+id, &u)
		if err == nil {
			return &u, nil
		}
	}
	// 从数据库获取
	var u model.User
	err := d.db.Preload("Emails").Preload("Roles").First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	// 存入缓存
	if d.cache != nil {
		ctx := context.Background()
		d.cache.Set(ctx, "user:"+id, u, 5*time.Minute)
	}
	return &u, nil
}

// ListUsers 列出用户
func (d *DBStore) ListUsers(q *model.ResourceQuery) ([]model.User, int64, error) {
	// 构建缓存键
	cacheKey := fmt.Sprintf("users:list:%s:%d:%d:%s:%s",
		q.Filter, q.StartIndex, q.Count, q.SortBy, q.SortOrder)

	// 尝试从缓存获取
	if d.cache != nil {
		ctx := context.Background()
		var result struct {
			Users []model.User
			Total int64
		}
		err := d.cache.Get(ctx, cacheKey, &result)
		if err == nil {
			return result.Users, result.Total, nil
		}
	}

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
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	offset := (q.StartIndex - 1) * q.Count
	query = query.Offset(offset).Limit(q.Count)

	// 排序
	if q.SortBy != "" {
		sort := d.toSnakeCase(q.SortBy)
		if q.SortOrder == "descending" {
			sort += " DESC"
		} else {
			sort += " ASC"
		}
		query = query.Order(sort)
	}

	// 执行查询
	var list []model.User
	if err := query.Find(&list).Error; err != nil {
		return nil, 0, err
	}

	// 存入缓存
	if d.cache != nil {
		ctx := context.Background()
		result := struct {
			Users []model.User
			Total int64
		}{
			Users: list,
			Total: total,
		}
		d.cache.Set(ctx, cacheKey, result, 2*time.Minute)
	}

	return list, total, nil
}

// UpdateUser 更新用户
func (d *DBStore) UpdateUser(u *model.User) error {
	// 使用事务确保原子性
	return d.db.Transaction(func(tx *gorm.DB) error {
		// 检查用户是否存在
		var existing model.User
		if err := tx.First(&existing, "id = ?", u.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrNotFound
			}
			return err
		}

		// 检查用户名唯一性（排除自己）
		var count int64
		tx.Model(&model.User{}).Where("user_name = ? AND id != ?", u.UserName, u.ID).Count(&count)
		if count > 0 {
			return model.ErrUniqueness
		}

		// 去重处理：检查 emails 和 roles 是否有重复
		u.Emails = deduplicateEmails(u.Emails)
		u.Roles = deduplicateRoles(u.Roles)

		// 验证 emails 是否与其他用户的记录重复
		if len(u.Emails) > 0 {
			for _, email := range u.Emails {
				var existingEmail model.Email
				if err := tx.Where("value = ? AND user_id != ?", email.Value, u.ID).First(&existingEmail).Error; err == nil {
					return fmt.Errorf("email already exists: %s", email.Value)
				}
			}
		}

		// 验证 roles 是否与其他用户的记录重复
		if len(u.Roles) > 0 {
			for _, role := range u.Roles {
				var existingRole model.Role
				if err := tx.Where("value = ? AND user_id != ?", role.Value, u.ID).First(&existingRole).Error; err == nil {
					return fmt.Errorf("role already exists: %s", role.Value)
				}
			}
		}

		// 获取当前用户的 emails 和 roles
		var currentEmails []model.Email
		var currentRoles []model.Role
		if err := tx.Where("user_id = ?", u.ID).Find(&currentEmails).Error; err != nil {
			return fmt.Errorf("failed to query current emails: %w", err)
		}
		if err := tx.Where("user_id = ?", u.ID).Find(&currentRoles).Error; err != nil {
			return fmt.Errorf("failed to query current roles: %w", err)
		}

		// 构建当前 emails 和 roles 的映射，用于快速查找
		currentEmailMap := make(map[string]model.Email)
		for _, email := range currentEmails {
			currentEmailMap[strings.ToLower(email.Value)] = email
		}
		currentRoleMap := make(map[string]model.Role)
		for _, role := range currentRoles {
			currentRoleMap[strings.ToLower(role.Value)] = role
		}

		// 构建目标 emails 和 roles 的映射
		targetEmailMap := make(map[string]model.Email)
		for _, email := range u.Emails {
			targetEmailMap[strings.ToLower(email.Value)] = email
		}
		targetRoleMap := make(map[string]model.Role)
		for _, role := range u.Roles {
			targetRoleMap[strings.ToLower(role.Value)] = role
		}

		// 处理 emails 的增量更新
		// 1. 删除不再存在的 emails
		for emailValue, email := range currentEmailMap {
			if _, exists := targetEmailMap[emailValue]; !exists {
				if err := tx.Delete(&email).Error; err != nil {
					return fmt.Errorf("failed to delete email: %w", err)
				}
			}
		}
		// 2. 添加新的 emails
		for emailValue, email := range targetEmailMap {
			if _, exists := currentEmailMap[emailValue]; !exists {
				email.UserID = u.ID
				if err := tx.Create(&email).Error; err != nil {
					return fmt.Errorf("failed to create email: %w", err)
				}
			}
		}

		// 处理 roles 的增量更新
		// 1. 删除不再存在的 roles
		for roleValue, role := range currentRoleMap {
			if _, exists := targetRoleMap[roleValue]; !exists {
				if err := tx.Delete(&role).Error; err != nil {
					return fmt.Errorf("failed to delete role: %w", err)
				}
			}
		}
		// 2. 添加新的 roles
		for roleValue, role := range targetRoleMap {
			if _, exists := currentRoleMap[roleValue]; !exists {
				role.UserID = u.ID
				if err := tx.Create(&role).Error; err != nil {
					return fmt.Errorf("failed to create role: %w", err)
				}
			}
		}

		// 更新版本号（每次更新必须变化）
		u.Version = generateVersion()
		// 更新 schemas（支持自定义 schemas）
		if len(u.Schemas) == 0 {
			u.Schemas = existing.Schemas
		}
		// 保留原有的创建时间，如果原记录有创建时间则保留，否则使用当前时间
		if !existing.CreatedAt.IsZero() {
			u.CreatedAt = existing.CreatedAt
		} else {
			u.CreatedAt = time.Now()
		}
		// UpdatedAt 由 GORM 的 autoUpdateTime 自动更新

		// 更新用户基本信息
		if err := tx.Model(&model.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
			"user_name":    u.UserName,
			"formatted":    u.Name.Formatted,
			"given_name":   u.Name.GivenName,
			"family_name":  u.Name.FamilyName,
			"middle_name":  u.Name.MiddleName,
			"active":       u.Active,
			"display_name": u.DisplayName,
			"nick_name":    u.NickName,
			"profile_url":  u.ProfileUrl,
			"version":      u.Version,
			"schemas":      u.Schemas,
		}).Error; err != nil {
			return err
		}

		// 清除相关缓存
		if d.cache != nil {
			ctx := context.Background()
			d.cache.Delete(ctx, "users:list")
			d.cache.Delete(ctx, "user:"+u.ID)
		}

		return nil
	})
}

// validateEmailsUniqueness 验证 emails 是否与数据库中已存在的记录重复
func validateEmailsUniqueness(tx *gorm.DB, userID string, emails []model.Email) ([]model.Email, error) {
	if len(emails) == 0 {
		return nil, nil
	}

	// 获取数据库中该用户的所有现有 emails
	var existingEmails []model.Email
	if err := tx.Where("user_id = ?", userID).Find(&existingEmails).Error; err != nil {
		return nil, fmt.Errorf("failed to query existing emails: %w", err)
	}

	// 构建现有 email 的集合（不区分大小写）
	existingEmailSet := make(map[string]bool)
	for _, email := range existingEmails {
		existingEmailSet[strings.ToLower(email.Value)] = true
	}

	// 检查新 emails 是否与现有 emails 重复
	var newEmails []model.Email
	for _, email := range emails {
		emailLower := strings.ToLower(email.Value)
		if !existingEmailSet[emailLower] {
			newEmails = append(newEmails, email)
		}
	}
	if len(newEmails) == 0 {
		return nil, nil
	}
	return newEmails, nil
}

// validateRolesUniqueness 验证 roles 是否与数据库中已存在的记录重复
func validateRolesUniqueness(tx *gorm.DB, userID string, roles []model.Role) ([]model.Role, error) {
	if len(roles) == 0 {
		return nil, nil
	}

	// 获取数据库中该用户的所有现有 roles
	var existingRoles []model.Role
	if err := tx.Where("user_id = ?", userID).Find(&existingRoles).Error; err != nil {
		return nil, fmt.Errorf("failed to query existing roles: %w", err)
	}

	// 构建现有 role 的集合（不区分大小写）
	existingRoleSet := make(map[string]bool)
	for _, role := range existingRoles {
		existingRoleSet[strings.ToLower(role.Value)] = true
	}

	// 检查新 roles 是否与现有 roles 重复
	var newRoles []model.Role
	for _, role := range roles {
		roleLower := strings.ToLower(role.Value)
		if !existingRoleSet[roleLower] {
			newRoles = append(newRoles, role)
		}
	}

	// 如果有重复，返回错误
	if len(newRoles) > 0 {
		return newRoles, nil
	}

	return nil, nil
}

// deduplicateEmails 对邮箱列表进行去重，保留第一个出现的记录
func deduplicateEmails(emails []model.Email) []model.Email {
	if len(emails) <= 1 {
		return emails
	}
	seen := make(map[string]bool)
	result := make([]model.Email, 0, len(emails))
	for _, email := range emails {
		key := strings.ToLower(email.Value)
		if !seen[key] {
			seen[key] = true
			result = append(result, email)
		}
	}
	return result
}

// deduplicateRoles 对角色列表进行去重，保留第一个出现的记录
func deduplicateRoles(roles []model.Role) []model.Role {
	if len(roles) <= 1 {
		return roles
	}
	seen := make(map[string]bool)
	result := make([]model.Role, 0, len(roles))
	for _, role := range roles {
		key := strings.ToLower(role.Value)
		if !seen[key] {
			seen[key] = true
			result = append(result, role)
		}
	}
	return result
}

func (d *DBStore) PatchUser(id string, ops []model.PatchOperation) error {
	// 在事务内执行 Patch 操作
	return d.db.Transaction(func(tx *gorm.DB) error {
		// 获取事务内的用户数据（包含关联的 emails 和 roles）
		var u model.User
		if err := tx.First(&u, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrNotFound
			}
			return err
		}

		// 预加载现有的 emails 和 roles
		if err := tx.Preload("Emails").Preload("Roles").First(&u, "id = ?", id).Error; err != nil {
			return err
		}

		// 保存原始的 emails 和 roles 用于后续对比
		originalEmails := make([]model.Email, len(u.Emails))
		copy(originalEmails, u.Emails)
		originalRoles := make([]model.Role, len(u.Roles))
		copy(originalRoles, u.Roles)

		// 应用补丁操作（处理通用字段和关联字段）
		err := PatchResource(d, id, &u, ops)
		if err != nil {
			return err
		}

		// 处理 emails 的差异更新
		if err := d.syncEmails(tx, id, originalEmails, u.Emails); err != nil {
			return err
		}

		// 处理 roles 的差异更新
		if err := d.syncRoles(tx, id, originalRoles, u.Roles); err != nil {
			return err
		}

		// 更新版本号（每次更新必须变化）
		u.Version = generateVersion()
		// UpdatedAt 由 GORM 的 autoUpdateTime 自动更新

		// 保存更新
		return tx.Session(&gorm.Session{FullSaveAssociations: true}).Save(&u).Error
	})
}

// syncEmails 同步 emails，仅对差异部分执行删除或新增操作
func (d *DBStore) syncEmails(tx *gorm.DB, userID string, originalEmails, newEmails []model.Email) error {
	// 构建原始 email 的 map，用于快速查找
	originalMap := make(map[string]model.Email)
	for _, email := range originalEmails {
		originalMap[strings.ToLower(email.Value)] = email
	}

	// 构建新 email 的 map
	newMap := make(map[string]model.Email)
	for _, email := range newEmails {
		newMap[strings.ToLower(email.Value)] = email
	}

	// 找出需要删除的 emails（在原始中存在但在新中不存在）
	for key, originalEmail := range originalMap {
		if _, exists := newMap[key]; !exists {
			// 删除该 email
			if err := tx.Where("user_id = ? AND LOWER(value) = LOWER(?)", userID, originalEmail.Value).Delete(&model.Email{}).Error; err != nil {
				return err
			}
		}
	}

	// 找出需要新增的 emails（在新中存在但在原始中不存在）
	for key, newEmail := range newMap {
		if _, exists := originalMap[key]; !exists {
			// 新增该 email
			newEmail.UserID = userID
			if err := tx.Create(&newEmail).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// syncRoles 同步 roles，仅对差异部分执行删除或新增操作
func (d *DBStore) syncRoles(tx *gorm.DB, userID string, originalRoles, newRoles []model.Role) error {
	// 构建原始 role 的 map，用于快速查找
	originalMap := make(map[string]model.Role)
	for _, role := range originalRoles {
		originalMap[strings.ToLower(role.Value)] = role
	}

	// 构建新 role 的 map
	newMap := make(map[string]model.Role)
	for _, role := range newRoles {
		newMap[strings.ToLower(role.Value)] = role
	}

	// 找出需要删除的 roles（在原始中存在但在新中不存在）
	for key, originalRole := range originalMap {
		if _, exists := newMap[key]; !exists {
			// 删除该 role
			if err := tx.Where("user_id = ? AND LOWER(value) = LOWER(?)", userID, originalRole.Value).Delete(&model.Role{}).Error; err != nil {
				return err
			}
		}
	}

	// 找出需要新增的 roles（在新中存在但在原始中不存在）
	for key, newRole := range newMap {
		if _, exists := originalMap[key]; !exists {
			// 新增该 role
			newRole.UserID = userID
			if err := tx.Create(&newRole).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// validateEmailsUniquenessByValues 验证 emails 是否重复（仅检查新值）
func validateEmailsUniquenessByValues(emails []model.Email) ([]model.Email, error) {
	if len(emails) == 0 {
		return nil, nil
	}

	existingEmailSet := make(map[string]bool)
	var newEmails []model.Email
	for _, email := range emails {
		emailLower := strings.ToLower(email.Value)
		if !existingEmailSet[emailLower] {
			existingEmailSet[emailLower] = true
			newEmails = append(newEmails, email)
		}
	}
	return newEmails, nil
}

// validateRolesUniquenessByValues 验证 roles 是否重复（仅检查新值）
func validateRolesUniquenessByValues(roles []model.Role) ([]model.Role, error) {
	if len(roles) == 0 {
		return nil, nil
	}

	existingRoleSet := make(map[string]bool)
	var newRoles []model.Role
	for _, role := range roles {
		roleLower := strings.ToLower(role.Value)
		if !existingRoleSet[roleLower] {
			existingRoleSet[roleLower] = true
			newRoles = append(newRoles, role)
		}
	}
	return newRoles, nil
}

// DeleteUser 删除用户
func (d *DBStore) DeleteUser(id string) error {
	result := d.db.Delete(&model.User{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return model.ErrNotFound
	}
	// 清除相关缓存
	if d.cache != nil {
		ctx := context.Background()
		d.cache.Delete(ctx, "users:list")
		d.cache.Delete(ctx, "user:"+id)
	}
	return nil
}

// RemoveEmailFromUser 从用户中移除指定邮箱
func (d *DBStore) RemoveEmailFromUser(userID, emailValue string) error {
	result := d.db.Where("user_id = ? AND LOWER(value) = LOWER(?)", userID, emailValue).Delete(&model.Email{})
	if result.Error != nil {
		return result.Error
	}
	// 如果找不到记录，返回 nil 而不是错误（可能已经被删除）
	return nil
}

// RemoveRoleFromUser 从用户中移除指定角色
func (d *DBStore) RemoveRoleFromUser(userID, roleValue string) error {
	result := d.db.Where("user_id = ? AND LOWER(value) = LOWER(?)", userID, roleValue).Delete(&model.Role{})
	if result.Error != nil {
		return result.Error
	}
	// 如果找不到记录，返回 nil 而不是错误（可能已经被删除）
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
	// 设置 SCIM Meta 字段（ResourceType 不持久化，由API层动态生成）
	g.Version = generateVersion()
	// 设置默认 schemas（如果未提供）
	if len(g.Schemas) == 0 {
		g.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:Group"}
	}
	return d.db.Create(g).Error
}

// GetGroup 获取组
func (d *DBStore) GetGroup(id string, preloadMembers bool) (*model.Group, error) {
	// 构建缓存键
	cacheKey := fmt.Sprintf("group:%s:%v", id, preloadMembers)

	// 尝试从缓存获取
	if d.cache != nil {
		ctx := context.Background()
		var g model.Group
		err := d.cache.Get(ctx, cacheKey, &g)
		if err == nil {
			return &g, nil
		}
	}

	// 从数据库获取
	var g model.Group
	query := d.db
	if preloadMembers {
		query = query.Preload("Members")
	}
	err := query.First(&g, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// 存入缓存
	if d.cache != nil {
		ctx := context.Background()
		d.cache.Set(ctx, cacheKey, g, 5*time.Minute)
	}

	return &g, nil
}

// ListGroups 列出组
func (d *DBStore) ListGroups(q *model.ResourceQuery, preloadMembers bool) ([]model.Group, int64, error) {
	// 构建缓存键
	cacheKey := fmt.Sprintf("groups:list:%s:%d:%d:%s:%s:%v",
		q.Filter, q.StartIndex, q.Count, q.SortBy, q.SortOrder, preloadMembers)

	// 尝试从缓存获取
	if d.cache != nil {
		ctx := context.Background()
		var result struct {
			Groups []model.Group
			Total  int64
		}
		err := d.cache.Get(ctx, cacheKey, &result)
		if err == nil {
			return result.Groups, result.Total, nil
		}
	}

	// 从数据库获取
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

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	offset := (q.StartIndex - 1) * q.Count
	query = query.Offset(offset).Limit(q.Count)

	// 排序
	if q.SortBy != "" {
		sort := d.toSnakeCase(q.SortBy)
		if q.SortOrder == "descending" {
			sort += " DESC"
		} else {
			sort += " ASC"
		}
		query = query.Order(sort)
	}

	// 执行查询
	if err := query.Find(&list).Error; err != nil {
		return nil, 0, err
	}

	// 存入缓存
	if d.cache != nil {
		ctx := context.Background()
		result := struct {
			Groups []model.Group
			Total  int64
		}{
			Groups: list,
			Total:  total,
		}
		d.cache.Set(ctx, cacheKey, result, 2*time.Minute)
	}

	return list, total, nil
}

// UpdateGroup 更新组
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

	// 更新版本号（每次更新必须变化）
	g.Version = generateVersion()
	// 更新 schemas（支持自定义 schemas）
	if len(g.Schemas) == 0 {
		g.Schemas = existing.Schemas
	}
	// 保留原有的创建时间，如果原记录有创建时间则保留，否则使用当前时间
	if !existing.CreatedAt.IsZero() {
		g.CreatedAt = existing.CreatedAt
	} else {
		g.CreatedAt = time.Now()
	}
	// UpdatedAt 由 GORM 的 autoUpdateTime 自动更新

	// 保存更新
	if err := d.db.Session(&gorm.Session{FullSaveAssociations: true}).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Save(g).Error; err != nil {
		return err
	}

	// 清除相关缓存
	if d.cache != nil {
		ctx := context.Background()
		d.cache.Delete(ctx, "groups:list")
		d.cache.Delete(ctx, fmt.Sprintf("group:%s:true", g.ID))
		d.cache.Delete(ctx, fmt.Sprintf("group:%s:false", g.ID))
	}

	return nil
}

// PatchGroup 补丁更新组
func (d *DBStore) PatchGroup(id string, ops []model.PatchOperation) error {
	g, err := d.GetGroup(id, false)
	if err != nil {
		return err
	}
	err = PatchResource(d, id, g, ops)
	if err != nil {
		return err
	}

	// 更新版本号（每次更新必须变化）
	g.Version = generateVersion()
	// UpdatedAt 由 GORM 的 autoUpdateTime 自动更新

	// 保存更新
	if err := d.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(g).Error; err != nil {
		return err
	}

	// 清除相关缓存
	if d.cache != nil {
		ctx := context.Background()
		d.cache.Delete(ctx, "groups:list")
		d.cache.Delete(ctx, fmt.Sprintf("group:%s:true", id))
		d.cache.Delete(ctx, fmt.Sprintf("group:%s:false", id))
	}

	return nil
}

// DeleteGroup 删除组
func (d *DBStore) DeleteGroup(id string) error {
	result := d.db.Delete(&model.Group{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return model.ErrNotFound
	}
	// 清除相关缓存
	if d.cache != nil {
		ctx := context.Background()
		d.cache.Delete(ctx, "groups:list")
		d.cache.Delete(ctx, fmt.Sprintf("group:%s:true", id))
		d.cache.Delete(ctx, fmt.Sprintf("group:%s:false", id))
	}
	return nil
}

// ---------------------- Group 成员管理 ----------------------

// AddMemberToGroup 添加成员到组（支持用户和组）
func (d *DBStore) AddMemberToGroup(groupID, memberID string, memberType ...model.MemberType) error {
	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	// 验证组是否存在
	var group model.Group
	if err := d.db.First(&group, "id = ?", groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ErrNotFound
		}
		return err
	}

	// 验证成员是否存在并获取显示名称
	member := model.Member{
		GroupID: groupID,
		Value:   memberID,
		Type:    mt,
	}

	if mt == "User" {
		var user model.User
		if err := d.db.First(&user, "id = ?", memberID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		if user.DisplayName != "" {
			member.Display = user.DisplayName
		} else {
			member.Display = user.UserName
		}
	} else if mt == "Group" {
		var subGroup model.Group
		if err := d.db.First(&subGroup, "id = ?", memberID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrNotFound
			}
			return err
		}
		member.Display = subGroup.DisplayName
	} else {
		return model.ErrInvalidValue
	}

	// 检查成员是否已在组中
	var count int64
	d.db.Table("scim_group_members").
		Where("group_id = ? AND value = ?", groupID, memberID).
		Count(&count)
	if count > 0 {
		return model.ErrMemberAlreadyInGroup
	}

	// 添加成员
	if err := d.db.Create(&member).Error; err != nil {
		return err
	}

	// 清除相关缓存
	if d.cache != nil {
		ctx := context.Background()
		d.cache.Delete(ctx, "groups:list")
		d.cache.Delete(ctx, fmt.Sprintf("group:%s:true", groupID))
		d.cache.Delete(ctx, fmt.Sprintf("group:%s:false", groupID))
		// 如果是用户，还需要清除用户相关的缓存
		if mt == "User" {
			d.cache.Delete(ctx, "user:"+memberID)
		}
	}

	return nil
}

// RemoveMemberFromGroup 从组中移除成员（支持用户和组）
func (d *DBStore) RemoveMemberFromGroup(groupID, memberID string, memberType ...model.MemberType) error {
	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	// 验证组是否存在
	var group model.Group
	if err := d.db.First(&group, "id = ?", groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ErrNotFound
		}
		return err
	}

	// 删除成员
	query := d.db.Table("scim_group_members").
		Where("group_id = ? AND value = ?", groupID, memberID)
	if mt != "" {
		query = query.Where("type = ?", mt)
	}
	result := query.Delete(&model.Member{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return model.ErrMemberNotInGroup
	}

	// 清除相关缓存
	if d.cache != nil {
		ctx := context.Background()
		d.cache.Delete(ctx, "groups:list")
		d.cache.Delete(ctx, fmt.Sprintf("group:%s:true", groupID))
		d.cache.Delete(ctx, fmt.Sprintf("group:%s:false", groupID))
		// 清除用户相关的缓存（假设成员是用户）
		if mt == "User" {
			d.cache.Delete(ctx, "user:"+memberID)
		}
	}

	return nil
}

// IsMemberInGroup 检查成员是否在组中（支持用户和组）
func (d *DBStore) IsMemberInGroup(groupID, memberID string, memberType ...model.MemberType) (bool, error) {
	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	// 验证组是否存在
	var group model.Group
	if err := d.db.First(&group, "id = ?", groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, model.ErrNotFound
		}
		return false, err
	}

	var count int64
	query := d.db.Table("scim_group_members").
		Where("group_id = ? AND value = ?", groupID, memberID)
	if mt != "" {
		query = query.Where("type = ?", mt)
	}
	query.Count(&count)

	return count > 0, nil
}

// GetMemberGroups 获取成员所属的所有组（支持用户和组）
func (d *DBStore) GetMemberGroups(memberID string, memberType ...model.MemberType) ([]model.UserGroup, error) {
	// 处理默认参数
	mt := model.MemberTypeUser
	if len(memberType) > 0 && memberType[0] != "" {
		mt = memberType[0]
	}

	var groups []model.UserGroup
	var err error
	if mt != "" {
		err = d.db.Raw(`
			SELECT g.id as value, g.display_name as display
			FROM scim_groups g
			JOIN scim_group_members m ON g.id = m.group_id
			WHERE m.value = ? AND m.type = ?
		`, memberID, mt).Scan(&groups).Error
	} else {
		err = d.db.Raw(`
			SELECT g.id as value, g.display_name as display
			FROM scim_groups g
			JOIN scim_group_members m ON g.id = m.group_id
			WHERE m.value = ?
		`, memberID).Scan(&groups).Error
	}

	return groups, err
}

// GetGroupMembers 获取组成员（支持分页和类型过滤）
func (d *DBStore) GetGroupMembers(groupID string, memberType model.MemberType, q *model.ResourceQuery) ([]model.Member, int64, error) {
	var group model.Group
	if err := d.db.First(&group, "id = ?", groupID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, model.ErrNotFound
		}
		return nil, 0, err
	}

	var members []model.Member
	query := d.db.Where("group_id = ?", groupID)
	if memberType != "" {
		query = query.Where("type = ?", memberType)
	}

	var total int64
	query.Model(&model.Member{}).Count(&total)

	startIndex := q.StartIndex
	count := q.Count

	if startIndex < 1 {
		startIndex = 1
	}
	if count <= 0 {
		count = 100
	}

	offset := startIndex - 1
	if err := query.Offset(offset).Limit(count).Find(&members).Error; err != nil {
		return nil, 0, err
	}

	return members, total, nil
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
