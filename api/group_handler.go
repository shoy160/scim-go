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

// GroupHandlers Group相关接口处理器
type GroupHandlers struct {
	store store.Store
	cfg   *model.ScimConfig
}

// NewGroupHandlers 创建Group处理器
func NewGroupHandlers(s store.Store, cfg *model.ScimConfig) *GroupHandlers {
	return &GroupHandlers{store: s, cfg: cfg}
}

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
	preloadMembers := h.needsMembers(q)

	groups, total, err := h.store.ListGroups(q, preloadMembers)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 获取当前请求的host和proto
	host := c.Request.Host
	proto := "http"
	if c.Request.TLS != nil {
		proto = "https"
	}

	// 处理组列表并应用属性选择
	resources := h.processGroupList(groups, q, host, proto)
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
	preloadMembers := h.needsMembers(q)

	group, err := h.store.GetGroup(id, preloadMembers)
	if err != nil {
		ErrorHandler(c, err, http.StatusNotFound, "notFound")
		return
	}

	// 如果组没有 schemas，设置默认 schema（支持自定义 schemas）
	if len(group.Schemas) == 0 {
		group.Schemas = []string{h.cfg.GroupSchema}
	}

	// 从数据库填充 meta 数据
	host := c.Request.Host
	proto := "http"
	if c.Request.TLS != nil {
		proto = "https"
	}
	baseURL := proto + "://" + host
	h.populateGroupMeta(group, baseURL)

	// 应用属性选择
	if q.Attributes != "" || q.ExcludedAttributes != "" {
		// 验证属性格式
		if err := util.ValidateAttributeFormat(q.Attributes); err != nil {
			ErrorHandler(c, err, http.StatusBadRequest, "invalidSyntax")
			return
		}
		if err := util.ValidateAttributeFormat(q.ExcludedAttributes); err != nil {
			ErrorHandler(c, err, http.StatusBadRequest, "invalidSyntax")
			return
		}

		filtered, err := util.ApplyAttributeSelectionWithSpecialRules(group, q.Attributes, q.ExcludedAttributes, "members")
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

	// 生成唯一ID
	g.ID = uuid.NewString()
	// 如果用户没有提供 schemas，使用默认 schema
	if len(g.Schemas) == 0 {
		g.Schemas = []string{h.cfg.GroupSchema}
	}

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

	// 从数据库填充 meta 数据
	host := c.Request.Host
	proto := "http"
	if c.Request.TLS != nil {
		proto = "https"
	}
	baseURL := proto + "://" + host
	h.populateGroupMeta(&g, baseURL)

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
	// 如果用户没有提供 schemas，使用默认 schema
	if len(g.Schemas) == 0 {
		g.Schemas = []string{h.cfg.GroupSchema}
	}

	if err := h.store.UpdateGroup(&g); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 从数据库重新获取组以获取最新的 meta 数据
	updatedGroup, err := h.store.GetGroup(id, false)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 从数据库填充 meta 数据
	host := c.Request.Host
	proto := "http"
	if c.Request.TLS != nil {
		proto = "https"
	}
	baseURL := proto + "://" + host
	h.populateGroupMeta(updatedGroup, baseURL)

	c.JSON(http.StatusOK, updatedGroup)
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
	// 如果组没有 schemas，设置默认 schema（支持自定义 schemas）
	if len(group.Schemas) == 0 {
		group.Schemas = []string{h.cfg.GroupSchema}
	}

	// 从数据库填充 meta 数据
	host := c.Request.Host
	proto := "http"
	if c.Request.TLS != nil {
		proto = "https"
	}
	baseURL := proto + "://" + host
	h.populateGroupMeta(group, baseURL)

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

	// 设置默认类型为 User
	if member.Type == "" {
		member.Type = "User"
	}

	// 添加成员到组
	if err := h.store.AddMemberToGroup(groupID, member.Value, member.Type); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		if errors.Is(err, model.ErrUserAlreadyInGroup) {
			ErrorHandler(c, err, http.StatusConflict, "uniqueness")
			return
		}
		if errors.Is(err, model.ErrInvalidValue) {
			ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
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
	// 如果组没有 schemas，设置默认 schema（支持自定义 schemas）
	if len(group.Schemas) == 0 {
		group.Schemas = []string{h.cfg.GroupSchema}
	}
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
		if errors.Is(err, model.ErrUserNotInGroup) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// SCIM标准：删除成功返回204 No Content
	c.Status(http.StatusNoContent)
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
func (h *GroupHandlers) processGroupList(groups []model.Group, q *model.ResourceQuery, host string, proto string) []interface{} {
	resources := make([]interface{}, 0, len(groups))
	baseURL := proto + "://" + host

	for i := range groups {
		groups[i].Schemas = []string{h.cfg.GroupSchema}

		// 从数据库填充 meta 数据
		h.populateGroupMeta(&groups[i], baseURL)

		// 应用属性选择
		if q.Attributes != "" || q.ExcludedAttributes != "" {
			// 验证属性格式
			if err := util.ValidateAttributeFormat(q.Attributes); err != nil {
				return nil
			}
			if err := util.ValidateAttributeFormat(q.ExcludedAttributes); err != nil {
				return nil
			}

			filtered, err := util.ApplyAttributeSelectionWithSpecialRules(&groups[i], q.Attributes, q.ExcludedAttributes, "members")
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

// needsMembers 检查是否需要加载组的成员信息
func (h *GroupHandlers) needsMembers(q *model.ResourceQuery) bool {
	return strings.Contains(q.Attributes, "members")
}

// populateGroupMeta 从数据库字段填充组 meta 数据
func (h *GroupHandlers) populateGroupMeta(group *model.Group, baseURL string) {
	// ResourceType 动态生成，不持久化
	group.Meta.ResourceType = "Group"

	// 从数据库时间戳生成 ISO 8601 格式时间（使用纳秒精度）
	if !group.CreatedAt.IsZero() {
		group.Meta.Created = group.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00")
	}
	if !group.UpdatedAt.IsZero() {
		group.Meta.LastModified = group.UpdatedAt.Format("2006-01-02T15:04:05.999999999Z07:00")
	}

	// 动态生成 location
	group.Meta.Location = baseURL + "/scim/v2/Groups/" + group.ID

	// 动态生成成员的 $ref
	for i := range group.Members {
		if group.Members[i].Type == "Group" {
			group.Members[i].Ref = baseURL + "/scim/v2/Groups/" + group.Members[i].Value
		} else {
			// 默认类型为 User
			group.Members[i].Type = "User"
			group.Members[i].Ref = baseURL + "/scim/v2/Users/" + group.Members[i].Value
		}
	}

	// 使用数据库中存储的版本号
	group.Meta.Version = group.Version
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
