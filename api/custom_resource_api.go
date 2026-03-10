package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"scim-go/model"
	"scim-go/util"
)

// @Summary 创建自定义资源类型
// @Description 创建新的自定义资源类型，需要提供ID、名称、端点和模式
// @Tags CustomResourceTypes
// @Accept json
// @Produce json
// @Param customResourceType body model.CustomResourceType true "自定义资源类型信息"
// @Success 201 {object} model.CustomResourceType
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /scim/v2/CustomResourceTypes [post]
func (reg *RouteRegistrar) createCustomResourceType(c *gin.Context) {
	var crt model.CustomResourceType
	if err := c.ShouldBindJSON(&crt); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidRequest")
		return
	}

	// 验证自定义资源类型
	if err := model.ValidateCustomResourceType(&crt); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 生成ID（如果未提供）
	if crt.ID == "" {
		crt.ID = util.GenerateID()
	}

	// 创建自定义资源类型
	err := reg.store.CreateCustomResourceType(&crt)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 获取完整的基础URL
	baseURL := getBaseURL(c)
	// 设置完整的Endpoint和Meta
	crt.Endpoint = baseURL + reg.cfg.APIPath + crt.Endpoint
	crt.Meta.Location = baseURL + reg.cfg.APIPath + "/ResourceTypes/" + crt.ID

	c.JSON(http.StatusCreated, crt)
}

// @Summary 获取自定义资源类型
// @Description 根据ID获取自定义资源类型的详细信息
// @Tags CustomResourceTypes
// @Accept json
// @Produce json
// @Param id path string true "自定义资源类型ID"
// @Success 200 {object} model.CustomResourceType
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /scim/v2/CustomResourceTypes/{id} [get]
func (reg *RouteRegistrar) getCustomResourceType(c *gin.Context) {
	id := c.Param("id")
	crt, err := reg.store.GetCustomResourceType(id)
	if err != nil {
		ErrorHandler(c, err, http.StatusNotFound, "notFound")
		return
	}

	// 获取完整的基础URL
	baseURL := getBaseURL(c)
	// 设置完整的Endpoint和Meta
	crt.Endpoint = baseURL + reg.cfg.APIPath + crt.Endpoint
	crt.Meta.Location = baseURL + reg.cfg.APIPath + "/ResourceTypes/" + crt.ID

	c.JSON(http.StatusOK, crt)
}

// @Summary 列出自定义资源类型
// @Description 获取自定义资源类型列表，支持分页
// @Tags CustomResourceTypes
// @Accept json
// @Produce json
// @Param startIndex query int false "起始索引"
// @Param count query int false "每页数量"
// @Success 200 {object} model.ListResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /scim/v2/CustomResourceTypes [get]
func (reg *RouteRegistrar) listCustomResourceTypes(c *gin.Context) {
	customResourceTypes, err := reg.store.ListCustomResourceTypes()
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 获取完整的基础URL
	baseURL := getBaseURL(c)
	// 设置完整的Endpoint和Meta
	for i := range customResourceTypes {
		crt := &customResourceTypes[i]
		crt.Endpoint = baseURL + reg.cfg.APIPath + crt.Endpoint
		crt.Meta.Location = baseURL + reg.cfg.APIPath + "/ResourceTypes/" + crt.ID
	}

	c.JSON(http.StatusOK, model.ListResponse{
		Schemas:      []string{model.ListSchema.String()},
		TotalResults: len(customResourceTypes),
		Resources:    customResourceTypes,
	})
}

// @Summary 更新自定义资源类型
// @Description 更新现有的自定义资源类型信息
// @Tags CustomResourceTypes
// @Accept json
// @Produce json
// @Param id path string true "自定义资源类型ID"
// @Param customResourceType body model.CustomResourceType true "自定义资源类型信息"
// @Success 200 {object} model.CustomResourceType
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /scim/v2/CustomResourceTypes/{id} [put]
func (reg *RouteRegistrar) updateCustomResourceType(c *gin.Context) {
	id := c.Param("id")
	var crt model.CustomResourceType
	if err := c.ShouldBindJSON(&crt); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidRequest")
		return
	}

	// 确保ID匹配
	if crt.ID != id {
		ErrorHandler(c, model.ErrInvalidValue, http.StatusBadRequest, "invalidValue")
		return
	}

	// 验证自定义资源类型
	if err := model.ValidateCustomResourceType(&crt); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidValue")
		return
	}

	// 更新自定义资源类型
	err := reg.store.UpdateCustomResourceType(&crt)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 获取完整的基础URL
	baseURL := getBaseURL(c)
	// 设置完整的Endpoint和Meta
	crt.Endpoint = baseURL + reg.cfg.APIPath + crt.Endpoint
	crt.Meta.Location = baseURL + reg.cfg.APIPath + "/ResourceTypes/" + crt.ID

	c.JSON(http.StatusOK, crt)
}

// @Summary 删除自定义资源类型
// @Description 根据ID删除自定义资源类型
// @Tags CustomResourceTypes
// @Accept json
// @Produce json
// @Param id path string true "自定义资源类型ID"
// @Success 204 {object} model.CustomResourceType
// @Failure 500 {object} model.ErrorResponse
// @Router /scim/v2/CustomResourceTypes/{id} [delete]
func (reg *RouteRegistrar) deleteCustomResourceType(c *gin.Context) {
	id := c.Param("id")
	err := reg.store.DeleteCustomResourceType(id)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary 列出自定义资源
// @Description 获取指定类型的自定义资源列表，支持分页和过滤
// @Tags CustomResources
// @Accept json
// @Produce json
// @Param resourceType path string true "资源类型"
// @Param startIndex query int false "起始索引"
// @Param count query int false "每页数量"
// @Param filter query string false "过滤条件"
// @Param sortBy query string false "排序字段"
// @Param sortOrder query string false "排序顺序"
// @Success 200 {object} model.ListResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /scim/v2/{resourceType} [get]
func (reg *RouteRegistrar) listCustomResources(c *gin.Context) {
	resourceType := c.Param("resourceType")
	if resourceType == "" {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 验证资源类型是否存在
	_, err := reg.store.GetCustomResourceType(resourceType)
	if err != nil {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 列出自定义资源
	// 解析查询参数
	q := model.CustomResourceQuery{
		ResourceType: resourceType,
		Filter:       c.Query("filter"),
		SortBy:       c.Query("sortBy"),
		SortOrder:    c.Query("sortOrder"),
	}

	// 解析分页参数
	startIndex, _ := strconv.Atoi(c.DefaultQuery("startIndex", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))
	q.StartIndex = startIndex
	q.Count = count

	// 列出自定义资源
	resources, total, err := reg.store.ListCustomResources(&q)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 获取完整的基础URL
	baseURL := getBaseURL(c)
	// 设置完整的Meta.Location
	for i := range resources {
		cr := &resources[i]
		cr.Meta.Location = baseURL + reg.cfg.APIPath + "/" + resourceType + "/" + cr.ID
	}

	c.JSON(http.StatusOK, model.ListResponse{
		Schemas:      []string{model.ListSchema.String()},
		TotalResults: int(total),
		StartIndex:   q.StartIndex,
		ItemsPerPage: q.Count,
		Resources:    resources,
	})
}

// @Summary 创建自定义资源
// @Description 创建新的自定义资源
// @Tags CustomResources
// @Accept json
// @Produce json
// @Param resourceType path string true "资源类型"
// @Param customResource body model.CustomResource true "自定义资源信息"
// @Success 201 {object} model.CustomResource
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /scim/v2/{resourceType} [post]
func (reg *RouteRegistrar) createCustomResource(c *gin.Context) {
	resourceType := c.Param("resourceType")
	if resourceType == "" {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 验证资源类型是否存在
	_, err := reg.store.GetCustomResourceType(resourceType)
	if err != nil {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 创建自定义资源
	var cr model.CustomResource
	if err := c.ShouldBindJSON(&cr); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidRequest")
		return
	}

	// 设置资源类型
	cr.ResourceType = resourceType

	// 生成ID（如果未提供）
	if cr.ID == "" {
		cr.ID = util.GenerateID()
	}

	// 创建自定义资源
	err = reg.store.CreateCustomResource(&cr)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 获取完整的基础URL
	baseURL := getBaseURL(c)
	// 设置完整的Meta.Location
	cr.Meta.Location = baseURL + reg.cfg.APIPath + "/" + resourceType + "/" + cr.ID

	c.JSON(http.StatusCreated, cr)
}

// @Summary 获取自定义资源
// @Description 根据ID获取自定义资源的详细信息
// @Tags CustomResources
// @Accept json
// @Produce json
// @Param resourceType path string true "资源类型"
// @Param resourceID path string true "资源ID"
// @Success 200 {object} model.CustomResource
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /scim/v2/{resourceType}/{resourceID} [get]
func (reg *RouteRegistrar) getCustomResource(c *gin.Context) {
	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceID")

	if resourceType == "" || resourceID == "" {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 验证资源类型是否存在
	_, err := reg.store.GetCustomResourceType(resourceType)
	if err != nil {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 获取单个自定义资源
	cr, err := reg.store.GetCustomResource(resourceID, resourceType)
	if err != nil {
		ErrorHandler(c, err, http.StatusNotFound, "notFound")
		return
	}

	// 获取完整的基础URL
	baseURL := getBaseURL(c)
	// 设置完整的Meta.Location
	cr.Meta.Location = baseURL + reg.cfg.APIPath + "/" + resourceType + "/" + cr.ID

	c.JSON(http.StatusOK, cr)
}

// @Summary 更新自定义资源
// @Description 更新现有的自定义资源信息
// @Tags CustomResources
// @Accept json
// @Produce json
// @Param resourceType path string true "资源类型"
// @Param resourceID path string true "资源ID"
// @Param customResource body model.CustomResource true "自定义资源信息"
// @Success 200 {object} model.CustomResource
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /scim/v2/{resourceType}/{resourceID} [put]
func (reg *RouteRegistrar) updateCustomResource(c *gin.Context) {
	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceID")

	if resourceType == "" || resourceID == "" {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 验证资源类型是否存在
	_, err := reg.store.GetCustomResourceType(resourceType)
	if err != nil {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 更新自定义资源
	var cr model.CustomResource
	if err := c.ShouldBindJSON(&cr); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidRequest")
		return
	}

	// 确保ID和资源类型匹配
	if cr.ID != resourceID || cr.ResourceType != resourceType {
		ErrorHandler(c, model.ErrInvalidValue, http.StatusBadRequest, "invalidValue")
		return
	}

	// 更新自定义资源
	err = reg.store.UpdateCustomResource(&cr)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 获取完整的基础URL
	baseURL := getBaseURL(c)
	// 设置完整的Meta.Location
	cr.Meta.Location = baseURL + reg.cfg.APIPath + "/" + resourceType + "/" + cr.ID

	c.JSON(http.StatusOK, cr)
}

// @Summary 补丁更新自定义资源
// @Description 使用补丁操作更新自定义资源的部分属性
// @Tags CustomResources
// @Accept json
// @Produce json
// @Param resourceType path string true "资源类型"
// @Param resourceID path string true "资源ID"
// @Param patch body []model.PatchOperation true "补丁操作"
// @Success 200 {object} model.CustomResource
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /scim/v2/{resourceType}/{resourceID} [patch]
func (reg *RouteRegistrar) patchCustomResource(c *gin.Context) {
	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceID")

	if resourceType == "" || resourceID == "" {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 验证资源类型是否存在
	_, err := reg.store.GetCustomResourceType(resourceType)
	if err != nil {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 补丁更新自定义资源
	var ops []model.PatchOperation
	if err := c.ShouldBindJSON(&ops); err != nil {
		ErrorHandler(c, err, http.StatusBadRequest, "invalidRequest")
		return
	}

	// 补丁更新自定义资源
	err = reg.store.PatchCustomResource(resourceID, resourceType, ops)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	// 获取更新后的资源
	cr, err := reg.store.GetCustomResource(resourceID, resourceType)
	if err != nil {
		ErrorHandler(c, err, http.StatusNotFound, "notFound")
		return
	}

	// 获取完整的基础URL
	baseURL := getBaseURL(c)
	// 设置完整的Meta.Location
	cr.Meta.Location = baseURL + reg.cfg.APIPath + "/" + resourceType + "/" + cr.ID

	c.JSON(http.StatusOK, cr)
}

// @Summary 删除自定义资源
// @Description 根据ID删除自定义资源
// @Tags CustomResources
// @Accept json
// @Produce json
// @Param resourceType path string true "资源类型"
// @Param resourceID path string true "资源ID"
// @Success 204 {object} model.CustomResource
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /scim/v2/{resourceType}/{resourceID} [delete]
func (reg *RouteRegistrar) deleteCustomResource(c *gin.Context) {
	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceID")

	if resourceType == "" || resourceID == "" {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 验证资源类型是否存在
	_, err := reg.store.GetCustomResourceType(resourceType)
	if err != nil {
		ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		return
	}

	// 删除自定义资源
	err = reg.store.DeleteCustomResource(resourceID, resourceType)
	if err != nil {
		ErrorHandler(c, err, http.StatusInternalServerError, "internalError")
		return
	}

	c.Status(http.StatusNoContent)
}
