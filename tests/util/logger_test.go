package util

import (
	"errors"
	"scim-go/util"
	"testing"
)

func TestLogger(t *testing.T) {
	// 测试不同级别的日志
	util.Debug("Debug message", map[string]string{"key": "value"})
	util.Info("Info message", map[string]string{"key": "value"})
	util.Warn("Warn message", errors.New("warn error"), map[string]string{"key": "value"})
	util.Error("Error message", errors.New("error error"), map[string]string{"key": "value"})
	util.Fatal("Fatal message", errors.New("fatal error"), map[string]string{"key": "value"})

	// 测试API错误日志
	util.LogAPIError(
		"API Error Test",
		errors.New("api error"),
		"/Users/123",
		"PUT",
		map[string]string{"userName": "test", "password": "secret"},
		"192.168.1.1",
		500,
		"10ms",
	)

	// 测试敏感信息过滤
	util.LogAPIError(
		"Sensitive Data Test",
		errors.New("test error"),
		"/test",
		"POST",
		map[string]string{
			"userName": "test",
			"password": "secret123",
			"token":    "abc123",
			"secret":   "def456",
			"key":      "ghi789",
		},
		"192.168.1.1",
		400,
		"5ms",
	)

	t.Log("Logger tests completed")
}
