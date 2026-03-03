package api

import (
	"net/http"
	"scim-go/model"
	"scim-go/store"

	_ "scim-go/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// ScimConfig SCIM配置结构体（从主配置注入）
type ScimConfig struct {
	DefaultSchema string
	GroupSchema   string
	ErrorSchema   string
	ListSchema    string
	MaxCount      int
	DefaultCount  int
}

// RegisterRoutes 注册所有SCIM接口
func RegisterRoutes(r *gin.Engine, s store.Store, cfg *ScimConfig, authToken string) {
	// Swagger API 文档（需要认证）
	swaggerGroup := r.Group("/swagger")
	swaggerGroup.Use(Auth(authToken))
	swaggerGroup.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 健康检查（无需认证）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "scim-server", "version": "1.0.0"})
	})

	// SCIM根路径（服务发现，无需认证）
	r.GET("/scim/v2", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"schemas":    []string{"urn:ietf:params:scim:api:messages:2.0:ServiceProviderConfig"},
			"patch":      gin.H{"supported": true},
			"filter":     gin.H{"supported": true, "maxResults": cfg.MaxCount},
			"sort":       gin.H{"supported": true},
			"pagination": gin.H{"supported": true, "maxItemsPerPage": cfg.MaxCount},
			"resources": gin.H{
				"User":  gin.H{"endpoint": "/scim/v2/Users"},
				"Group": gin.H{"endpoint": "/scim/v2/Groups"},
			},
		})
	})

	// SCIM核心组（需要认证+参数绑定+分页）
	scimGroup := r.Group("/scim/v2")
	scimGroup.Use(Auth(authToken))
	scimGroup.Use(BindQuery())
	scimGroup.Use(Pagination(cfg.DefaultCount, cfg.MaxCount))

	// ServiceProviderConfig 端点（无需认证，符合 RFC 7644）
	r.GET("/scim/v2/ServiceProviderConfig", func(c *gin.Context) {
		config := model.GetServiceProviderConfig(cfg.MaxCount)
		c.JSON(http.StatusOK, config)
	})

	// ResourceTypes 端点（需要认证）
	scimGroup.GET("/ResourceTypes", func(c *gin.Context) {
		resourceTypes := []model.ResourceType{
			*model.GetUserResourceType(),
			*model.GetGroupResourceType(),
		}
		c.JSON(http.StatusOK, model.ListResponse{
			Schemas:      []string{model.ListSchema},
			TotalResults: len(resourceTypes),
			Resources:    resourceTypes,
		})
	})

	scimGroup.GET("/ResourceTypes/:id", func(c *gin.Context) {
		id := c.Param("id")
		switch id {
		case "User":
			c.JSON(http.StatusOK, model.GetUserResourceType())
		case "Group":
			c.JSON(http.StatusOK, model.GetGroupResourceType())
		default:
			ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		}
	})

	// Schemas 端点（需要认证）
	scimGroup.GET("/Schemas", func(c *gin.Context) {
		schemas := []model.Schema{
			*model.GetUserSchema(),
			*model.GetGroupSchema(),
		}
		c.JSON(http.StatusOK, model.ListResponse{
			Schemas:      []string{model.ListSchema},
			TotalResults: len(schemas),
			Resources:    schemas,
		})
	})

	scimGroup.GET("/Schemas/:id", func(c *gin.Context) {
		id := c.Param("id")
		switch id {
		case model.UserSchema:
			c.JSON(http.StatusOK, model.GetUserSchema())
		case model.GroupSchema:
			c.JSON(http.StatusOK, model.GetGroupSchema())
		default:
			ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		}
	})

	// 初始化处理器
	userHandler := NewUserHandlers(s, &model.ScimConfig{
		DefaultSchema: cfg.DefaultSchema,
		GroupSchema:   cfg.GroupSchema,
		ErrorSchema:   cfg.ErrorSchema,
		ListSchema:    cfg.ListSchema,
	})
	groupHandler := NewGroupHandlers(s, &model.ScimConfig{
		DefaultSchema: cfg.DefaultSchema,
		GroupSchema:   cfg.GroupSchema,
		ErrorSchema:   cfg.ErrorSchema,
		ListSchema:    cfg.ListSchema,
	})

	// 注册User接口（全方法：GET/POST/PUT/PATCH/DELETE）
	userGroup := scimGroup.Group("/Users")
	userGroup.GET("", userHandler.ListUsers)
	userGroup.GET("/:id", userHandler.GetUser)
	userGroup.POST("", userHandler.CreateUser)
	userGroup.PUT("/:id", userHandler.UpdateUser)
	userGroup.PATCH("/:id", userHandler.PatchUser)
	userGroup.DELETE("/:id", userHandler.DeleteUser)

	// 注册Group接口（全方法：GET/POST/PUT/PATCH/DELETE）
	groupGroup := scimGroup.Group("/Groups")
	groupGroup.GET("", groupHandler.ListGroups)
	groupGroup.GET("/:id", groupHandler.GetGroup)
	groupGroup.POST("", groupHandler.CreateGroup)
	groupGroup.PUT("/:id", groupHandler.UpdateGroup)
	groupGroup.PATCH("/:id", groupHandler.PatchGroup)
	groupGroup.DELETE("/:id", groupHandler.DeleteGroup)

	// 注册Group成员管理接口
	groupGroup.POST("/:id/members", groupHandler.AddUserToGroup)
	groupGroup.DELETE("/:id/members/:userId", groupHandler.RemoveUserFromGroup)
}
