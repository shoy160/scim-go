package model

// ResourceType SCIM ResourceType 定义 (RFC 7643)
type ResourceType struct {
	Schemas          []string          `json:"schemas"`
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Endpoint         string            `json:"endpoint"`
	Description      string            `json:"description,omitempty"`
	Schema           string            `json:"schema"`
	SchemaExtensions []SchemaExtension `json:"schemaExtensions,omitempty"`
}

// SchemaExtension 模式扩展定义
type SchemaExtension struct {
	Schema   string `json:"schema"`
	Required bool   `json:"required"`
}

// SchemaAttribute SCIM 模式属性定义
type SchemaAttribute struct {
	Name            string            `json:"name"`
	Type            string            `json:"type"` // string, boolean, decimal, integer, dateTime, reference, complex
	SubAttributes   []SchemaAttribute `json:"subAttributes,omitempty"`
	MultiValued     bool              `json:"multiValued"`
	Description     string            `json:"description,omitempty"`
	Required        bool              `json:"required"`
	CanonicalValues []string          `json:"canonicalValues,omitempty"`
	CaseExact       bool              `json:"caseExact,omitempty"`
	Mutability      string            `json:"mutability,omitempty"` // readWrite, readOnly, immutable, writeOnly
	Returned        string            `json:"returned,omitempty"`   // always, never, default, request
	Uniqueness      string            `json:"uniqueness,omitempty"` // none, server, global
	ReferenceTypes  []string          `json:"referenceTypes,omitempty"`
}

// Schema SCIM Schema 定义 (RFC 7643)
type Schema struct {
	Schemas     []string          `json:"schemas"`
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Attributes  []SchemaAttribute `json:"attributes"`
}

// ServiceProviderConfig SCIM 服务提供者配置 (RFC 7643)
type ServiceProviderConfig struct {
	Schemas               []string               `json:"schemas"`
	DocumentationURI      string                 `json:"documentationUri,omitempty"`
	Patch                 Supported              `json:"patch"`
	Bulk                  BulkSupport            `json:"bulk"`
	Filter                FilterSupport          `json:"filter"`
	ChangePassword        Supported              `json:"changePassword"`
	Sort                  Supported              `json:"sort"`
	ETag                  Supported              `json:"etag"`
	AuthenticationSchemes []AuthenticationScheme `json:"authenticationSchemes"`
	Meta                  *Meta                  `json:"meta,omitempty"`
}

// Supported 功能支持状态
type Supported struct {
	Supported bool `json:"supported"`
}

// BulkSupport 批量操作支持
type BulkSupport struct {
	Supported      bool `json:"supported"`
	MaxOperations  int  `json:"maxOperations,omitempty"`
	MaxPayloadSize int  `json:"maxPayloadSize,omitempty"`
}

// FilterSupport 过滤支持
type FilterSupport struct {
	Supported  bool `json:"supported"`
	MaxResults int  `json:"maxResults,omitempty"`
}

// AuthenticationScheme 认证方案
type AuthenticationScheme struct {
	Type             string `json:"type"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	SpecURI          string `json:"specUri,omitempty"`
	DocumentationURI string `json:"documentationUri,omitempty"`
	Primary          bool   `json:"primary,omitempty"`
}

// Meta 资源元数据
type Meta struct {
	ResourceType string `json:"resourceType,omitempty"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Location     string `json:"location,omitempty"`
	Version      string `json:"version,omitempty"`
}

const (
	// ResourceTypeSchema ResourceType 的 Schema URN
	ResourceTypeSchema = "urn:ietf:params:scim:schemas:core:2.0:ResourceType"
	// SchemaDefinitionSchema Schema 定义的 Schema URN
	SchemaDefinitionSchema = "urn:ietf:params:scim:schemas:core:2.0:Schema"
	// ServiceProviderConfigSchema ServiceProviderConfig 的 Schema URN
	ServiceProviderConfigSchema = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
)

// GetUserResourceType 返回 User 资源类型定义
func GetUserResourceType() *ResourceType {
	return &ResourceType{
		Schemas:     []string{ResourceTypeSchema},
		ID:          "User",
		Name:        "User",
		Endpoint:    "/Users",
		Description: "User Account",
		Schema:      UserSchema.String(),
	}
}

// GetGroupResourceType 返回 Group 资源类型定义
func GetGroupResourceType() *ResourceType {
	return &ResourceType{
		Schemas:     []string{ResourceTypeSchema},
		ID:          "Group",
		Name:        "Group",
		Endpoint:    "/Groups",
		Description: "Group",
		Schema:      GroupSchema.String(),
	}
}

// GetUserSchema 返回 User Schema 定义
func GetUserSchema() *Schema {
	return &Schema{
		Schemas:     []string{SchemaDefinitionSchema},
		ID:          UserSchema.String(),
		Name:        "User",
		Description: "User Account",
		Attributes: []SchemaAttribute{
			{
				Name:        "userName",
				Type:        "string",
				MultiValued: false,
				Description: "Unique identifier for the User, typically used by the user to directly authenticate to the service provider.",
				Required:    true,
				CaseExact:   false,
				Mutability:  "readWrite",
				Returned:    "default",
				Uniqueness:  "server",
			},
			{
				Name:        "name",
				Type:        "complex",
				MultiValued: false,
				Description: "The components of the user's real name.",
				Required:    false,
				SubAttributes: []SchemaAttribute{
					{
						Name:        "formatted",
						Type:        "string",
						MultiValued: false,
						Description: "The full name, including all middle names, titles, and suffixes as appropriate, formatted for display.",
						Required:    false,
					},
					{
						Name:        "familyName",
						Type:        "string",
						MultiValued: false,
						Description: "The family name of the User, or last name in most Western languages.",
						Required:    true,
					},
					{
						Name:        "givenName",
						Type:        "string",
						MultiValued: false,
						Description: "The given name of the User, or first name in most Western languages.",
						Required:    true,
					},
					{
						Name:        "middleName",
						Type:        "string",
						MultiValued: false,
						Description: "The middle name(s) of the User.",
						Required:    false,
					},
				},
			},
			{
				Name:        "displayName",
				Type:        "string",
				MultiValued: false,
				Description: "The name of the User, suitable for display to end-users.",
				Required:    false,
				Mutability:  "readWrite",
				Returned:    "default",
			},
			{
				Name:        "nickName",
				Type:        "string",
				MultiValued: false,
				Description: "The casual way to address the user in real life.",
				Required:    false,
			},
			{
				Name:           "profileUrl",
				Type:           "reference",
				MultiValued:    false,
				Description:    "A fully qualified URL pointing to a page representing the User online.",
				Required:       false,
				ReferenceTypes: []string{"external"},
			},
			{
				Name:        "active",
				Type:        "boolean",
				MultiValued: false,
				Description: "A Boolean value indicating the User's administrative status.",
				Required:    false,
				Mutability:  "readWrite",
				Returned:    "default",
			},
			{
				Name:        "emails",
				Type:        "complex",
				MultiValued: true,
				Description: "Email addresses for the user.",
				Required:    false,
				SubAttributes: []SchemaAttribute{
					{
						Name:        "value",
						Type:        "string",
						MultiValued: false,
						Description: "Email addresses for the user.",
						Required:    false,
					},
					{
						Name:        "display",
						Type:        "string",
						MultiValued: false,
						Description: "A human-readable name, primarily used for display purposes.",
						Required:    false,
					},
					{
						Name:            "type",
						Type:            "string",
						MultiValued:     false,
						Description:     "A label indicating the attribute's function.",
						Required:        false,
						CanonicalValues: []string{"work", "home", "other"},
					},
					{
						Name:        "primary",
						Type:        "boolean",
						MultiValued: false,
						Description: "A Boolean value indicating the 'primary' or preferred attribute value for this attribute.",
						Required:    false,
					},
				},
			},
			{
				Name:        "roles",
				Type:        "complex",
				MultiValued: true,
				Description: "A list of roles for the User that collectively represent who the User is.",
				Required:    false,
				SubAttributes: []SchemaAttribute{
					{
						Name:        "value",
						Type:        "string",
						MultiValued: false,
						Description: "The value of a role.",
						Required:    false,
					},
					{
						Name:        "display",
						Type:        "string",
						MultiValued: false,
						Description: "A human-readable name, primarily used for display purposes.",
						Required:    false,
					},
					{
						Name:        "type",
						Type:        "string",
						MultiValued: false,
						Description: "A label indicating the attribute's function.",
						Required:    false,
					},
					{
						Name:        "primary",
						Type:        "boolean",
						MultiValued: false,
						Description: "A Boolean value indicating the 'primary' or preferred attribute value for this attribute.",
						Required:    false,
					},
				},
			},
			{
				Name:        "externalId",
				Type:        "string",
				MultiValued: false,
				Description: "A identifier of the User's identity from an external identity provider.",
				Required:    false,
				CaseExact:   true,
				Mutability:  "readWrite",
				Returned:    "default",
				Uniqueness:  "none",
			},
		},
	}
}

// GetGroupSchema 返回 Group Schema 定义
func GetGroupSchema() *Schema {
	return &Schema{
		Schemas:     []string{SchemaDefinitionSchema},
		ID:          GroupSchema.String(),
		Name:        "Group",
		Description: "Group",
		Attributes: []SchemaAttribute{
			{
				Name:        "displayName",
				Type:        "string",
				MultiValued: false,
				Description: "A human-readable name for the Group.",
				Required:    true,
				CaseExact:   false,
				Mutability:  "readWrite",
				Returned:    "default",
				Uniqueness:  "server",
			},
			{
				Name:        "members",
				Type:        "complex",
				MultiValued: true,
				Description: "A list of members of the Group.",
				Required:    false,
				SubAttributes: []SchemaAttribute{
					{
						Name:        "value",
						Type:        "string",
						MultiValued: false,
						Description: "Identifier of the member of this Group.",
						Required:    false,
					},
					{
						Name:           "$ref",
						Type:           "reference",
						MultiValued:    false,
						Description:    "The URI corresponding to a SCIM resource that is a member of this Group.",
						Required:       false,
						ReferenceTypes: []string{"User", "Group"},
					},
					{
						Name:        "display",
						Type:        "string",
						MultiValued: false,
						Description: "A human-readable name, primarily used for display purposes.",
						Required:    false,
					},
					{
						Name:            "type",
						Type:            "string",
						MultiValued:     false,
						Description:     "A label indicating the type of resource.",
						Required:        false,
						CanonicalValues: []string{"User", "Group"},
					},
				},
			},
			{
				Name:        "externalId",
				Type:        "string",
				MultiValued: false,
				Description: "A identifier of the Group's external identity.",
				Required:    false,
				CaseExact:   true,
				Mutability:  "readWrite",
				Returned:    "default",
				Uniqueness:  "none",
			},
		},
	}
}

// GetServiceProviderConfig 返回服务提供者配置
func GetServiceProviderConfig(maxResults int) *ServiceProviderConfig {
	return &ServiceProviderConfig{
		Schemas: []string{ServiceProviderConfigSchema},
		Patch: Supported{
			Supported: true,
		},
		Bulk: BulkSupport{
			Supported:      false,
			MaxOperations:  0,
			MaxPayloadSize: 0,
		},
		Filter: FilterSupport{
			Supported:  true,
			MaxResults: maxResults,
		},
		ChangePassword: Supported{
			Supported: false,
		},
		Sort: Supported{
			Supported: true,
		},
		ETag: Supported{
			Supported: false,
		},
		AuthenticationSchemes: []AuthenticationScheme{
			{
				Type:        "oauthbearertoken",
				Name:        "OAuth Bearer Token",
				Description: "Authentication scheme using the OAuth Bearer Token Standard",
				SpecURI:     "https://www.rfc-editor.org/rfc/rfc6750.html",
				Primary:     true,
			},
		},
	}
}
