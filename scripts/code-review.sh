#!/bin/bash

# 代码审查脚本
# 用于检测重复代码、代码复杂度和代码风格

# 设置颜色变量
GREEN="\033[0;32m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m" # No Color

echo -e "${GREEN}=== SCIM-Go 代码审查工具 ===${NC}"
echo

# 检查是否安装了必要的工具
check_tool() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}错误: $1 未安装${NC}"
        echo -e "${YELLOW}请运行: go install $2${NC}"
        return 1
    fi
    return 0
}

echo -e "${GREEN}检查必要的工具...${NC}"
check_tool "dupl" "honnef.co/go/tools/cmd/dupl"
check_tool "gocyclo" "github.com/fzipp/gocyclo/cmd/gocyclo"
check_tool "golint" "golang.org/x/lint/golint"
check_tool "goimports" "golang.org/x/tools/cmd/goimports"

echo

# 运行代码重复检测
echo -e "${GREEN}=== 检测重复代码 ===${NC}"
dupl -t 50 ./...

echo

# 运行代码复杂度检测
echo -e "${GREEN}=== 检测代码复杂度 ===${NC}"
gocyclo -over 15 ./...

echo

# 运行代码风格检查
echo -e "${GREEN}=== 检查代码风格 ===${NC}"
golint ./...

echo

# 运行代码静态分析
echo -e "${GREEN}=== 代码静态分析 ===${NC}"
go vet ./...

echo

# 检查导入格式
echo -e "${GREEN}=== 检查导入格式 ===${NC}"
goimports -l ./...

echo

# 运行测试
echo -e "${GREEN}=== 运行测试 ===${NC}"
go test ./...

echo

echo -e "${GREEN}=== 代码审查完成 ===${NC}"
