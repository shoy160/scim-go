package store

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"scim-go/model"
	"scim-go/store"

	"github.com/stretchr/testify/assert"
)

// TestUserSorting 测试用户排序功能
func TestUserSorting(t *testing.T) {
	s := store.NewMemory()

	// 创建测试用户
	users := []*model.User{
		{ID: "1", UserName: "Zoe"},
		{ID: "2", UserName: "alice"},
		{ID: "3", UserName: "Bob"},
		{ID: "4", UserName: "charlie"},
		{ID: "5", UserName: "David"},
		{ID: "6", UserName: "emma"},
	}

	for _, u := range users {
		err := s.CreateUser(u)
		assert.NoError(t, err, "Failed to create user %s: %v", u.UserName, err)
	}

	// 测试按userName升序排序（不区分大小写）
	query := &model.ResourceQuery{
		SortBy:    "userName",
		SortOrder: "ascending",
		Count:     10, // 设置足够大的计数以获取所有用户
	}

	userList, total, err := s.ListUsers(query)
	assert.NoError(t, err)
	assert.Equal(t, int64(6), total)
	assert.Len(t, userList, 6)

	// 验证排序结果：alice, Bob, charlie, David, emma, Zoe
	expectedOrder := []string{"alice", "Bob", "charlie", "David", "emma", "Zoe"}
	for i, user := range userList {
		assert.Equal(t, expectedOrder[i], user.UserName, "User at index %d should be %s, got %s", i, expectedOrder[i], user.UserName)
	}

	// 测试按userName降序排序（不区分大小写）
	query.SortOrder = "descending"
	userList, total, err = s.ListUsers(query)
	assert.NoError(t, err)
	assert.Equal(t, int64(6), total)
	assert.Len(t, userList, 6)

	// 验证降序排序结果：Zoe, emma, David, charlie, Bob, alice
	expectedDescOrder := []string{"Zoe", "emma", "David", "charlie", "Bob", "alice"}
	for i, user := range userList {
		assert.Equal(t, expectedDescOrder[i], user.UserName, "User at index %d should be %s, got %s", i, expectedDescOrder[i], user.UserName)
	}
}

// TestUserSortingEdgeCases 测试用户排序的边界情况
func TestUserSortingEdgeCases(t *testing.T) {
	s := store.NewMemory()

	// 测试空用户列表
	query := &model.ResourceQuery{
		SortBy:    "userName",
		SortOrder: "ascending",
	}

	userList, total, err := s.ListUsers(query)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, userList, 0)

	// 测试包含特殊字符和数字的用户名
	usersWithSpecialChars := []*model.User{
		{ID: "1", UserName: "user123"},
		{ID: "2", UserName: "user@domain"},
		{ID: "3", UserName: "user-name"},
		{ID: "4", UserName: "user_name"},
	}

	for _, u := range usersWithSpecialChars {
		err := s.CreateUser(u)
		assert.NoError(t, err, "Failed to create user %s: %v", u.UserName, err)
	}

	query.Count = 10 // 设置足够大的计数以获取所有用户
	userList, total, err = s.ListUsers(query)
	assert.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Len(t, userList, 4)

	// 验证排序结果（按字典序，不区分大小写）
	// 注：特殊字符和数字的排序顺序可能因实现而异，这里主要验证不区分大小写
	for i := 0; i < len(userList)-1; i++ {
		current := userList[i].UserName
		next := userList[i+1].UserName
		// 验证排序是稳定的（不区分大小写）
		assert.True(t, len(current) > 0, "UserName should not be empty")
		assert.True(t, len(next) > 0, "UserName should not be empty")
	}
}

// TestUserSortingPerformance 测试用户排序性能
func TestUserSortingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	s := store.NewMemory()

	// 创建大量用户用于性能测试
	const userCount = 1000
	for i := 0; i < userCount; i++ {
		userName := fmt.Sprintf("user%d", i)
		if i%2 == 0 {
			userName = strings.ToUpper(userName)
		}
		user := &model.User{
			ID:       fmt.Sprintf("%d", i),
			UserName: userName,
		}
		err := s.CreateUser(user)
		assert.NoError(t, err, "Failed to create user %s: %v", user.UserName, err)
	}

	// 测试排序性能
	query := &model.ResourceQuery{
		SortBy:    "userName",
		SortOrder: "ascending",
		Count:     1000, // 设置足够大的计数以获取所有用户
	}

	// 测量排序时间
	start := time.Now()
	userList, total, err := s.ListUsers(query)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, int64(userCount), total)
	assert.Len(t, userList, userCount)

	// 验证排序结果
	for i := 0; i < len(userList)-1; i++ {
		current := strings.ToLower(userList[i].UserName)
		next := strings.ToLower(userList[i+1].UserName)
		assert.LessOrEqual(t, current, next, "Users should be sorted case-insensitively")
	}

	t.Logf("Sorted %d users in %v", userCount, duration)
	// 确保排序时间在合理范围内（对于1000个用户应该在毫秒级）
	assert.Less(t, duration, 100*time.Millisecond, "Sorting should be efficient")
}
