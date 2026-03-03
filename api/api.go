package api

import (
	"net/http"
	"scim-go/model"
	"scim-go/store"

	_ "scim-go/docs"

	"github.com/gin-contrib/cors"
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

// RouteRegistrar 路由注册器
// 用于模块化注册不同类型的路由
type RouteRegistrar struct {
	r              *gin.Engine
	store          store.Store
	cfg            *ScimConfig
	authToken      string
	swaggerEnabled bool
	swaggerPath    string
}

// NewRouteRegistrar 创建新的路由注册器
func NewRouteRegistrar(r *gin.Engine, s store.Store, cfg *ScimConfig, authToken string, swaggerEnabled bool, swaggerPath string) *RouteRegistrar {
	return &RouteRegistrar{
		r:              r,
		store:          s,
		cfg:            cfg,
		authToken:      authToken,
		swaggerEnabled: swaggerEnabled,
		swaggerPath:    swaggerPath,
	}
}

// RegisterRoutes 注册所有SCIM接口
// 主入口函数，协调所有路由的注册
func RegisterRoutes(r *gin.Engine, s store.Store, cfg *ScimConfig, authToken string, swaggerEnabled bool, swaggerPath string) {
	registrar := NewRouteRegistrar(r, s, cfg, authToken, swaggerEnabled, swaggerPath)

	// 注册全局中间件
	registrar.registerGlobalMiddlewares()

	// 注册Swagger文档路由
	registrar.registerSwaggerRoutes()

	// 注册健康检查路由
	registrar.registerHealthRoutes()

	// 注册服务发现路由
	registrar.registerDiscoveryRoutes()

	// 注册SCIM核心路由
	registrar.registerScimRoutes()
}

// registerGlobalMiddlewares 注册全局中间件
func (reg *RouteRegistrar) registerGlobalMiddlewares() {
	// 配置CORS中间件
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	corsConfig.AllowCredentials = true
	reg.r.Use(cors.New(corsConfig))

	// 注册性能监控中间件
	reg.r.Use(PerformanceMonitor())

	// 注册恢复中间件
	reg.r.Use(Recovery())
}

// registerSwaggerRoutes 注册Swagger文档路由
func (reg *RouteRegistrar) registerSwaggerRoutes() {
	if !reg.swaggerEnabled {
		return
	}

	// 处理所有 Swagger 相关路径
	reg.r.GET(reg.swaggerPath+"/*any", func(c *gin.Context) {
		// 获取路径参数
		path := c.Param("any")
		// 如果路径为空或只是斜杠，重定向到 index.html
		if path == "" || path == "/" {
			c.Redirect(http.StatusMovedPermanently, reg.swaggerPath+"/index.html")
			return
		}
		// 否则使用 Swagger 处理器
		ginSwagger.WrapHandler(swaggerFiles.Handler)(c)
	})

	// 处理根路径请求（如 /swagger）
	reg.r.GET(reg.swaggerPath, func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, reg.swaggerPath+"/index.html")
	})
}

// registerHealthRoutes 注册健康检查路由
func (reg *RouteRegistrar) registerHealthRoutes() {
	// 健康检查（无需认证）
	reg.r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "scim-server",
			"version": "1.0.0",
		})
	})

	// 就绪检查（无需认证）
	reg.r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	})
}

// registerDiscoveryRoutes 注册服务发现路由
func (reg *RouteRegistrar) registerDiscoveryRoutes() {
	// SCIM根路径（服务发现，无需认证）
	reg.r.GET("/scim/v2", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:ServiceProviderConfig"},
			"patch":   gin.H{"supported": true},
			"filter":  gin.H{"supported": true, "maxResults": reg.cfg.MaxCount},
			"sort":    gin.H{"supported": true},
			"pagination": gin.H{
				"supported":       true,
				"maxItemsPerPage": reg.cfg.MaxCount,
			},
			"resources": gin.H{
				"User":  gin.H{"endpoint": "/scim/v2/Users"},
				"Group": gin.H{"endpoint": "/scim/v2/Groups"},
			},
		})
	})
}

// registerScimRoutes 注册SCIM核心路由
func (reg *RouteRegistrar) registerScimRoutes() {
	// ServiceProviderConfig 端点（无需认证，符合 RFC 7644）
	reg.r.GET("/scim/v2/ServiceProviderConfig", func(c *gin.Context) {
		config := model.GetServiceProviderConfig(reg.cfg.MaxCount)
		c.JSON(http.StatusOK, config)
	})

	// SCIM核心组（需要认证+参数绑定+分页）
	scimGroup := reg.r.Group("/scim/v2")
	scimGroup.Use(Auth(reg.authToken))
	scimGroup.Use(BindQuery())
	scimGroup.Use(Pagination(reg.cfg.DefaultCount, reg.cfg.MaxCount))

	// 注册ResourceTypes路由
	reg.registerResourceTypesRoutes(scimGroup)

	// 注册Schemas路由
	reg.registerSchemasRoutes(scimGroup)

	// 注册User和Group路由
	reg.registerUserRoutes(scimGroup)
	reg.registerGroupRoutes(scimGroup)
}

// registerResourceTypesRoutes 注册ResourceTypes路由
func (reg *RouteRegistrar) registerResourceTypesRoutes(scimGroup *gin.RouterGroup) {
	// ResourceTypes 端点（需要认证）
	scimGroup.GET("/ResourceTypes", func(c *gin.Context) {
		resourceTypes := []model.ResourceType{
			*model.GetUserResourceType(),
			*model.GetGroupResourceType(),
		}
		c.JSON(http.StatusOK, model.ListResponse{
			Schemas:      []string{model.ListSchema.String()},
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
}

// registerSchemasRoutes 注册Schemas路由
func (reg *RouteRegistrar) registerSchemasRoutes(scimGroup *gin.RouterGroup) {
	// Schemas 端点（需要认证）
	scimGroup.GET("/Schemas", func(c *gin.Context) {
		schemas := []model.Schema{
			*model.GetUserSchema(),
			*model.GetGroupSchema(),
		}
		c.JSON(http.StatusOK, model.ListResponse{
			Schemas:      []string{model.ListSchema.String()},
			TotalResults: len(schemas),
			Resources:    schemas,
		})
	})

	scimGroup.GET("/Schemas/:id", func(c *gin.Context) {
		id := c.Param("id")
		switch id {
		case model.UserSchema.String():
			c.JSON(http.StatusOK, model.GetUserSchema())
		case model.GroupSchema.String():
			c.JSON(http.StatusOK, model.GetGroupSchema())
		default:
			ErrorHandler(c, model.ErrNotFound, http.StatusNotFound, "notFound")
		}
	})
}

// registerUserRoutes 注册User路由
func (reg *RouteRegistrar) registerUserRoutes(scimGroup *gin.RouterGroup) {
	// 初始化用户处理器
	userHandler := NewUserHandlers(reg.store, &model.ScimConfig{
		DefaultSchema: reg.cfg.DefaultSchema,
		GroupSchema:   reg.cfg.GroupSchema,
		ErrorSchema:   reg.cfg.ErrorSchema,
		ListSchema:    reg.cfg.ListSchema,
	})

	// 注册User接口（全方法：GET/POST/PUT/PATCH/DELETE）
	userGroup := scimGroup.Group("/Users")
	userGroup.GET("", userHandler.ListUsers)
	userGroup.GET("/:id", userHandler.GetUser)
	userGroup.POST("", userHandler.CreateUser)
	userGroup.PUT("/:id", userHandler.UpdateUser)
	userGroup.PATCH("/:id", userHandler.PatchUser)
	userGroup.DELETE("/:id", userHandler.DeleteUser)
}

// registerGroupRoutes 注册Group路由
func (reg *RouteRegistrar) registerGroupRoutes(scimGroup *gin.RouterGroup) {
	// 初始化组处理器
	groupHandler := NewGroupHandlers(reg.store, &model.ScimConfig{
		DefaultSchema: reg.cfg.DefaultSchema,
		GroupSchema:   reg.cfg.GroupSchema,
		ErrorSchema:   reg.cfg.ErrorSchema,
		ListSchema:    reg.cfg.ListSchema,
	})

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
