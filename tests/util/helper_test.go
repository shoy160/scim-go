package util_test

import (
	"testing"

	"scim-go/util"
)

// TestStructConverter_ToMap 测试结构体转换为 map
func TestStructConverter_ToMap(t *testing.T) {
	converter := util.StructConverter{}

	testStruct := struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{
		ID:   "test-id",
		Name: "test-name",
		Age:  25,
	}

	result, err := converter.ToMap(testStruct)
	if err != nil {
		t.Errorf("ToMap() error = %v, want nil", err)
	}

	if result["id"] != "test-id" {
		t.Errorf("ToMap() id = %v, want %v", result["id"], "test-id")
	}

	if result["name"] != "test-name" {
		t.Errorf("ToMap() name = %v, want %v", result["name"], "test-name")
	}

	// JSON 解析会将 int 转换为 float64
	if result["age"] != float64(25) {
		t.Errorf("ToMap() age = %v, want %v", result["age"], 25)
	}
}

// TestStructConverter_FromMap 测试 map 转换为结构体
func TestStructConverter_FromMap(t *testing.T) {
	converter := util.StructConverter{}

	testMap := map[string]interface{}{
		"id":   "test-id",
		"name": "test-name",
		"age":  25,
	}

	var result struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	err := converter.FromMap(testMap, &result)
	if err != nil {
		t.Errorf("FromMap() error = %v, want nil", err)
	}

	if result.ID != "test-id" {
		t.Errorf("FromMap() id = %v, want %v", result.ID, "test-id")
	}

	if result.Name != "test-name" {
		t.Errorf("FromMap() name = %v, want %v", result.Name, "test-name")
	}

	if result.Age != 25 {
		t.Errorf("FromMap() age = %v, want %v", result.Age, 25)
	}
}

// TestAttributeValidator_ValidateEmail 测试邮箱验证
func TestAttributeValidator_ValidateEmail(t *testing.T) {
	validator := util.AttributeValidator{}

	// 测试有效邮箱
	err := validator.ValidateEmail("test@example.com")
	if err != nil {
		t.Errorf("ValidateEmail('test@example.com') error = %v, want nil", err)
	}

	// 测试空邮箱
	err = validator.ValidateEmail("")
	if err == nil {
		t.Error("ValidateEmail('') should return error")
	}

	// 测试无效邮箱（缺少 @）
	err = validator.ValidateEmail("testexample.com")
	if err == nil {
		t.Error("ValidateEmail('testexample.com') should return error")
	}

	// 测试无效邮箱（缺少域名）
	err = validator.ValidateEmail("test@")
	if err == nil {
		t.Error("ValidateEmail('test@') should return error")
	}

	// 测试无效邮箱（缺少用户名）
	err = validator.ValidateEmail("@example.com")
	if err == nil {
		t.Error("ValidateEmail('@example.com') should return error")
	}
}

// TestAttributeValidator_ValidateRole 测试角色验证
func TestAttributeValidator_ValidateRole(t *testing.T) {
	validator := util.AttributeValidator{}

	// 测试有效角色
	err := validator.ValidateRole("admin")
	if err != nil {
		t.Errorf("ValidateRole('admin') error = %v, want nil", err)
	}

	// 测试空角色
	err = validator.ValidateRole("")
	if err == nil {
		t.Error("ValidateRole('') should return error")
	}
}

// TestAttributeValidator_ValidateAttribute 测试属性格式验证
func TestAttributeValidator_ValidateAttribute(t *testing.T) {
	validator := util.AttributeValidator{}

	// 测试有效属性
	err := validator.ValidateAttribute("userName")
	if err != nil {
		t.Errorf("ValidateAttribute('userName') error = %v, want nil", err)
	}

	// 测试有效嵌套属性
	err = validator.ValidateAttribute("name.givenName")
	if err != nil {
		t.Errorf("ValidateAttribute('name.givenName') error = %v, want nil", err)
	}

	// 测试有效通配符属性
	err = validator.ValidateAttribute("members.*")
	if err != nil {
		t.Errorf("ValidateAttribute('members.*') error = %v, want nil", err)
	}

	// 测试空属性
	err = validator.ValidateAttribute("")
	if err == nil {
		t.Error("ValidateAttribute('') should return error")
	}

	// 测试以点开头的属性
	err = validator.ValidateAttribute(".userName")
	if err == nil {
		t.Error("ValidateAttribute('.userName') should return error")
	}

	// 测试包含连续点的属性
	err = validator.ValidateAttribute("name..givenName")
	if err == nil {
		t.Error("ValidateAttribute('name..givenName') should return error")
	}

	// 测试包含非法字符的属性
	err = validator.ValidateAttribute("user@name")
	if err == nil {
		t.Error("ValidateAttribute('user@name') should return error")
	}

	// 测试无效通配符模式
	err = validator.ValidateAttribute("members*.*")
	if err == nil {
		t.Error("ValidateAttribute('members*.*') should return error")
	}

	// 测试以数字开头的属性
	err = validator.ValidateAttribute("123user")
	if err == nil {
		t.Error("ValidateAttribute('123user') should return error")
	}
}

// TestListProcessor_Filter 测试列表过滤
func TestListProcessor_Filter(t *testing.T) {
	processor := util.ListProcessor{}

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
	result, err := processor.Filter(items, "age eq 30", toMap)
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

	// 测试空过滤条件
	result, err = processor.Filter(items, "", toMap)
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

// TestListProcessor_Paginate 测试列表分页
func TestListProcessor_Paginate(t *testing.T) {
	processor := util.ListProcessor{}

	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// 测试正常分页
	result := processor.Paginate(items, 1, 5)
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
	result = processor.Paginate(items, 11, 5)
	if result == nil {
		t.Error("Paginate(11, 5) should not return nil")
	}
	resultSlice, ok = result.([]int)
	if !ok || len(resultSlice) != 0 {
		t.Errorf("Paginate(11, 5) = %v, want []", resultSlice)
	}

	// 测试边界情况：count 大于剩余数量
	result = processor.Paginate(items, 8, 5)
	if result == nil {
		t.Error("Paginate(8, 5) should not return nil")
	}
	resultSlice, ok = result.([]int)
	if !ok || len(resultSlice) != 3 || resultSlice[0] != 8 || resultSlice[2] != 10 {
		t.Errorf("Paginate(8, 5) = %v, want [8, 9, 10]", resultSlice)
	}
}

// TestListProcessor_Sort 测试列表排序
func TestListProcessor_Sort(t *testing.T) {
	processor := util.ListProcessor{}

	items := []struct {
		ID   string
		Name string
		Age  int
	}{
		{"1", "Alice", 25},
		{"2", "Bob", 30},
		{"3", "Charlie", 35},
	}

	// 测试按年龄升序排序
	processor.Sort(items, "age", "ascending", func(i, j int) bool {
		return items[i].Age < items[j].Age
	})

	if items[0].Age != 25 || items[1].Age != 30 || items[2].Age != 35 {
		t.Errorf("Sort(age, ascending) = %v, want sorted by age ascending", items)
	}

	// 测试按年龄降序排序
	processor.Sort(items, "age", "descending", func(i, j int) bool {
		return items[i].Age < items[j].Age
	})

	if items[0].Age != 35 || items[1].Age != 30 || items[2].Age != 25 {
		t.Errorf("Sort(age, descending) = %v, want sorted by age descending", items)
	}
}
