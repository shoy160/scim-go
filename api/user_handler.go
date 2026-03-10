package api

import (
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

	// 构建列表请求
	req := ListRequest{
		Query: q,
		ValidateFilter: func(filter string) error {
			return h.validateFilter(c, filter)
		},
		ListFunc: func(q *model.ResourceQuery) ([]interface{}, int64, error) {
			users, total, err := h.store.ListUsers(q)
			if err != nil {
				return nil, 0, err
			}
			resources := make([]interface{}, len(users))
			for i, user := range users {
				resources[i] = user
			}
			return resources, total, nil
		},
		ProcessFunc: func(resources []interface{}, q *model.ResourceQuery, host, proto string) []interface{} {
			needGroups := h.needsGroups(q)
			users := make([]model.User, len(resources))
			for i, resource := range resources {
				users[i] = resource.(model.User)
			}
			return h.processUserList(users, q, needGroups, host, proto)
		},
		Schema: h.cfg.ListSchema,
	}

	// 处理列表请求
	HandleList(c, req)
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

	// 构建获取请求
	req := GetRequest{
		ID:    id,
		Query: q,
		GetFunc: func(id string, preload bool) (interface{}, error) {
			return h.store.GetUser(id)
		},
		ProcessFunc: func(resource interface{}, q *model.ResourceQuery, host, proto string) error {
			user := resource.(*model.User)
			baseURL := proto + "://" + host

			// 设置Schema并加载组信息
			h.enrichUser(user, q, baseURL)

			// 处理企业扩展属性
			h.ProcessEnterpriseExtension(user)

			// 从数据库填充 meta 数据
			h.populateUserMeta(user, baseURL)

			return nil
		},
		AttributeName: "groups",
		Schema:        h.cfg.DefaultSchema,
	}

	// 处理获取请求
	HandleGet(c, req)
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

	// 生成唯一ID
	u.ID = uuid.NewString()
	// 如果用户没有提供 schemas，使用默认 schema
	if len(u.Schemas) == 0 {
		u.Schemas = []string{h.cfg.DefaultSchema}
	}

	// 处理企业扩展属性
	h.ProcessEnterpriseExtension(&u)

	// 构建创建请求
	req := CreateRequest{
		Resource: &u,
		ValidateFunc: func(resource interface{}) (string, error) {
			user := resource.(*model.User)
			return h.validateUserRequiredFields(user)
		},
		CreateFunc: func(resource interface{}) error {
			user := resource.(*model.User)
			return h.store.CreateUser(user)
		},
		ProcessFunc: func(resource interface{}, host, proto string) error {
			user := resource.(*model.User)
			baseURL := proto + "://" + host
			h.populateUserMeta(user, baseURL)
			return nil
		},
		Schema: h.cfg.DefaultSchema,
	}

	// 处理创建请求
	HandleCreate(c, req)
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

	// 检查配置是否存在
	if h.cfg == nil {
		ErrorHandler(c, errors.New("configuration is nil"), http.StatusInternalServerError, "internalError")
		return
	}

	// 强制使用URL中的ID
	u.ID = id
	// 如果用户没有提供 schemas，使用默认 schema
	if len(u.Schemas) == 0 {
		u.Schemas = []string{h.cfg.DefaultSchema}
	}

	// 处理企业扩展属性
	h.ProcessEnterpriseExtension(&u)

	// 构建更新请求
	req := UpdateRequest{
		ID:       id,
		Resource: &u,
		ValidateFunc: func(resource interface{}) (string, error) {
			user := resource.(*model.User)
			return h.validateUserRequiredFields(user)
		},
		UpdateFunc: func(resource interface{}) error {
			user := resource.(*model.User)
			return h.store.UpdateUser(user)
		},
		GetFunc: func(id string) (interface{}, error) {
			return h.store.GetUser(id)
		},
		ProcessFunc: func(resource interface{}, host, proto string) error {
			user := resource.(*model.User)

			// 处理企业扩展属性
			h.ProcessEnterpriseExtension(user)

			// 从数据库填充 meta 数据
			baseURL := proto + "://" + host
			h.populateUserMeta(user, baseURL)

			return nil
		},
		Schema: h.cfg.DefaultSchema,
	}

	// 处理更新请求
	HandleUpdate(c, req)
}

// PatchUser 补丁更新用户属性
// @Summary 补丁更新用户
// @Description 使用补丁操作更新用户属性。支持两种操作模式：1) 指定path参数更新特定属性；2) path为空时，value对象包含完整的资源属性修改
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param patch body model.PatchRequest true "补丁操作。当path为空时，value必须是对象类型，包含需要修改的资源属性"
// @Success 200 {object} model.User "更新成功"
// @Failure 400 {object} model.ErrorResponse "请求参数错误 - 当path为空时，value必须是非空对象"
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

	// 构建补丁请求
	reqPatch := PatchRequest{
		ID:  id,
		Ops: req.Operations,
		GetFunc: func(id string) (interface{}, error) {
			return h.store.GetUser(id)
		},
		PatchFunc: func(id string, ops []model.PatchOperation) error {
			return h.store.PatchUser(id, ops)
		},
		ProcessFunc: func(resource interface{}, host, proto string) error {
			user := resource.(*model.User)

			// 如果用户没有 schemas，设置默认 schema（支持自定义 schemas）
			if len(user.Schemas) == 0 {
				user.Schemas = []string{h.cfg.DefaultSchema}
			}

			// 处理企业扩展属性
			h.ProcessEnterpriseExtension(user)

			// 从数据库填充 meta 数据
			baseURL := proto + "://" + host
			h.populateUserMeta(user, baseURL)

			return nil
		},
		Schema: h.cfg.DefaultSchema,
	}

	// 处理补丁请求
	HandlePatch(c, reqPatch)
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

	// 构建删除请求
	req := DeleteRequest{
		ID: id,
		DeleteFunc: func(id string) error {
			return h.store.DeleteUser(id)
		},
	}

	// 处理删除请求
	HandleDelete(c, req)
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
		// 初始化 schemas
		users[i].Schemas = []string{h.cfg.DefaultSchema}

		// 处理企业扩展属性
		h.ProcessEnterpriseExtension(&users[i])

		// 从数据库填充 meta 数据
		h.populateUserMeta(&users[i], baseURL)

		// 如果需要，加载用户所属的组
		if needGroups {
			groups, err := h.store.GetMemberGroups(users[i].ID)
			if err != nil {
				return nil
			}
			for idx := range groups {
				groups[idx].Ref = util.ResolveRef(baseURL, h.cfg.APIPath, "Group", groups[idx].Value)
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
func (h *UserHandlers) enrichUser(user *model.User, q *model.ResourceQuery, baseURL string) {
	// 如果用户没有 schemas，设置默认 schema
	if len(user.Schemas) == 0 {
		user.Schemas = []string{h.cfg.DefaultSchema}
	}

	// 处理企业扩展属性（只检查属性，不更新 schemas，因为 schemas 会在其他地方处理）
	if user.EnterpriseUserExtension != nil {
		// 检查 manager 是否为空，如果为空则设为 nil
		if user.EnterpriseUserExtension.Manager != nil && (user.EnterpriseUserExtension.Manager.Value == "" && user.EnterpriseUserExtension.Manager.Ref == "") {
			user.EnterpriseUserExtension.Manager = nil
		}

		// 检查是否有任何企业扩展属性
		hasEnterpriseAttrs := user.EnterpriseUserExtension.EmployeeNumber != "" ||
			user.EnterpriseUserExtension.CostCenter != "" ||
			user.EnterpriseUserExtension.Organization != "" ||
			user.EnterpriseUserExtension.Division != "" ||
			user.EnterpriseUserExtension.Department != "" ||
			user.EnterpriseUserExtension.Manager != nil

		// 如果不包含企业扩展属性，将 EnterpriseUserExtension 设为 nil
		if !hasEnterpriseAttrs {
			user.EnterpriseUserExtension = nil
		}
	}

	// 检查是否需要加载用户的组信息
	if h.needsGroups(q) {
		groups, _ := h.store.GetMemberGroups(user.ID)
		// 动态添加 $ref 字段
		for i := range groups {
			groups[i].Ref = util.ResolveRef(baseURL, h.cfg.APIPath, "Group", groups[i].Value)
		}
		user.Groups = groups
	}
}

// validateUserRequiredFields 验证用户必填字段
// 根据 SCIM 2.0 规范（RFC 7644），只有 userName 是必填字段
func (h *UserHandlers) validateUserRequiredFields(u *model.User) (string, error) {
	if u.UserName == "" {
		return "userName", errors.New("userName is required")
	}

	// givenName 和 familyName 改为非必填项，符合 SCIM 2.0 规范
	// name 属性整体是可选的，其中的子字段也是可选的

	// 检查重复的 emails
	if len(u.Emails) > 0 {
		emailMap := make(map[string]bool)
		for _, email := range u.Emails {
			if email.Value == "" {
				return "emails", errors.New("email value cannot be empty")
			}
			if emailMap[email.Value] {
				return "emails", fmt.Errorf("duplicate email address: %s", email.Value)
			}
			emailMap[email.Value] = true
		}
	}

	// 检查重复的 roles
	if len(u.Roles) > 0 {
		roleMap := make(map[string]bool)
		for _, role := range u.Roles {
			if role.Value == "" {
				return "roles", errors.New("role value cannot be empty")
			}
			if roleMap[role.Value] {
				return "roles", fmt.Errorf("duplicate role: %s", role.Value)
			}
			roleMap[role.Value] = true
		}
	}

	return "", nil
}

// ProcessEnterpriseExtension 处理企业扩展属性
// 检查企业扩展属性是否为空，更新 schemas 列表，并处理 manager 字段
func (h *UserHandlers) ProcessEnterpriseExtension(user *model.User) bool {
	enterpriseSchema := model.EnterpriseUserSchema.String()
	hasEnterpriseAttrs := false

	if user.EnterpriseUserExtension != nil {
		// 检查 manager 是否为空，如果为空则设为 nil
		if user.EnterpriseUserExtension.Manager != nil && (user.EnterpriseUserExtension.Manager.Value == "" && user.EnterpriseUserExtension.Manager.Ref == "") {
			user.EnterpriseUserExtension.Manager = nil
		}

		// 检查是否有任何企业扩展属性
		hasEnterpriseAttrs = user.EnterpriseUserExtension.EmployeeNumber != "" ||
			user.EnterpriseUserExtension.CostCenter != "" ||
			user.EnterpriseUserExtension.Organization != "" ||
			user.EnterpriseUserExtension.Division != "" ||
			user.EnterpriseUserExtension.Department != "" ||
			user.EnterpriseUserExtension.Manager != nil
	}

	// 根据企业扩展属性的状态更新 schemas
	if hasEnterpriseAttrs {
		if !util.ContainsSchema(user.Schemas, enterpriseSchema) {
			user.Schemas = append(user.Schemas, enterpriseSchema)
		}
	} else {
		// 如果不包含企业扩展属性，将 EnterpriseUserExtension 设为 nil，避免返回空的 manager 对象
		user.EnterpriseUserExtension = nil
		// 从 schemas 中移除企业扩展 schema
		user.Schemas = util.RemoveSchema(user.Schemas, enterpriseSchema)
	}

	return hasEnterpriseAttrs
}

// validatePatchRequest 验证Patch请求
func (h *UserHandlers) validatePatchRequest(req *model.PatchRequest) error {
	return util.ValidatePatchRequest(req)
}
