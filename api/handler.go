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
	if q.Filter != "" {
		if err := util.ValidateFilter(q.Filter); err != nil {
			ErrorHandler(c, err, http.StatusBadRequest, "invalidFilter")
			return
		}
	}

	users, total, err := h.store.ListUsers(q)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 检查是否需要加载用户的组信息
	needGroups := strings.Contains(q.Attributes, "groups") || strings.Contains(q.ExcludedAttributes, "groups")

	// 给每个用户设置默认Schema并应用属性选择
	var resources []interface{}
	for i := range users {
		users[i].Schemas = []string{h.cfg.DefaultSchema}

		// 如果需要，加载用户所属的组
		if needGroups {
			groups, err := h.store.GetUserGroups(users[i].ID)
			if err != nil {
				ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
				return
			}
			users[i].Groups = groups
		}

		// 应用属性选择
		if q.Attributes != "" || q.ExcludedAttributes != "" {
			filtered, err := util.ApplyAttributeSelection(&users[i], q.Attributes, q.ExcludedAttributes)
			if err != nil {
				ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
				return
			}
			resources = append(resources, filtered)
		} else {
			resources = append(resources, users[i])
		}
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
	user.Schemas = []string{h.cfg.DefaultSchema}

	// 检查是否需要加载用户的组信息
	needGroups := strings.Contains(q.Attributes, "groups") || strings.Contains(q.ExcludedAttributes, "groups")
	if needGroups {
		groups, err := h.store.GetUserGroups(user.ID)
		if err != nil {
			ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
			return
		}
		user.Groups = groups
	}

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
	if u.UserName == "" {
		ErrorHandler(c, errors.New("userName is required"), http.StatusBadRequest, "invalidValue")
		return
	}
	if u.Name.GivenName == "" || u.Name.FamilyName == "" {
		ErrorHandler(c, errors.New("name.givenName and name.familyName are required"), http.StatusBadRequest, "invalidValue")
		return
	}

	// 生成唯一ID
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

	// 校验Patch Schema
	if len(req.Schemas) == 0 || req.Schemas[0] != model.PatchSchema {
		ErrorHandler(c, errors.New("invalid patch schema"), http.StatusBadRequest, "invalidSchema")
		return
	}

	// 验证操作
	for _, op := range req.Operations {
		if op.Op != "add" && op.Op != "remove" && op.Op != "replace" {
			ErrorHandler(c, errors.New("invalid operation: "+op.Op), http.StatusBadRequest, "invalidValue")
			return
		}
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
	if q.Filter != "" {
		if err := util.ValidateFilter(q.Filter); err != nil {
			ErrorHandler(c, err, http.StatusBadRequest, "invalidFilter")
			return
		}
	}
	preloadMembers := false
	if q.Attributes != "" {
		attrs := strings.Split(q.Attributes, ",")
		var filtered []string
		for _, attr := range attrs {
			if strings.TrimSpace(attr) == "members" {
				preloadMembers = true
			} else {
				filtered = append(filtered, attr)
			}
		}
		q.Attributes = strings.Join(filtered, ",")
	}
	groups, total, err := h.store.ListGroups(q, preloadMembers)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 给每个组设置默认Schema并应用属性选择
	var resources []interface{}
	for i := range groups {
		groups[i].Schemas = []string{h.cfg.GroupSchema}

		// 应用属性选择
		if q.Attributes != "" || q.ExcludedAttributes != "" {
			filtered, err := util.ApplyAttributeSelection(&groups[i], q.Attributes, q.ExcludedAttributes)
			if err != nil {
				ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
				return
			}
			resources = append(resources, filtered)
		} else {
			resources = append(resources, groups[i])
		}
	}

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

	preloadMembers := false
	if q.Attributes != "" {
		attrs := strings.Split(q.Attributes, ",")
		var filtered []string
		for _, attr := range attrs {
			if strings.TrimSpace(attr) == "members" {
				preloadMembers = true
			} else {
				filtered = append(filtered, attr)
			}
		}
		q.Attributes = strings.Join(filtered, ",")
	}
	group, err := h.store.GetGroup(id, preloadMembers)
	if err != nil {
		ErrorHandler(c, err, http.StatusNotFound, "notFound")
		return
	}
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
// @Description 创建新组，可以包含成员
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

	// 验证成员（如果提供了成员）
	if len(g.Members) > 0 {
		for _, member := range g.Members {
			if member.Value == "" {
				ErrorHandler(c, errors.New("member value (userId) is required"), http.StatusBadRequest, "invalidValue")
				return
			}

			// 验证用户是否存在
			_, err := h.store.GetUser(member.Value)
			if err != nil {
				ErrorHandler(c, errors.New("user not found: "+member.Value), http.StatusBadRequest, "invalidValue")
				return
			}

			// 设置 GroupID
			member.GroupID = g.ID
		}
	}

	g.ID = uuid.NewString()
	g.Schemas = []string{h.cfg.GroupSchema}

	if err := h.store.CreateGroup(&g); err != nil {
		if errors.Is(err, model.ErrUniqueness) {
			ErrorHandler(c, err, http.StatusConflict, "uniqueness")
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

	if len(req.Schemas) == 0 || req.Schemas[0] != model.PatchSchema {
		ErrorHandler(c, errors.New("invalid patch schema"), http.StatusBadRequest, "invalidSchema")
		return
	}

	// 验证操作
	for _, op := range req.Operations {
		if op.Op != "add" && op.Op != "remove" && op.Op != "replace" {
			ErrorHandler(c, errors.New("invalid operation: "+op.Op), http.StatusBadRequest, "invalidValue")
			return
		}
	}

	if err := h.store.PatchGroup(id, req.Operations); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	group, err := h.store.GetGroup(id, false)
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

	c.Status(http.StatusNoContent)
}

// AddUserToGroup 添加用户到组
// @Summary 添加用户到组
// @Description 将用户添加到指定组
// @Tags Groups
// @Accept json
// @Produce json
// @Param id path string true "组ID"
// @Param request body object true "用户ID" example({"userId":"user-123"})
// @Success 200 {object} model.Group "添加成功，返回更新后的组信息"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "组或用户不存在"
// @Failure 409 {object} model.ErrorResponse "用户已在组中"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups/{id}/members [post]
// @Security BearerAuth
func (h *GroupHandlers) AddUserToGroup(c *gin.Context) {
	groupID := c.Param("id")

	var req struct {
		UserID string `json:"userId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	if err := h.store.AddUserToGroup(groupID, req.UserID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		if err.Error() == "user already in group" {
			ErrorHandler(c, err, http.StatusConflict, "uniqueness")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	group, err := h.store.GetGroup(groupID, false)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}
	group.Schemas = []string{h.cfg.GroupSchema}
	c.JSON(http.StatusOK, group)
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

	if err := h.store.RemoveUserFromGroup(groupID, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		if err.Error() == "user not in group" {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	c.Status(http.StatusNoContent)
}
