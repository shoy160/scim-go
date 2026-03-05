package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"scim-go/internal/config"
)

func main() {
	var (
		format  = flag.String("format", "yaml", "配置文件格式 (yaml/json)")
		output  = flag.String("output", "", "输出文件路径 (默认输出到控制台)")
		mode    = flag.String("mode", "debug", "运行模式 (debug/test/release)")
		storage = flag.String("storage", "memory", "存储驱动 (memory/redis/mysql/postgres/authing)")
		helper  = flag.Bool("help", false, "显示帮助信息")
	)
	flag.Parse()

	if *helper {
		printHelp()
		return
	}

	// 验证格式
	formatLower := strings.ToLower(*format)
	if formatLower != "yaml" && formatLower != "yml" && formatLower != "json" {
		fmt.Fprintf(os.Stderr, "错误: 不支持的格式 '%s'。支持的格式: yaml, json\n", *format)
		os.Exit(1)
	}

	// 验证模式
	validModes := map[string]bool{"debug": true, "test": true, "release": true}
	if !validModes[*mode] {
		fmt.Fprintf(os.Stderr, "错误: 无效的模式 '%s'。有效值: debug, test, release\n", *mode)
		os.Exit(1)
	}

	// 验证存储驱动
	validStorages := map[string]bool{"memory": true, "redis": true, "mysql": true, "postgres": true, "authing": true}
	if !validStorages[*storage] {
		fmt.Fprintf(os.Stderr, "错误: 无效的存储驱动 '%s'。有效值: memory, redis, mysql, postgres, authing\n", *storage)
		os.Exit(1)
	}

	// 生成配置模板
	template, err := config.GenerateConfigTemplate(formatLower)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 生成配置模板失败: %v\n", err)
		os.Exit(1)
	}

	// 替换模板中的值
	template = customizeTemplate(template, *mode, *storage)

	// 输出配置
	if *output != "" {
		if err := os.WriteFile(*output, []byte(template), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "错误: 写入文件失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("配置文件已生成: %s\n", *output)
		fmt.Printf("运行模式: %s\n", *mode)
		fmt.Printf("存储驱动: %s\n", *storage)
	} else {
		fmt.Println(template)
	}
}

func customizeTemplate(template, mode, storage string) string {
	// 替换模式
	template = strings.ReplaceAll(template, "mode: debug", fmt.Sprintf("mode: %s", mode))

	// 替换存储驱动
	template = strings.ReplaceAll(template, "driver: memory", fmt.Sprintf("driver: %s", storage))

	// 根据存储驱动添加连接字符串示例
	switch storage {
	case "redis":
		template = strings.ReplaceAll(template, "redis_uri: \"\"", "redis_uri: \"redis://localhost:6379/0\"")
	case "mysql":
		template = strings.ReplaceAll(template, "mysql_dsn: \"\"", "mysql_dsn: \"user:password@tcp(localhost:3306)/scim?charset=utf8mb4&parseTime=True&loc=Local\"")
	case "postgres":
		template = strings.ReplaceAll(template, "postgres_dsn: \"\"", "postgres_dsn: \"host=localhost user=postgres password=postgres dbname=scim port=5432 sslmode=disable\"")
	}

	return template
}

func printHelp() {
	fmt.Println("SCIM 2.0 API 配置生成工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  go run cmd/config-gen/main.go [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -format string   配置文件格式 (yaml/json，默认: yaml)")
	fmt.Println("  -output string   输出文件路径 (默认输出到控制台)")
	fmt.Println("  -mode string     运行模式 (debug/test/release，默认: debug)")
	fmt.Println("  -storage string  存储驱动 (memory/redis/mysql/postgres/authing，默认: memory)")
	fmt.Println("  -help            显示帮助信息")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  # 生成默认配置到控制台")
	fmt.Println("  go run cmd/config-gen/main.go")
	fmt.Println()
	fmt.Println("  # 生成生产环境配置")
	fmt.Println("  go run cmd/config-gen/main.go -mode release -storage mysql -output config.yaml")
	fmt.Println()
	fmt.Println("  # 生成 JSON 格式配置")
	fmt.Println("  go run cmd/config-gen/main.go -format json -output config.json")
}
