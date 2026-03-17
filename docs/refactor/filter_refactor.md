# 过滤器模块重构说明文档

## 概述

本次重构针对 `filter.go` 和 `path_filter.go` 两个文件中存在的功能重复问题进行了系统性重构。通过创建统一的抽象层和公共组件，消除了冗余代码，提高了代码的可维护性和执行效率。

## 重构前的问题分析

### 1. 操作符定义重复
- `filter.go`: 使用字符串常量表示操作符
- `path_filter.go`: 定义了 `FilterOperator` 类型和常量

### 2. 过滤表达式解析重复
- `filter.go`: `ParseFilter` 函数解析完整的 SCIM 过滤表达式
- `path_filter.go`: `parseFilterExpression` 函数解析路径中的过滤表达式

### 3. 匹配逻辑重复
- `filter.go`: `MatchFilter` 函数匹配对象
- `path_filter.go`: `MatchPathFilter` 函数匹配对象

### 4. 比较操作重复
- `filter.go`: 多个比较函数 (`compareEq`, `compareCo`, `compareSw`, `compareEw`, `comparePr`, `compareNumeric`)
- `path_filter.go`: `MatchPathFilter` 中的 switch 语句，`compareValues`

## 重构方案

### 新增文件

#### `filter_operator.go`
创建统一的操作符定义和比较逻辑模块：

```go
// Operator 定义 SCIM 过滤操作符类型
type Operator string

const (
    OpEq  Operator = "eq"  // 等于
    OpNe  Operator = "ne"  // 不等于
    OpGt  Operator = "gt"  // 大于
    OpGe  Operator = "ge"  // 大于等于
    OpLt  Operator = "lt"  // 小于
    OpLe  Operator = "le"  // 小于等于
    OpCo  Operator = "co"  // 包含
    OpSw  Operator = "sw"  // 以...开始
    OpEw  Operator = "ew"  // 以...结束
    OpPr  Operator = "pr"  // 存在
    OpAnd Operator = "and" // 逻辑与
    OpOr  Operator = "or"  // 逻辑或
    OpNot Operator = "not" // 逻辑非
)
```

主要功能：
- `CompareValues`: 统一的比较操作入口
- `compareEqual`: 字符串相等比较（支持大小写敏感配置）
- `compareContains`: 包含比较
- `compareStartsWith`: 前缀匹配
- `compareEndsWith`: 后缀匹配
- `compareOrdered`: 有序比较（支持数值和字符串）
- `FormatValue`: 值格式化

### 修改文件

#### `filter.go`
- 使用统一的 `Operator` 类型
- 使用 `CompareValues` 函数进行比较
- 简化 `matchComparison` 函数

#### `path_filter.go`
- 使用统一的 `Operator` 类型
- 使用 `CompareValues` 函数进行比较
- 简化 `MatchPathFilter` 函数
- 添加 `PathFilterToFilterNode` 转换函数

## 重构后的架构

```
┌─────────────────────────────────────────────────────────────┐
│                    filter_operator.go                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Operator 类型定义                        │   │
│  │  OpEq, OpNe, OpGt, OpGe, OpLt, OpLe, OpCo,           │   │
│  │  OpSw, OpEw, OpPr, OpAnd, OpOr, OpNot               │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │           统一比较函数 CompareValues                   │   │
│  │  - compareEqual (大小写敏感配置)                       │   │
│  │  - compareContains                                   │   │
│  │  - compareStartsWith                                 │   │
│  │  - compareEndsWith                                   │   │
│  │  - compareOrdered (数值/字符串)                       │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ 使用
          ┌───────────────────┴───────────────────┐
          │                                        │
┌─────────┴─────────┐                ┌────────────┴────────────┐
│    filter.go       │                │    path_filter.go       │
│                    │                │                         │
│  - ParseFilter     │                │  - ParsePathWithFilter  │
│  - MatchFilter     │                │  - MatchPathFilter      │
│  - FilterToSQL     │                │  - FindMatchingIndices  │
│  - FilterNode      │                │  - PathFilter           │
│                    │                │  - ParsedPath           │
└────────────────────┘                └─────────────────────────┘
```

## 性能优化

### 1. 减少重复计算
- 统一的比较函数避免了重复的字符串转换和比较逻辑
- 使用 `strings.EqualFold` 进行大小写不敏感比较，比 `strings.ToLower` + `==` 更高效

### 2. 缓存机制
- `filter.go` 中保留了过滤器解析结果缓存 (`filterCache`)
- 使用 `sync.Map` 实现并发安全的缓存

### 3. 内存优化
- 在字符串拼接中使用 `strings.Builder` 预分配容量
- 减少了临时字符串的创建

## 代码质量改进

### 1. 类型安全
- 使用 `Operator` 类型替代字符串常量，提供编译时类型检查
- 避免了字符串拼写错误导致的运行时问题

### 2. 可维护性
- 统一的比较逻辑便于维护和扩展
- 清晰的函数职责划分
- 完善的注释文档

### 3. 可测试性
- 独立的比较函数便于单元测试
- 统一的测试用例覆盖所有操作符

## 测试覆盖

### 新增测试
- `TestCompareValues`: 测试统一的比较函数
- 覆盖所有操作符的测试用例

### 测试结果
所有测试全部通过：
- `TestParsePathWithFilter`: 12 个测试用例
- `TestMatchPathFilter`: 10 个测试用例
- `TestFindMatchingIndices`: 4 个测试用例
- `TestCompareValues`: 10 个测试用例
- `TestParseFilter`: 11 个测试用例
- `TestMatchFilter`: 11 个测试用例
- `TestFilterToSQL`: 7 个测试用例
- `TestValidateFilter`: 3 个测试用例

## 向后兼容性

重构保持了完全的向后兼容性：
- 所有公共 API 保持不变
- 行为与重构前一致
- 现有代码无需修改

## 文件变更统计

| 文件 | 变更类型 | 行数变化 |
|------|----------|----------|
| `filter_operator.go` | 新增 | +197 |
| `filter.go` | 修改 | -150/+320 |
| `path_filter.go` | 修改 | -200/+197 |
| `path_filter_test.go` | 修改 | +535 |

## 总结

本次重构成功消除了 `filter.go` 和 `path_filter.go` 之间的功能重复，通过创建统一的 `filter_operator.go` 模块，实现了：

1. **代码复用**: 统一的操作符定义和比较逻辑
2. **类型安全**: 使用 `Operator` 类型提供编译时检查
3. **性能优化**: 减少重复计算，优化内存使用
4. **可维护性**: 清晰的架构，完善的文档
5. **可测试性**: 独立的函数便于测试，完整的测试覆盖

重构后的代码更加清晰、高效、易于维护，为后续功能扩展奠定了良好的基础。
