package api

import (
	"errors"
	"net/http"
	"scim-go/model"
	"scim-go/store"
	"scim-go/util"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserHandlers User相关接口处理器
type UserHandlers struct {
	store store.Store
	cfg   *model.ScimConfig
}

// GroupHandlers Group相关接口处理器
type GroupHandlers struct {
	store store.Store
	cfg   *model.ScimConfig
}

// NewUserHandlers 创建User处理器
func NewUserHandlers(s store.Store, cfg *model.ScimConfig) *UserHandlers {
	return &UserHandlers{store: s, cfg: cfg}
}

// NewGroupHandlers 创建Group处理器
func NewGroupHandlers(s store.Store, cfg *model.ScimConfig) *GroupHandlers {
	return &GroupHandlers{store: s, cfg: cfg}
}

// ---------------------- User 接口实现 ----------------------

// ListUsers 列出所有用户
// @Summary 列出所有用户
// @Description 获取用户列表，支持分页、过滤和属性选择
// @Tags Users
// @Accept json
// @Produce json
// @Param startIndex query int false "起始索引" default(1)
// @Param count query int false "每页数量" default(20)
// @Param filter query string false "过滤条件"
// @Param attributes query string false "要返回的属性列表，逗号分隔"
// @Param excludedAttributes query string false "要排除的属性列表，逗号分隔"
// @Success 200 {object} model.ListResponse "用户列表"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Users [get]
// @Security BearerAuth
func (h *UserHandlers) ListUsers(c *gin.Context) {
	q := c.MustGet("scim_query").(*model.ResourceQuery)

	// 验证过滤器语法
	if err := h.validateFilter(c, q.Filter); err != nil {
		return
	}

	users, total, err := h.store.ListUsers(q)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 检查是否需要加载用户的组信息
	needGroups := h.needsGroups(q)

	// 处理用户列表并应用属性选择
	resources := h.processUserList(users, q, needGroups)
	if resources == nil {
		ErrorHandler(c, errors.New("failed to process user list"), http.StatusInternalServerError, "internalError")
		return
	}

	// 构造SCIM标准列表响应
	c.JSON(http.StatusOK, model.ListResponse{
		Schemas:      []string{h.cfg.ListSchema},
		TotalResults: int(total),
		StartIndex:   q.StartIndex,
		ItemsPerPage: q.Count,
		Resources:    resources,
	})
}

// GetUser 获取单个用户
// @Summary 获取单个用户
// @Description 根据用户ID获取用户详细信息
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param attributes query string false "要返回的属性列表，逗号分隔"
// @Param excludedAttributes query string false "要排除的属性列表，逗号分隔"
// @Success 200 {object} model.User "用户信息"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Users/{id} [get]
// @Security BearerAuth
func (h *UserHandlers) GetUser(c *gin.Context) {
	id := c.Param("id")
	q := c.MustGet("scim_query").(*model.ResourceQuery)

	user, err := h.store.GetUser(id)
	if err != nil {
		ErrorHandler(c, err, http.StatusNotFound, "notFound")
		return
	}

	// 设置Schema并加载组信息
	h.enrichUser(user, q)

	// 应用属性选择
	if q.Attributes != "" || q.ExcludedAttributes != "" {
		filtered, err := util.ApplyAttributeSelection(user, q.Attributes, q.ExcludedAttributes)
		if err != nil {
			ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
			return
		}
		c.JSON(http.StatusOK, filtered)
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户，需要提供用户名和姓名信息
// @Tags Users
// @Accept json
// @Produce json
// @Param user body model.User true "用户信息"
// @Success 201 {object} model.User "创建成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 409 {object} model.ErrorResponse "用户已存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Users [post]
// @Security BearerAuth
func (h *UserHandlers) CreateUser(c *gin.Context) {
	var u model.User
	if err := c.ShouldBindJSON(&u); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 验证必填字段
	if err := h.validateUserRequiredFields(&u); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 生成唯一ID并设置Schema
	u.ID = uuid.NewString()
	u.Schemas = []string{h.cfg.DefaultSchema}

	// 保存用户
	if err := h.store.CreateUser(&u); err != nil {
		if errors.Is(err, model.ErrUniqueness) {
			ErrorHandler(c, err, http.StatusConflict, "uniqueness")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	c.JSON(http.StatusCreated, u)
}

// UpdateUser 全量替换用户
// @Summary 全量更新用户
// @Description 全量替换用户信息
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param user body model.User true "用户信息"
// @Success 200 {object} model.User "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Users/{id} [put]
// @Security BearerAuth
func (h *UserHandlers) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var u model.User
	if err := c.ShouldBindJSON(&u); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 验证必填字段
	if u.UserName == "" {
		ErrorHandler(c, errors.New("userName is required"), http.StatusBadRequest, "invalidValue")
		return
	}

	// 强制使用URL中的ID
	u.ID = id
	u.Schemas = []string{h.cfg.DefaultSchema}

	if err := h.store.UpdateUser(&u); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	c.JSON(http.StatusOK, u)
}

// PatchUser 补丁更新用户属性
// @Summary 补丁更新用户
// @Description 使用补丁操作更新用户属性
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param patch body model.PatchRequest true "补丁操作"
// @Success 200 {object} model.User "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Users/{id} [patch]
// @Security BearerAuth
func (h *UserHandlers) PatchUser(c *gin.Context) {
	id := c.Param("id")
	var req model.PatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 验证Patch请求
	if err := h.validatePatchRequest(&req); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 执行补丁更新
	if err := h.store.PatchUser(id, req.Operations); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 返回更新后的用户
	user, err := h.store.GetUser(id)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}
	user.Schemas = []string{h.cfg.DefaultSchema}
	c.JSON(http.StatusOK, user)
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 根据用户ID删除用户
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 204 "删除成功"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "用户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Users/{id} [delete]
// @Security BearerAuth
func (h *UserHandlers) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.DeleteUser(id); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// SCIM标准：删除成功返回204 No Content
	c.Status(http.StatusNoContent)
}

// ---------------------- Group 接口实现 ----------------------

// ListGroups 列出所有组
// @Summary 列出所有组
// @Description 获取组列表，支持分页、过滤和属性选择
// @Tags Groups
// @Accept json
// @Produce json
// @Param startIndex query int false "起始索引" default(1)
// @Param count query int false "每页数量" default(20)
// @Param filter query string false "过滤条件"
// @Param attributes query string false "要返回的属性列表，逗号分隔"
// @Param excludedAttributes query string false "要排除的属性列表，逗号分隔"
// @Success 200 {object} model.ListResponse "组列表"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups [get]
// @Security BearerAuth
func (h *GroupHandlers) ListGroups(c *gin.Context) {
	q := c.MustGet("scim_query").(*model.ResourceQuery)

	// 验证过滤器语法
	if err := h.validateFilter(c, q.Filter); err != nil {
		return
	}

	// 检查是否需要预加载成员
	preloadMembers := strings.Contains(q.Attributes, "members") || strings.Contains(q.ExcludedAttributes, "members")

	groups, total, err := h.store.ListGroups(q, preloadMembers)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 处理组列表并应用属性选择
	resources := h.processGroupList(groups, q)
	if resources == nil {
		ErrorHandler(c, errors.New("failed to process group list"), http.StatusInternalServerError, "internalError")
		return
	}

	// 构造SCIM标准列表响应
	c.JSON(http.StatusOK, model.ListResponse{
		Schemas:      []string{h.cfg.ListSchema},
		TotalResults: int(total),
		StartIndex:   q.StartIndex,
		ItemsPerPage: q.Count,
		Resources:    resources,
	})
}

// GetGroup 获取单个组
// @Summary 获取单个组
// @Description 根据组ID获取组详细信息
// @Tags Groups
// @Accept json
// @Produce json
// @Param id path string true "组ID"
// @Param attributes query string false "要返回的属性列表，逗号分隔"
// @Param excludedAttributes query string false "要排除的属性列表，逗号分隔"
// @Success 200 {object} model.Group "组信息"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "组不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups/{id} [get]
// @Security BearerAuth
func (h *GroupHandlers) GetGroup(c *gin.Context) {
	id := c.Param("id")
	q := c.MustGet("scim_query").(*model.ResourceQuery)

	// 检查是否需要预加载成员
	preloadMembers := strings.Contains(q.Attributes, "members") || strings.Contains(q.ExcludedAttributes, "members")

	group, err := h.store.GetGroup(id, preloadMembers)
	if err != nil {
		ErrorHandler(c, err, http.StatusNotFound, "notFound")
		return
	}

	// 设置Schema
	group.Schemas = []string{h.cfg.GroupSchema}

	// 应用属性选择
	if q.Attributes != "" || q.ExcludedAttributes != "" {
		filtered, err := util.ApplyAttributeSelection(group, q.Attributes, q.ExcludedAttributes)
		if err != nil {
			ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
			return
		}
		c.JSON(http.StatusOK, filtered)
		return
	}

	c.JSON(http.StatusOK, group)
}

// CreateGroup 创建组
// @Summary 创建组
// @Description 创建新组，需要提供组显示名称
// @Tags Groups
// @Accept json
// @Produce json
// @Param group body model.Group true "组信息"
// @Success 201 {object} model.Group "创建成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 409 {object} model.ErrorResponse "组已存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups [post]
// @Security BearerAuth
func (h *GroupHandlers) CreateGroup(c *gin.Context) {
	var g model.Group
	if err := c.ShouldBindJSON(&g); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 验证必填字段
	if g.DisplayName == "" {
		ErrorHandler(c, errors.New("displayName is required"), http.StatusBadRequest, "invalidValue")
		return
	}

	// 生成唯一ID并设置Schema
	g.ID = uuid.NewString()
	g.Schemas = []string{h.cfg.GroupSchema}

	// 保存组
	if err := h.store.CreateGroup(&g); err != nil {
		if errors.Is(err, model.ErrUniqueness) {
			ErrorHandler(c, err, http.StatusConflict, "uniqueness")
			return
		}
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, errors.New("invalid member reference"), http.StatusBadRequest, "invalidValue")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	c.JSON(http.StatusCreated, g)
}

// UpdateGroup 全量替换组
// @Summary 全量更新组
// @Description 全量替换组信息
// @Tags Groups
// @Accept json
// @Produce json
// @Param id path string true "组ID"
// @Param group body model.Group true "组信息"
// @Success 200 {object} model.Group "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "组不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups/{id} [put]
// @Security BearerAuth
func (h *GroupHandlers) UpdateGroup(c *gin.Context) {
	id := c.Param("id")
	var g model.Group
	if err := c.ShouldBindJSON(&g); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 验证必填字段
	if g.DisplayName == "" {
		ErrorHandler(c, errors.New("displayName is required"), http.StatusBadRequest, "invalidValue")
		return
	}

	// 强制使用URL中的ID
	g.ID = id
	g.Schemas = []string{h.cfg.GroupSchema}

	if err := h.store.UpdateGroup(&g); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	c.JSON(http.StatusOK, g)
}

// PatchGroup 补丁更新组属性
// @Summary 补丁更新组
// @Description 使用补丁操作更新组属性
// @Tags Groups
// @Accept json
// @Produce json
// @Param id path string true "组ID"
// @Param patch body model.PatchRequest true "补丁操作"
// @Success 200 {object} model.Group "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "组不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups/{id} [patch]
// @Security BearerAuth
func (h *GroupHandlers) PatchGroup(c *gin.Context) {
	id := c.Param("id")
	var req model.PatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 验证Patch请求
	if err := h.validatePatchRequest(&req); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 执行补丁更新
	if err := h.store.PatchGroup(id, req.Operations); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 返回更新后的组
	group, err := h.store.GetGroup(id, true)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}
	group.Schemas = []string{h.cfg.GroupSchema}
	c.JSON(http.StatusOK, group)
}

// DeleteGroup 删除组
// @Summary 删除组
// @Description 根据组ID删除组
// @Tags Groups
// @Accept json
// @Produce json
// @Param id path string true "组ID"
// @Success 204 "删除成功"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "组不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups/{id} [delete]
// @Security BearerAuth
func (h *GroupHandlers) DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.DeleteGroup(id); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// SCIM标准：删除成功返回204 No Content
	c.Status(http.StatusNoContent)
}

// AddUserToGroup 添加用户到组
// @Summary 添加用户到组
// @Description 将用户添加到指定组
// @Tags Groups
// @Accept json
// @Produce json
// @Param id path string true "组ID"
// @Param member body model.Member true "成员信息"
// @Success 201 {object} model.Group "添加成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "组或用户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups/{id}/members [post]
// @Security BearerAuth
func (h *GroupHandlers) AddUserToGroup(c *gin.Context) {
	groupID := c.Param("id")
	var member model.Member
	if err := c.ShouldBindJSON(&member); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 验证成员值
	if member.Value == "" {
		ErrorHandler(c, errors.New("member value is required"), http.StatusBadRequest, "invalidValue")
		return
	}

	// 添加用户到组
	if err := h.store.AddUserToGroup(groupID, member.Value); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		if errors.Is(err, model.ErrUniqueness) {
			ErrorHandler(c, err, http.StatusConflict, "uniqueness")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 返回更新后的组
	group, err := h.store.GetGroup(groupID, true)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}
	group.Schemas = []string{h.cfg.GroupSchema}
	c.JSON(http.StatusCreated, group)
}

// RemoveUserFromGroup 从组中移除用户
// @Summary 从组中移除用户
// @Description 将用户从指定组中移除
// @Tags Groups
// @Accept json
// @Produce json
// @Param id path string true "组ID"
// @Param userId path string true "用户ID"
// @Success 204 "移除成功"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "组或用户不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups/{id}/members/{userId} [delete]
// @Security BearerAuth
func (h *GroupHandlers) RemoveUserFromGroup(c *gin.Context) {
	groupID := c.Param("id")
	userID := c.Param("userId")

	// 从组中移除用户
	if err := h.store.RemoveUserFromGroup(groupID, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// SCIM标准：删除成功返回204 No Content
	c.Status(http.StatusNoContent)
}

// ---------------------- User 辅助方法 ----------------------

// validateFilter 验证过滤器语法
func (h *UserHandlers) validateFilter(c *gin.Context, filter string) error {
	if filter == "" {
		return nil
	}
	if err := util.ValidateFilter(filter); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidFilter")
		return err
	}
	return nil
}

// needsGroups 检查是否需要加载用户的组信息
func (h *UserHandlers) needsGroups(q *model.ResourceQuery) bool {
	return strings.Contains(q.Attributes, "groups") ||
		strings.Contains(q.ExcludedAttributes, "groups")
}

// processUserList 处理用户列表并应用属性选择
func (h *UserHandlers) processUserList(users []model.User, q *model.ResourceQuery, needGroups bool) []interface{} {
	resources := make([]interface{}, 0, len(users))

	for i := range users {
		users[i].Schemas = []string{h.cfg.DefaultSchema}

		// 如果需要，加载用户所属的组
		if needGroups {
			groups, err := h.store.GetUserGroups(users[i].ID)
			if err != nil {
				return nil
			}
			users[i].Groups = groups
		}

		// 应用属性选择
		if q.Attributes != "" || q.ExcludedAttributes != "" {
			filtered, err := util.ApplyAttributeSelection(&users[i], q.Attributes, q.ExcludedAttributes)
			if err != nil {
				return nil
			}
			resources = append(resources, filtered)
		} else {
			resources = append(resources, users[i])
		}
	}

	return resources
}

// enrichUser 丰富用户信息（设置Schema和加载组信息）
func (h *UserHandlers) enrichUser(user *model.User, q *model.ResourceQuery) {
	user.Schemas = []string{h.cfg.DefaultSchema}

	// 检查是否需要加载用户的组信息
	if h.needsGroups(q) {
		groups, _ := h.store.GetUserGroups(user.ID)
		user.Groups = groups
	}
}

// validateUserRequiredFields 验证用户必填字段
func (h *UserHandlers) validateUserRequiredFields(u *model.User) error {
	if u.UserName == "" {
		return errors.New("userName is required")
	}
	if u.Name.GivenName == "" || u.Name.FamilyName == "" {
		return errors.New("name.givenName and name.familyName are required")
	}
	return nil
}

// validatePatchRequest 验证Patch请求
func (h *UserHandlers) validatePatchRequest(req *model.PatchRequest) error {
	// 校验Patch Schema
	if len(req.Schemas) == 0 || req.Schemas[0] != model.PatchSchema.String() {
		return errors.New("invalid patch schema")
	}

	// 验证操作类型
	validOps := map[string]bool{"add": true, "remove": true, "replace": true}
	for _, op := range req.Operations {
		if !validOps[op.Op] {
			return errors.New("invalid operation: " + op.Op)
		}
	}

	return nil
}

// ---------------------- Group 辅助方法 ----------------------

// validateFilter 验证过滤器语法
func (h *GroupHandlers) validateFilter(c *gin.Context, filter string) error {
	if filter == "" {
		return nil
	}
	if err := util.ValidateFilter(filter); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidFilter")
		return err
	}
	return nil
}

// processGroupList 处理组列表并应用属性选择
func (h *GroupHandlers) processGroupList(groups []model.Group, q *model.ResourceQuery) []interface{} {
	resources := make([]interface{}, 0, len(groups))

	for i := range groups {
		groups[i].Schemas = []string{h.cfg.GroupSchema}

		// 应用属性选择
		if q.Attributes != "" || q.ExcludedAttributes != "" {
			filtered, err := util.ApplyAttributeSelection(&groups[i], q.Attributes, q.ExcludedAttributes)
			if err != nil {
				return nil
			}
			resources = append(resources, filtered)
		} else {
			resources = append(resources, groups[i])
		}
	}

	return resources
}

// validatePatchRequest 验证Patch请求
func (h *GroupHandlers) validatePatchRequest(req *model.PatchRequest) error {
	// 校验Patch Schema
	if len(req.Schemas) == 0 || req.Schemas[0] != model.PatchSchema.String() {
		return errors.New("invalid patch schema")
	}

	// 验证操作类型
	validOps := map[string]bool{"add": true, "remove": true, "replace": true}
	for _, op := range req.Operations {
		if !validOps[op.Op] {
			return errors.New("invalid operation: " + op.Op)
		}
	}

	return nil
}
