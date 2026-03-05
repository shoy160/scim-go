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

// NewUserHandlers 创建User处理器
func NewUserHandlers(s store.Store, cfg *model.ScimConfig) *UserHandlers {
	return &UserHandlers{store: s, cfg: cfg}
}

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
	// 获取查询参数
	q := c.MustGet("scim_query").(*model.ResourceQuery)

	// 验证过滤器语法
	if err := h.validateFilter(c, q.Filter); err != nil {
		return
	}

	// 获取用户列表
	users, total, err := h.store.ListUsers(q)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 检查是否需要加载用户的组信息
	needGroups := h.needsGroups(q)

	// 获取当前请求的协议和主机
	proto := GetRequestProtocol(c)
	host := c.Request.Host

	// 处理用户列表并应用属性选择
	resources := h.processUserList(users, q, needGroups, host, proto)
	if resources == nil {
		ErrorHandler(c, errors.New("failed to process user list"), http.StatusInternalServerError, "internalError")
		return
	}

	// 构造SCIM标准列表响应
	response := util.NewListResponse(h.cfg.ListSchema, int(total), q.StartIndex, q.Count, resources)
	c.JSON(http.StatusOK, response)
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

	// 从数据库填充 meta 数据
	host := c.Request.Host
	proto := GetRequestProtocol(c)
	baseURL := proto + "://" + host
	h.populateUserMeta(user, baseURL)

	// 应用属性选择（属性格式已在中间件验证）
	if q.Attributes != "" || q.ExcludedAttributes != "" {
		filtered, err := util.ApplyAttributeSelectionWithSpecialRules(user, q.Attributes, q.ExcludedAttributes, "groups")
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

	// 生成唯一ID
	u.ID = uuid.NewString()
	// 如果用户没有提供 schemas，使用默认 schema
	if len(u.Schemas) == 0 {
		u.Schemas = []string{h.cfg.DefaultSchema}
	}

	// 保存用户
	if err := h.store.CreateUser(&u); err != nil {
		if errors.Is(err, model.ErrUniqueness) {
			ErrorHandler(c, err, http.StatusConflict, "uniqueness")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 从数据库填充 meta 数据
	host := c.Request.Host
	proto := GetRequestProtocol(c)
	baseURL := proto + "://" + host
	h.populateUserMeta(&u, baseURL)

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
	// 如果用户没有提供 schemas，使用默认 schema
	if len(u.Schemas) == 0 {
		u.Schemas = []string{h.cfg.DefaultSchema}
	}

	if err := h.store.UpdateUser(&u); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			ErrorHandler(c, err, http.StatusNotFound, "notFound")
			return
		}
		// 检查是否为重复性错误
		if strings.Contains(err.Error(), "duplicate") {
			ErrorHandler(c, err, http.StatusBadRequest, "uniqueness")
			return
		}
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 从数据库重新获取用户以获取最新的 meta 数据
	updatedUser, err := h.store.GetUser(id)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 从数据库填充 meta 数据
	host := c.Request.Host
	proto := GetRequestProtocol(c)
	baseURL := proto + "://" + host
	h.populateUserMeta(updatedUser, baseURL)

	c.JSON(http.StatusOK, updatedUser)
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
		// 检查是否为重复性错误
		if strings.Contains(err.Error(), "duplicate") {
			ErrorHandler(c, err, http.StatusBadRequest, "uniqueness")
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
	// 如果用户没有 schemas，设置默认 schema（支持自定义 schemas）
	if len(user.Schemas) == 0 {
		user.Schemas = []string{h.cfg.DefaultSchema}
	}

	// 从数据库填充 meta 数据
	host := c.Request.Host
	proto := GetRequestProtocol(c)
	baseURL := proto + "://" + host
	h.populateUserMeta(user, baseURL)

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

// ---------------------- User 辅助方法 ----------------------

// validateFilter 验证过滤器语法
func (h *UserHandlers) validateFilter(c *gin.Context, filter string) error {
	return ValidateFilter(c, filter, ErrorHandler)
}

// needsGroups 检查是否需要加载用户的组信息
// 返回是否需要加载 groups，但不修改 q.Attributes
func (h *UserHandlers) needsGroups(q *model.ResourceQuery) bool {
	if q.Attributes == "" {
		return true
	}
	// 检查是否包含 "groups" 或以 "groups." 开头的属性
	attrs := util.ParseAttributeList(q.Attributes)
	for _, attr := range attrs {
		if attr == "groups" || strings.HasPrefix(attr, "groups.") || attr == "groups.*" {
			return true
		}
	}
	return false
}

// processUserList 处理用户列表并应用属性选择
func (h *UserHandlers) processUserList(users []model.User, q *model.ResourceQuery, needGroups bool, host string, proto string) []interface{} {
	resources := make([]interface{}, 0, len(users))
	baseURL := proto + "://" + host

	for i := range users {
		users[i].Schemas = []string{h.cfg.DefaultSchema}

		// 从数据库填充 meta 数据
		h.populateUserMeta(&users[i], baseURL)

		// 如果需要，加载用户所属的组
		if needGroups {
			groups, err := h.store.GetMemberGroups(users[i].ID)
			if err != nil {
				return nil
			}
			users[i].Groups = groups
		}
		// 应用属性选择（属性格式已在中间件验证）
		if q.Attributes != "" || q.ExcludedAttributes != "" {
			filtered, err := util.ApplyAttributeSelectionWithSpecialRules(&users[i], q.Attributes, q.ExcludedAttributes, "groups")
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

// populateUserMeta 从数据库字段填充用户 meta 数据
func (h *UserHandlers) populateUserMeta(user *model.User, baseURL string) {
	user.Meta = util.PopulateMeta("User", user.ID, user.CreatedAt, user.UpdatedAt, user.Version, baseURL, h.cfg.APIPath)
}

// enrichUser 丰富用户信息（设置Schema和加载组信息）
func (h *UserHandlers) enrichUser(user *model.User, q *model.ResourceQuery) {
	// 如果用户没有 schemas，设置默认 schema
	if len(user.Schemas) == 0 {
		user.Schemas = []string{h.cfg.DefaultSchema}
	}

	// 检查是否需要加载用户的组信息
	if h.needsGroups(q) {
		groups, _ := h.store.GetMemberGroups(user.ID)
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
	return util.ValidatePatchRequest(req)
}
