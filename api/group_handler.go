package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	proto := GetRequestProtocol(c)

	// 处理组列表并应用属性选择
	resources := h.processGroupList(groups, q, host, proto)
	if resources == nil {
		ErrorHandler(c, errors.New("failed to process group list"), http.StatusInternalServerError, "internalError")
		return
	}

	// 构造SCIM标准列表响应
	response := util.NewListResponse(h.cfg.ListSchema, int(total), q.StartIndex, q.Count, resources)
	c.JSON(http.StatusOK, response)
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
	proto := GetRequestProtocol(c)
	baseURL := proto + "://" + host
	h.populateGroupMeta(group, baseURL)

	// 应用属性选择（属性格式已在中间件验证）
	if q.Attributes != "" || q.ExcludedAttributes != "" {
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
	proto := GetRequestProtocol(c)
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
	proto := GetRequestProtocol(c)
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
	proto := GetRequestProtocol(c)
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

// AddMembersToGroup 添加成员到组
// @Summary 添加成员到组
// @Description 将一个或多个成员添加到指定组
// @Tags Groups
// @Accept json
// @Produce json
// @Param id path string true "组ID"
// @Param member body model.Member false "单个成员信息"
// @Param members body []model.Member false "成员信息数组"
// @Success 201 {object} model.Group "添加成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "组或成员不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups/{id}/members [post]
// @Security BearerAuth
func (h *GroupHandlers) AddMembersToGroup(c *gin.Context) {
	groupID := c.Param("id")

	// 读取请求体到缓冲区
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(c.Request.Body); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 尝试绑定为数组
	var members []model.Member
	if err := json.Unmarshal(buf.Bytes(), &members); err == nil {
		// 验证成员数组
		if len(members) == 0 {
			ErrorHandler(c, errors.New("members array is required and cannot be empty"), http.StatusBadRequest, "invalidValue")
			return
		}

		// 处理每个成员
		for _, member := range members {
			if err := h.processMember(c, groupID, member); err != nil {
				return
			}
		}
	} else {
		// 尝试绑定为单个成员
		var member model.Member
		if err := json.Unmarshal(buf.Bytes(), &member); err != nil {
			ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
			return
		}

		// 处理单个成员
		if err := h.processMember(c, groupID, member); err != nil {
			return
		}
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

	// 填充组的 meta 数据和成员的 $ref 属性
	host := c.Request.Host
	proto := GetRequestProtocol(c)
	baseURL := proto + "://" + host
	h.populateGroupMeta(group, baseURL)

	c.JSON(http.StatusCreated, group)
}

// processMember 处理单个成员的添加逻辑
func (h *GroupHandlers) processMember(c *gin.Context, groupID string, member model.Member) error {
	// 验证成员值
	if member.Value == "" {
		ErrorHandler(c, errors.New("member value is required"), http.StatusBadRequest, "invalidValue")
		return errors.New("member value is required")
	}

	// 验证并设置成员类型
	memberType, err := util.ValidateMemberType(string(member.Type))
	if err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return err
	}

	// 添加成员到组
	if err := h.store.AddMemberToGroup(groupID, member.Value, memberType); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return err
		}
		if errors.Is(err, model.ErrMemberAlreadyInGroup) {
			ErrorHandler(c, err, http.StatusConflict, "uniqueness")
			return err
		}
		if errors.Is(err, model.ErrInvalidValue) {
			ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
			return err
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return err
	}

	return nil
}

// RemoveMemberFromGroup 从组中移除成员
// @Summary 从组中移除成员
// @Description 将成员从指定组中移除
// @Tags Groups
// @Accept json
// @Produce json
// @Param id path string true "组ID"
// @Param userId path string true "成员ID"
// @Success 204 "移除成功"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "组或成员不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups/{id}/members/{userId} [delete]
// @Security BearerAuth
func (h *GroupHandlers) RemoveMemberFromGroup(c *gin.Context) {
	groupID := c.Param("id")
	memberID := c.Param("userId")

	// 从组中移除成员（支持User和Group类型）
	if err := h.store.RemoveMemberFromGroup(groupID, memberID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		if errors.Is(err, model.ErrMemberNotInGroup) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// SCIM标准：删除成功返回204 No Content
	c.Status(http.StatusNoContent)
}

// GetGroupMembers 获取组成员
// @Summary 获取组成员
// @Description 获取指定组的成员列表，支持分页和成员类型过滤
// @Tags Groups
// @Accept json
// @Produce json
// @Param id path string true "组ID"
// @Param startIndex query int false "起始索引（从1开始）"
// @Param count query int false "每页数量"
// @Param memberType query string false "成员类型过滤（User或Group）"
// @Success 200 {object} model.ListResponse "组成员列表"
// @Failure 401 {object} model.ErrorResponse "未授权"
// @Failure 404 {object} model.ErrorResponse "组不存在"
// @Failure 500 {object} model.ErrorResponse "服务器内部错误"
// @Router /Groups/{id}/members [get]
// @Security BearerAuth
func (h *GroupHandlers) GetGroupMembers(c *gin.Context) {
	groupID := c.Param("id")
	q := c.MustGet("scim_query").(*model.ResourceQuery)
	memberTypeStr := c.Query("memberType")

	// 验证 memberType 参数
	var memberType model.MemberType
	if memberTypeStr != "" {
		if mt, ok := model.ParseMemberType(memberTypeStr); ok {
			memberType = mt
		} else {
			ErrorHandler(c, errors.New("invalid memberType: must be 'User' or 'Group'"), http.StatusBadRequest, "invalidValue")
			return
		}
	}

	members, total, err := h.store.GetGroupMembers(groupID, memberType, q)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	host := c.Request.Host
	proto := GetRequestProtocol(c)
	baseURL := proto + "://" + host

	// 生成成员的$ref属性
	util.GenerateMembersRef(members, baseURL, h.cfg.APIPath)

	// 将members转换为[]interface{}
	resources := make([]interface{}, len(members))
	for i, member := range members {
		resources[i] = member
	}
	response := util.NewListResponse(h.cfg.ListSchema, int(total), q.StartIndex, q.Count, resources)
	c.JSON(http.StatusOK, response)
}

// parseInt 解析字符串为整数
func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// ---------------------- Group 辅助方法 ----------------------

// validateFilter 验证过滤器语法
func (h *GroupHandlers) validateFilter(c *gin.Context, filter string) error {
	return ValidateFilter(c, filter, ErrorHandler)
}

// processGroupList 处理组列表并应用属性选择
func (h *GroupHandlers) processGroupList(groups []model.Group, q *model.ResourceQuery, host string, proto string) []interface{} {
	resources := make([]interface{}, 0, len(groups))
	baseURL := proto + "://" + host

	for i := range groups {
		groups[i].Schemas = []string{h.cfg.GroupSchema}

		// 从数据库填充 meta 数据
		h.populateGroupMeta(&groups[i], baseURL)

		// 应用属性选择（属性格式已在中间件验证）
		if q.Attributes != "" || q.ExcludedAttributes != "" {
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
	// 填充组的meta数据
	group.Meta = util.PopulateMeta("Group", group.ID, group.CreatedAt, group.UpdatedAt, group.Version, baseURL, h.cfg.APIPath)

	// 动态生成成员的 $ref
	util.GenerateMembersRef(group.Members, baseURL, h.cfg.APIPath)
}

// validatePatchRequest 验证Patch请求
func (h *GroupHandlers) validatePatchRequest(req *model.PatchRequest) error {
	return util.ValidatePatchRequest(req)
}
