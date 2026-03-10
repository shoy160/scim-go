package store_test

import (
	"testing"
	"time"

	"scim-go/model"
	"scim-go/store"
)

// TestBaseResource 测试基础资源结构
func TestBaseResource(t *testing.T) {
	r := &store.BaseResource{
		ID:      "test-id",
		Schemas: []string{"test-schema"},
		Version: "test-version",
	}

	if r.GetID() != "test-id" {
		t.Errorf("GetID() = %v, want %v", r.GetID(), "test-id")
	}

	if len(r.GetSchemas()) != 1 || r.GetSchemas()[0] != "test-schema" {
		t.Errorf("GetSchemas() = %v, want %v", r.GetSchemas(), []string{"test-schema"})
	}

	r.SetSchemas([]string{"new-schema"})
	if len(r.GetSchemas()) != 1 || r.GetSchemas()[0] != "new-schema" {
		t.Errorf("SetSchemas() failed, got %v, want %v", r.GetSchemas(), []string{"new-schema"})
	}

	if r.GetVersion() != "test-version" {
		t.Errorf("GetVersion() = %v, want %v", r.GetVersion(), "test-version")
	}

	r.SetVersion("new-version")
	if r.GetVersion() != "new-version" {
		t.Errorf("SetVersion() failed, got %v, want %v", r.GetVersion(), "new-version")
	}
}

// TestGenerateMeta 测试生成 meta 属性
func TestGenerateMeta(t *testing.T) {
	meta, createdAt, updatedAt := store.GenerateMeta("User")

	if meta.ResourceType != "User" {
		t.Errorf("ResourceType = %v, want %v", meta.ResourceType, "User")
	}

	if meta.Created == "" {
		t.Error("Created should not be empty")
	}

	if meta.LastModified == "" {
		t.Error("LastModified should not be empty")
	}

	if meta.Version == "" {
		t.Error("Version should not be empty")
	}

	if createdAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	if updatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

// TestUpdateMeta 测试更新 meta 属性
func TestUpdateMeta(t *testing.T) {
	meta := model.Meta{
		ResourceType: "User",
		Created:      time.Now().Format(time.RFC3339Nano),
		LastModified: time.Now().Format(time.RFC3339Nano),
		Version:      "old-version",
	}

	oldVersion := meta.Version
	newVersion := store.UpdateMeta(&meta)

	if meta.Version == oldVersion {
		t.Error("Version should be updated")
	}

	if newVersion != meta.Version {
		t.Errorf("UpdateMeta() return value = %v, want %v", newVersion, meta.Version)
	}

	if meta.LastModified == oldVersion {
		t.Error("LastModified should be updated")
	}
}

// TestPaginate 测试通用分页函数
func TestPaginate(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// 测试正常分页
	result := store.Paginate(items, 1, 5)
	if result == nil {
		t.Error("Paginate(1, 5) should not return nil")
	}
	resultSlice, ok := result.([]int)
	if !ok {
		t.Error("Paginate(1, 5) should return []int")
	}
	if len(resultSlice) != 5 || resultSlice[0] != 1 || resultSlice[4] != 5 {
		t.Errorf("Paginate(1, 5) = %v, want [1, 2, 3, 4, 5]", resultSlice)
	}

	// 测试边界情况：startIndex 大于长度
	result = store.Paginate(items, 11, 5)
	if result == nil {
		t.Error("Paginate(11, 5) should not return nil")
	}
	resultSlice, ok = result.([]int)
	if !ok || len(resultSlice) != 0 {
		t.Errorf("Paginate(11, 5) = %v, want []", resultSlice)
	}

	// 测试边界情况：count 大于剩余数量
	result = store.Paginate(items, 8, 5)
	if result == nil {
		t.Error("Paginate(8, 5) should not return nil")
	}
	resultSlice, ok = result.([]int)
	if !ok || len(resultSlice) != 3 || resultSlice[0] != 8 || resultSlice[2] != 10 {
		t.Errorf("Paginate(8, 5) = %v, want [8, 9, 10]", resultSlice)
	}

	// 测试边界情况：startIndex 为 0
	result = store.Paginate(items, 0, 5)
	if result == nil {
		t.Error("Paginate(0, 5) should not return nil")
	}
	resultSlice, ok = result.([]int)
	if !ok || len(resultSlice) != 5 || resultSlice[0] != 1 || resultSlice[4] != 5 {
		t.Errorf("Paginate(0, 5) = %v, want [1, 2, 3, 4, 5]", resultSlice)
	}
}

// TestToMap 测试通用转换为 map 函数
func TestToMap(t *testing.T) {
	testStruct := struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{
		ID:   "test-id",
		Name: "test-name",
	}

	result, err := store.ToMap(testStruct)
	if err != nil {
		t.Errorf("ToMap() error = %v, want nil", err)
	}

	if result["id"] != "test-id" {
		t.Errorf("ToMap() id = %v, want %v", result["id"], "test-id")
	}

	if result["name"] != "test-name" {
		t.Errorf("ToMap() name = %v, want %v", result["name"], "test-name")
	}
}

// TestFilter 测试通用过滤函数
func TestFilter(t *testing.T) {
	items := []struct {
		ID   string
		Name string
		Age  int
	}{
		{"1", "Alice", 25},
		{"2", "Bob", 30},
		{"3", "Charlie", 35},
	}

	toMap := func(item interface{}) (map[string]interface{}, error) {
		typedItem := item.(struct {
			ID   string
			Name string
			Age  int
		})
		return map[string]interface{}{
			"id":   typedItem.ID,
			"name": typedItem.Name,
			"age":  typedItem.Age,
		}, nil
	}

	// 测试过滤条件：age eq 30
	result, err := store.Filter(items, "age eq 30", toMap)
	if err != nil {
		t.Errorf("Filter() error = %v, want nil", err)
	}

	typedResult, ok := result.([]struct {
		ID   string
		Name string
		Age  int
	})
	if !ok {
		t.Error("Filter() should return []struct")
	}
	if len(typedResult) != 1 || typedResult[0].Name != "Bob" {
		t.Errorf("Filter('age eq 30') = %v, want [{ID:2 Name:Bob Age:30}]", typedResult)
	}

	// 测试过滤条件：age gt 25
	result, err = store.Filter(items, "age gt 25", toMap)
	if err != nil {
		t.Errorf("Filter() error = %v, want nil", err)
	}

	typedResult, ok = result.([]struct {
		ID   string
		Name string
		Age  int
	})
	if !ok {
		t.Error("Filter() should return []struct")
	}
	if len(typedResult) != 2 || typedResult[0].Name != "Bob" || typedResult[1].Name != "Charlie" {
		t.Errorf("Filter('age gt 25') = %v, want [{ID:2 Name:Bob Age:30}, {ID:3 Name:Charlie Age:35}]", typedResult)
	}

	// 测试空过滤条件
	result, err = store.Filter(items, "", toMap)
	if err != nil {
		t.Errorf("Filter() error = %v, want nil", err)
	}

	typedResult, ok = result.([]struct {
		ID   string
		Name string
		Age  int
	})
	if !ok {
		t.Error("Filter() should return []struct")
	}
	if len(typedResult) != 3 {
		t.Errorf("Filter('') = %v, want all items", typedResult)
	}
}
