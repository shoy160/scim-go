#!/bin/bash

# 代码质量监控脚本
# 用于检测代码质量指标并生成报告

set -e

# 设置颜色变量
GREEN="\033[0;32m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m" # No Color

# 设置报告目录
REPORT_DIR="reports"
mkdir -p $REPORT_DIR

# 生成报告文件名
REPORT_FILE="$REPORT_DIR/quality-report-$(date +%Y%m%d-%H%M%S).txt"
HTML_REPORT="$REPORT_DIR/quality-report-$(date +%Y%m%d-%H%M%S).html"

echo -e "${GREEN}=== SCIM-Go 代码质量监控 ===${NC}"
echo -e "${GREEN}报告生成时间: $(date)${NC}"
echo ""

# 创建报告文件
echo "=== SCIM-Go 代码质量报告 ===" > $REPORT_FILE
echo "生成时间: $(date)" >> $REPORT_FILE
echo "" >> $REPORT_FILE

# 代码重复检测
echo -e "${YELLOW}=== 检测代码重复 ===${NC}"
echo "=== 代码重复检测 ===" >> $REPORT_FILE
echo "" >> $REPORT_FILE

if command -v dupl &> /dev/null; then
    dupl -t 50 ./... >> $REPORT_FILE 2>&1 || true
    echo -e "${GREEN}代码重复检测完成${NC}"
else
    echo -e "${RED}dupl 未安装，跳过代码重复检测${NC}"
    echo "dupl 未安装，跳过代码重复检测" >> $REPORT_FILE
fi
echo "" >> $REPORT_FILE

# 圈复杂度检测
echo -e "${YELLOW}=== 检测圈复杂度 ===${NC}"
echo "=== 圈复杂度检测 ===" >> $REPORT_FILE
echo "" >> $REPORT_FILE

if command -v gocyclo &> /dev/null; then
    gocyclo -over 15 ./... >> $REPORT_FILE 2>&1 || true
    echo -e "${GREEN}圈复杂度检测完成${NC}"
else
    echo -e "${RED}gocyclo 未安装，跳过圈复杂度检测${NC}"
    echo "gocyclo 未安装，跳过圈复杂度检测" >> $REPORT_FILE
fi
echo "" >> $REPORT_FILE

# 代码风格检查
echo -e "${YELLOW}=== 检查代码风格 ===${NC}"
echo "=== 代码风格检查 ===" >> $REPORT_FILE
echo "" >> $REPORT_FILE

if command -v golint &> /dev/null; then
    golint ./... >> $REPORT_FILE 2>&1 || true
    echo -e "${GREEN}代码风格检查完成${NC}"
else
    echo -e "${RED}golint 未安装，跳过代码风格检查${NC}"
    echo "golint 未安装，跳过代码风格检查" >> $REPORT_FILE
fi
echo "" >> $REPORT_FILE

# 静态代码分析
echo -e "${YELLOW}=== 静态代码分析 ===${NC}"
echo "=== 静态代码分析 ===" >> $REPORT_FILE
echo "" >> $REPORT_FILE

go vet ./... >> $REPORT_FILE 2>&1 || true
echo -e "${GREEN}静态代码分析完成${NC}"
echo "" >> $REPORT_FILE

# 检查导入格式
echo -e "${YELLOW}=== 检查导入格式 ===${NC}"
echo "=== 导入格式检查 ===" >> $REPORT_FILE
echo "" >> $REPORT_FILE

if command -v goimports &> /dev/null; then
    goimports -l ./... >> $REPORT_FILE 2>&1 || true
    echo -e "${GREEN}导入格式检查完成${NC}"
else
    echo -e "${RED}goimports 未安装，跳过导入格式检查${NC}"
    echo "goimports 未安装，跳过导入格式检查" >> $REPORT_FILE
fi
echo "" >> $REPORT_FILE

# 运行测试
echo -e "${YELLOW}=== 运行测试 ===${NC}"
echo "=== 测试执行 ===" >> $REPORT_FILE
echo "" >> $REPORT_FILE

go test ./... >> $REPORT_FILE 2>&1 || true
echo -e "${GREEN}测试执行完成${NC}"
echo "" >> $REPORT_FILE

# 测试覆盖率
echo -e "${YELLOW}=== 检测测试覆盖率 ===${NC}"
echo "=== 测试覆盖率 ===" >> $REPORT_FILE
echo "" >> $REPORT_FILE

COVERAGE_FILE="$REPORT_DIR/coverage-$(date +%Y%m%d-%H%M%S).out"
go test -coverprofile=$COVERAGE_FILE ./... >> $REPORT_FILE 2>&1 || true

if [ -f $COVERAGE_FILE ]; then
    go tool cover -func=$COVERAGE_FILE >> $REPORT_FILE 2>&1 || true
    go tool cover -html=$COVERAGE_FILE -o $HTML_REPORT
    echo -e "${GREEN}测试覆盖率检测完成${NC}"
    echo -e "${GREEN}HTML 报告: $HTML_REPORT${NC}"
else
    echo -e "${RED}测试覆盖率文件生成失败${NC}"
fi
echo "" >> $REPORT_FILE

# 生成统计摘要
echo "=== 统计摘要 ===" >> $REPORT_FILE
echo "" >> $REPORT_FILE

# 统计代码行数
echo "代码行数统计:" >> $REPORT_FILE
find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs wc -l | tail -1 >> $REPORT_FILE
echo "" >> $REPORT_FILE

# 统计文件数量
echo "文件数量统计:" >> $REPORT_FILE
find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | wc -l >> $REPORT_FILE
echo "" >> $REPORT_FILE

# 统计包数量
echo "包数量统计:" >> $REPORT_FILE
find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" -exec dirname {} \; | sort -u | wc -l >> $REPORT_FILE
echo "" >> $REPORT_FILE

echo "=== 报告完成 ===" >> $REPORT_FILE
echo "" >> $REPORT_FILE

# 显示报告摘要
echo -e "${GREEN}=== 报告生成完成 ===${NC}"
echo -e "${GREEN}文本报告: $REPORT_FILE${NC}"
if [ -f $HTML_REPORT ]; then
    echo -e "${GREEN}HTML 报告: $HTML_REPORT${NC}"
fi
echo ""
echo -e "${YELLOW}建议定期查看报告以了解代码质量趋势${NC}"

# 清理旧的报告（保留最近 30 天）
find $REPORT_DIR -name "quality-report-*.txt" -mtime +30 -delete
find $REPORT_DIR -name "coverage-*.out" -mtime +30 -delete
find $REPORT_DIR -name "quality-report-*.html" -mtime +30 -delete

echo -e "${GREEN}已清理 30 天前的旧报告${NC}"
