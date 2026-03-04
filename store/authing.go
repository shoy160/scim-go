package store

import (
	"encoding/json"
	"fmt"
	"net/http"
	"scim-go/model"
	"scim-go/util"
	"strings"
	"time"
)

// AuthingStore Authing 存储实现
type AuthingStore struct {
	host         string
	userPoolID   string
	accessKey    string
	accessSecret string
	httpClient   *http.Client
}

// NewAuthingStore 创建 Authing 存储实例
func NewAuthingStore(host, userPoolID, accessKey, accessSecret string) *AuthingStore {
	if host == "" {
		host = "https://core.authing.cn"
	}

	return &AuthingStore{
		host:         host,
		userPoolID:   userPoolID,
		accessKey:    accessKey,
		accessSecret: accessSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// ---------------------- User 相关 ----------------------

// CreateUser 创建用户
func (s *AuthingStore) CreateUser(u *model.User) error {
	// 调用 Authing API 创建用户
	url := s.host + "/api/v3/create-user"
	headers := s.getAuthHeaders()

	// 构建 Authing 用户请求
	status := "Activated"
	if !u.Active {
		status = "Deactivated"
	}
	authUser := map[string]interface{}{
		"username":    u.UserName,
		"email":       getPrimaryEmail(u.Emails),
		"name":        u.Name.GivenName + " " + u.Name.FamilyName,
		"givenName":   u.Name.GivenName,
		"familyName":  u.Name.FamilyName,
		"displayName": u.DisplayName,
		"nickname":    u.NickName,
		"profile":     u.ProfileUrl,
		"status":      status,
	}

	// 添加自定义属性
	if len(u.Emails) > 0 {
		authUser["emails"] = u.Emails
	}

	// 发送请求
	resp, err := s.sendRequest("POST", url, headers, authUser)
	if err != nil {
		return err
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取用户 ID
	if userID, ok := result["id"].(string); ok {
		u.ID = userID
		// 生成 meta 属性
		now := time.Now()
		u.Meta = model.Meta{
			ResourceType: "User",
			Created:      now.Format(time.RFC3339),
			LastModified: now.Format(time.RFC3339),
			Location:     "", // 由API层动态生成
			Version:      util.GenerateVersion(),
		}
	} else {
		return fmt.Errorf("未找到用户 ID")
	}

	return nil
}

// GetUser 获取用户
func (s *AuthingStore) GetUser(id string) (*model.User, error) {
	url := s.host + "/api/v3/get-user?userId=" + id
	headers := s.getAuthHeaders()

	resp, err := s.sendRequest("GET", url, headers, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 转换为 SCIM 用户模型
	// 生成 meta 属性
	baseURL := "https://api.scim.dev/scim/v2"
	now := time.Now()
	meta := util.GenerateUserMeta(baseURL, result["id"].(string), now, now)

	user := &model.User{
		ID:          result["id"].(string),
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		UserName:    result["username"].(string),
		Active:      result["status"].(string) == "activated",
		DisplayName: getStringValue(result, "displayName"),
		NickName:    getStringValue(result, "nickname"),
		ProfileUrl:  getStringValue(result, "profile"),
		Meta:        meta,
	}

	// 姓名信息
	if name, ok := result["name"].(map[string]interface{}); ok {
		user.Name.GivenName = getStringValue(name, "givenName")
		user.Name.FamilyName = getStringValue(name, "familyName")
		user.Name.MiddleName = getStringValue(name, "middleName")
	} else {
		// 兼容直接存储的姓名
		user.Name.GivenName = getStringValue(result, "givenName")
		user.Name.FamilyName = getStringValue(result, "familyName")
	}

	// 邮箱信息
	if emails, ok := result["emails"].([]interface{}); ok {
		user.Emails = make([]model.Email, 0, len(emails))
		for _, email := range emails {
			if emailMap, ok := email.(map[string]interface{}); ok {
				user.Emails = append(user.Emails, model.Email{
					UserID:  user.ID,
					Value:   emailMap["value"].(string),
					Type:    getStringValue(emailMap, "type"),
					Primary: getBoolValue(emailMap, "primary"),
				})
			}
		}
	}

	// 获取用户所属组
	groups, err := s.GetUserGroups(user.ID)
	if err == nil {
		user.Groups = groups
	}

	return user, nil
}

// ListUsers 列出用户
func (s *AuthingStore) ListUsers(q *model.ResourceQuery) ([]model.User, int64, error) {
	url := s.host + "/api/v3/list-users"
	headers := s.getAuthHeaders()

	// 构建查询参数
	params := map[string]interface{}{
		"page":      q.StartIndex,
		"limit":     q.Count,
		"sortBy":    q.SortBy,
		"sortOrder": q.SortOrder,
	}

	// 添加过滤条件
	if q.Filter != "" {
		params["filter"] = q.Filter
	}

	resp, err := s.sendRequest("GET", url, headers, params)
	if err != nil {
		return nil, 0, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, 0, fmt.Errorf("解析响应失败: %v", err)
	}

	// 解析用户列表
	users := []model.User{}
	if items, ok := result["list"].([]interface{}); ok {
		for _, item := range items {
			if userMap, ok := item.(map[string]interface{}); ok {
				user := model.User{
					ID:          userMap["id"].(string),
					Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
					UserName:    userMap["username"].(string),
					Active:      userMap["status"].(string) == "Activated",
					DisplayName: getStringValue(userMap, "displayName"),
					NickName:    getStringValue(userMap, "nickname"),
					ProfileUrl:  getStringValue(userMap, "profile"),
				}

				// 姓名信息
				if name, ok := userMap["name"].(map[string]interface{}); ok {
					user.Name.GivenName = getStringValue(name, "givenName")
					user.Name.FamilyName = getStringValue(name, "familyName")
				} else {
					user.Name.GivenName = getStringValue(userMap, "givenName")
					user.Name.FamilyName = getStringValue(userMap, "familyName")
				}

				users = append(users, user)
			}
		}
	}

	// 总用户数
	total := int64(0)
	if totalCount, ok := result["total"].(float64); ok {
		total = int64(totalCount)
	}

	return users, total, nil
}

// UpdateUser 更新用户
func (s *AuthingStore) UpdateUser(u *model.User) error {
	url := s.host + "/users/" + u.ID
	headers := s.getAuthHeaders()

	// 构建更新请求
	updateData := map[string]interface{}{
		"username":    u.UserName,
		"displayName": u.DisplayName,
		"nickname":    u.NickName,
		"profile":     u.ProfileUrl,
		"name": map[string]interface{}{
			"givenName":  u.Name.GivenName,
			"familyName": u.Name.FamilyName,
			"middleName": u.Name.MiddleName,
		},
		"status": map[string]interface{}{"activated": u.Active},
	}

	// 更新邮箱
	if len(u.Emails) > 0 {
		updateData["emails"] = u.Emails
	}

	_, err := s.sendRequest("PUT", url, headers, updateData)
	if err != nil {
		return err
	}

	// 更新 meta 属性
	baseURL := "https://api.scim.dev/scim/v2"
	util.UpdateMeta(&u.Meta, baseURL, u.ID, "User")

	return nil
}

// PatchUser 补丁更新用户
func (s *AuthingStore) PatchUser(id string, ops []model.PatchOperation) error {
	// 获取当前用户
	user, err := s.GetUser(id)
	if err != nil {
		return err
	}

	// 应用补丁操作
	err = PatchResource(s, id, user, ops)
	if err != nil {
		return err
	}

	// 保存更新
	return s.UpdateUser(user)
}

// DeleteUser 删除用户
func (s *AuthingStore) DeleteUser(id string) error {
	url := s.host + "/users/" + id
	headers := s.getAuthHeaders()

	_, err := s.sendRequest("DELETE", url, headers, nil)
	return err
}

// ---------------------- Group 相关 ----------------------

// CreateGroup 创建组
func (s *AuthingStore) CreateGroup(g *model.Group) error {
	url := s.host + "/groups"
	headers := s.getAuthHeaders()

	// 构建组请求
	groupData := map[string]interface{}{
		"name": g.DisplayName,
	}

	resp, err := s.sendRequest("POST", url, headers, groupData)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	if groupID, ok := result["id"].(string); ok {
		g.ID = groupID
		// 生成 meta 属性
		baseURL := "https://api.scim.dev/scim/v2"
		now := time.Now()
		g.Meta = util.GenerateGroupMeta(baseURL, groupID, now, now)
	} else {
		return fmt.Errorf("未找到组 ID")
	}

	// 添加成员
	for _, member := range g.Members {
		err := s.AddUserToGroup(g.ID, member.Value)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetGroup 获取组
func (s *AuthingStore) GetGroup(id string, preloadMembers bool) (*model.Group, error) {
	url := s.host + "/groups/" + id
	headers := s.getAuthHeaders()

	resp, err := s.sendRequest("GET", url, headers, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 生成 meta 属性
	baseURL := "https://api.scim.dev/scim/v2"
	now := time.Now()
	meta := util.GenerateGroupMeta(baseURL, result["id"].(string), now, now)

	group := &model.Group{
		ID:          result["id"].(string),
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		DisplayName: result["name"].(string),
		Meta:        meta,
	}

	// 加载成员
	if preloadMembers {
		membersURL := s.host + "/groups/" + id + "/members"
		membersResp, err := s.sendRequest("GET", membersURL, headers, nil)
		if err == nil {
			var membersResult map[string]interface{}
			if err := json.Unmarshal(membersResp, &membersResult); err == nil {
				if members, ok := membersResult["list"].([]interface{}); ok {
					group.Members = make([]model.Member, 0, len(members))
					for _, member := range members {
						if memberMap, ok := member.(map[string]interface{}); ok {
							group.Members = append(group.Members, model.Member{
								Value:   memberMap["id"].(string),
								Display: getStringValue(memberMap, "displayName"),
							})
						}
					}
				}
			}
		}
	}

	return group, nil
}

// ListGroups 列出组
func (s *AuthingStore) ListGroups(q *model.ResourceQuery, preloadMembers bool) ([]model.Group, int64, error) {
	url := s.host + "/groups"
	headers := s.getAuthHeaders()

	// 构建查询参数
	params := map[string]interface{}{
		"page":      q.StartIndex,
		"limit":     q.Count,
		"sortBy":    q.SortBy,
		"sortOrder": q.SortOrder,
	}

	resp, err := s.sendRequest("GET", url, headers, params)
	if err != nil {
		return nil, 0, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, 0, fmt.Errorf("解析响应失败: %v", err)
	}

	groups := []model.Group{}
	if items, ok := result["list"].([]interface{}); ok {
		for _, item := range items {
			if groupMap, ok := item.(map[string]interface{}); ok {
				group := model.Group{
					ID:          groupMap["id"].(string),
					Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
					DisplayName: groupMap["name"].(string),
				}

				// 加载成员
				if preloadMembers {
					membersURL := s.host + "/groups/" + group.ID + "/members"
					membersResp, err := s.sendRequest("GET", membersURL, headers, nil)
					if err == nil {
						var membersResult map[string]interface{}
						if err := json.Unmarshal(membersResp, &membersResult); err == nil {
							if members, ok := membersResult["list"].([]interface{}); ok {
								group.Members = make([]model.Member, 0, len(members))
								for _, member := range members {
									if memberMap, ok := member.(map[string]interface{}); ok {
										group.Members = append(group.Members, model.Member{
											Value:   memberMap["id"].(string),
											Display: getStringValue(memberMap, "displayName"),
										})
									}
								}
							}
						}
					}
				}

				groups = append(groups, group)
			}
		}
	}

	total := int64(0)
	if totalCount, ok := result["total"].(float64); ok {
		total = int64(totalCount)
	}

	return groups, total, nil
}

// UpdateGroup 更新组
func (s *AuthingStore) UpdateGroup(g *model.Group) error {
	url := s.host + "/groups/" + g.ID
	headers := s.getAuthHeaders()

	updateData := map[string]interface{}{
		"name": g.DisplayName,
	}

	_, err := s.sendRequest("PUT", url, headers, updateData)
	if err != nil {
		return err
	}

	// 先清空所有成员
	clearMembersURL := s.host + "/groups/" + g.ID + "/members"
	_, err = s.sendRequest("DELETE", clearMembersURL, headers, nil)
	if err != nil {
		return err
	}

	// 添加新成员
	for _, member := range g.Members {
		err := s.AddUserToGroup(g.ID, member.Value)
		if err != nil {
			return err
		}
	}

	// 更新 meta 属性
	baseURL := "https://api.scim.dev/scim/v2"
	util.UpdateMeta(&g.Meta, baseURL, g.ID, "Group")

	return nil
}

// PatchGroup 补丁更新组
func (s *AuthingStore) PatchGroup(id string, ops []model.PatchOperation) error {
	// 获取当前组
	group, err := s.GetGroup(id, true)
	if err != nil {
		return err
	}

	// 应用补丁操作
	err = PatchResource(s, id, group, ops)
	if err != nil {
		return err
	}

	// 保存更新
	return s.UpdateGroup(group)
}

// DeleteGroup 删除组
func (s *AuthingStore) DeleteGroup(id string) error {
	url := s.host + "/groups/" + id
	headers := s.getAuthHeaders()

	_, err := s.sendRequest("DELETE", url, headers, nil)
	return err
}

// ---------------------- Group 成员管理 ----------------------

// AddMemberToGroup 添加成员到组（支持用户和组）
func (s *AuthingStore) AddMemberToGroup(groupID, memberID, memberType string) error {
	// Authing API 目前只支持添加用户到组
	// 对于组类型的成员，暂时返回错误
	if memberType != "User" {
		return model.ErrInvalidValue
	}

	url := s.host + "/groups/" + groupID + "/members"
	headers := s.getAuthHeaders()

	addData := map[string]interface{}{
		"userId": memberID,
	}

	_, err := s.sendRequest("POST", url, headers, addData)
	return err
}

// AddUserToGroup 添加用户到组
func (s *AuthingStore) AddUserToGroup(groupID, userID string) error {
	return s.AddMemberToGroup(groupID, userID, "User")
}

// RemoveMemberFromGroup 从组中移除成员（支持用户和组）
func (s *AuthingStore) RemoveMemberFromGroup(groupID, memberID string) error {
	// Authing API 目前只支持从组中移除用户
	url := s.host + "/groups/" + groupID + "/members/" + memberID
	headers := s.getAuthHeaders()

	_, err := s.sendRequest("DELETE", url, headers, nil)
	return err
}

// RemoveUserFromGroup 从组中移除用户
func (s *AuthingStore) RemoveUserFromGroup(groupID, userID string) error {
	return s.RemoveMemberFromGroup(groupID, userID)
}

// IsUserInGroup 检查用户是否在组中
func (s *AuthingStore) IsUserInGroup(groupID, userID string) (bool, error) {
	url := s.host + "/groups/" + groupID + "/members"
	headers := s.getAuthHeaders()

	resp, err := s.sendRequest("GET", url, headers, nil)
	if err != nil {
		return false, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return false, fmt.Errorf("解析响应失败: %v", err)
	}

	if members, ok := result["list"].([]interface{}); ok {
		for _, member := range members {
			if memberMap, ok := member.(map[string]interface{}); ok {
				if memberMap["id"].(string) == userID {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// GetUserGroups 获取用户所属的所有组
func (s *AuthingStore) GetUserGroups(userID string) ([]model.UserGroup, error) {
	url := s.host + "/users/" + userID + "/groups"
	headers := s.getAuthHeaders()

	resp, err := s.sendRequest("GET", url, headers, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	groups := []model.UserGroup{}
	if items, ok := result["list"].([]interface{}); ok {
		for _, item := range items {
			if groupMap, ok := item.(map[string]interface{}); ok {
				groups = append(groups, model.UserGroup{
					Value:   groupMap["id"].(string),
					Display: groupMap["name"].(string),
				})
			}
		}
	}

	return groups, nil
}

// SetHost 设置服务主机地址
func (s *AuthingStore) SetHost(host string) {
	s.host = host
}

// ---------------------- 辅助方法 ----------------------

// getAuthHeaders 获取认证头部
func (s *AuthingStore) getAuthHeaders() map[string]string {
	return map[string]string{
		"Authorization":         "Bearer " + s.accessKey,
		"Content-Type":          "application/json",
		"x-authing-userpool-id": s.userPoolID,
	}
}

// sendRequest 发送 HTTP 请求
func (s *AuthingStore) sendRequest(method, url string, headers map[string]string, data interface{}) ([]byte, error) {
	// 构建请求体
	var body []byte
	var err error
	if data != nil {
		body, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("序列化数据失败: %v", err)
		}
	}

	// 创建请求
	req, err := http.NewRequest(method, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置头部
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	responseBody := make([]byte, 0, 4096)
	buffer := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			responseBody = append(responseBody, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("请求失败 (状态码: %d): %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// getPrimaryEmail 获取主邮箱
func getPrimaryEmail(emails []model.Email) string {
	for _, email := range emails {
		if email.Primary {
			return email.Value
		}
	}
	if len(emails) > 0 {
		return emails[0].Value
	}
	return ""
}

// getStringValue 安全获取字符串值
func getStringValue(data map[string]interface{}, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// getBoolValue 安全获取布尔值
func getBoolValue(data map[string]interface{}, key string) bool {
	if value, ok := data[key]; ok {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

// RemoveEmailFromUser 从用户中移除指定邮箱
// Authing 不支持直接删除邮箱，需要通过更新用户来实现
func (s *AuthingStore) RemoveEmailFromUser(userID, emailValue string) error {
	user, err := s.GetUser(userID)
	if err != nil {
		return err
	}

	updatedEmails := make([]model.Email, 0)
	for _, email := range user.Emails {
		if !strings.EqualFold(email.Value, emailValue) {
			updatedEmails = append(updatedEmails, email)
		}
	}

	// 如果数量没变，说明没找到，直接返回 nil
	if len(updatedEmails) == len(user.Emails) {
		return nil
	}

	user.Emails = updatedEmails
	return s.UpdateUser(user)
}

// RemoveRoleFromUser 从用户中移除指定角色
// Authing 不支持直接删除角色，需要通过更新用户来实现
func (s *AuthingStore) RemoveRoleFromUser(userID, roleValue string) error {
	user, err := s.GetUser(userID)
	if err != nil {
		return err
	}

	updatedRoles := make([]model.Role, 0)
	for _, role := range user.Roles {
		if !strings.EqualFold(role.Value, roleValue) {
			updatedRoles = append(updatedRoles, role)
		}
	}

	// 如果数量没变，说明没找到，直接返回 nil
	if len(updatedRoles) == len(user.Roles) {
		return nil
	}

	user.Roles = updatedRoles
	return s.UpdateUser(user)
}
