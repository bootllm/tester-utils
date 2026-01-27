package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试辅助函数：创建临时测试脚本
func createTestScript(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0755)
	require.NoError(t, err)
	return path
}

// ============== 核心功能测试 ==============

func TestRun_Execute(t *testing.T) {
	// 测试不带输入执行程序
	r := Run(".", "echo", "hello").Execute()

	assert.NoError(t, r.Error())
	assert.Contains(t, r.GetStdout(), "hello")
}

func TestRun_Stdin(t *testing.T) {
	// 测试带输入执行程序 (使用 grep)
	r := Run(".", "grep", "cat").Stdin("has cat")

	assert.NoError(t, r.Error())
	assert.Contains(t, r.GetStdout(), "cat")
}

func TestRun_Stdout(t *testing.T) {
	// 测试输出包含检查
	r := Run(".", "echo", "hello world").Execute().Stdout("world")
	assert.NoError(t, r.Error())

	// 测试输出不包含时应该报错
	r = Run(".", "echo", "hello").Execute().Stdout("world")
	assert.Error(t, r.Error())
	assert.IsType(t, &Mismatch{}, r.Error())
}

func TestRun_StdoutExact(t *testing.T) {
	// 测试精确匹配
	r := Run(".", "echo", "hello").Execute().StdoutExact("hello")
	assert.NoError(t, r.Error())

	// 测试不匹配时报错
	r = Run(".", "echo", "hello world").Execute().StdoutExact("hello")
	assert.Error(t, r.Error())
}

func TestRun_StdoutRegex(t *testing.T) {
	// 测试正则匹配
	r := Run(".", "echo", "hello123").Execute().StdoutRegex(`hello\d+`)
	assert.NoError(t, r.Error())

	// 测试不匹配时报错
	r = Run(".", "echo", "hello").Execute().StdoutRegex(`\d+`)
	assert.Error(t, r.Error())
}

func TestRun_Exit(t *testing.T) {
	// 测试退出码检查
	r := Run(".", "true").Execute().Exit(0)
	assert.NoError(t, r.Error())

	r = Run(".", "false").Execute().Exit(1)
	assert.NoError(t, r.Error())

	// 测试退出码不匹配
	r = Run(".", "false").Execute().Exit(0)
	assert.Error(t, r.Error())
	assert.IsType(t, &ExitCodeMismatch{}, r.Error())
}

func TestRun_ChainedCalls(t *testing.T) {
	// 测试链式调用
	r := Run(".", "echo", "hello").
		Execute().
		Stdout("hello").
		Exit(0)

	assert.NoError(t, r.Error())
}

func TestRun_ErrorPropagation(t *testing.T) {
	// 测试错误传播 - 一旦有错误，后续调用不执行
	r := Run(".", "echo", "hello").
		Execute().
		Stdout("nonexistent"). // 这里会失败
		Exit(0)                // 这个不应该执行

	assert.Error(t, r.Error())
	assert.IsType(t, &Mismatch{}, r.Error())
}

// ============== 配置选项测试 ==============

func TestWithTimeout(t *testing.T) {
	// 测试超时设置
	r := Run(".", "sleep", "10").
		WithTimeout(100 * time.Millisecond).
		Execute()

	// 应该超时
	result := r.Result()
	assert.NotNil(t, result)
}

func TestWithPty(t *testing.T) {
	// 测试 PTY 模式
	r := Run(".", "echo", "test").
		WithPty().
		Execute()

	assert.NoError(t, r.Error())
	// PTY 模式下输出可能包含 \r\n
	assert.Contains(t, r.GetStdout(), "test")
}

// ============== 交互模式测试 ==============

func TestStart_And_Kill(t *testing.T) {
	// 测试启动和终止
	r := Run(".", "sleep", "60").Start()
	assert.NoError(t, r.Error())
	assert.True(t, r.started)

	r.Kill()
	assert.False(t, r.started)
}

func TestReject(t *testing.T) {
	// 创建一个会等待输入的脚本
	tmpDir := t.TempDir()
	script := `#!/bin/bash
read -p "Enter: " input
echo "Got: $input"
`
	createTestScript(t, tmpDir, "wait_input.sh", script)

	// 程序应该拒绝（继续等待输入）
	r := Run(tmpDir, "wait_input.sh").
		WithPty().
		Start().
		Reject(200 * time.Millisecond)

	assert.NoError(t, r.Error(), "program should still be waiting for input")
	r.Kill()
}

func TestReject_Failure(t *testing.T) {
	// 创建一个立即退出的脚本
	tmpDir := t.TempDir()
	script := `#!/bin/bash
echo "done"
exit 0
`
	createTestScript(t, tmpDir, "quick_exit.sh", script)

	// 程序不应该拒绝（它立即退出了）
	r := Run(tmpDir, "quick_exit.sh").Start()

	// 等待程序退出
	time.Sleep(100 * time.Millisecond)

	r = r.Reject(200 * time.Millisecond)

	assert.Error(t, r.Error())
	assert.IsType(t, &RejectError{}, r.Error())
}

// ============== 辅助函数测试 ==============

func TestNormalizeOutput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello\r\nworld\r\n", "hello\nworld\n"},
		{"hello\nworld\n", "hello\nworld\n"},
		{"no newlines", "no newlines"},
		{"mixed\r\nand\nnewlines", "mixed\nand\nnewlines"},
	}

	for _, tc := range tests {
		result := normalizeOutput(tc.input)
		assert.Equal(t, tc.expected, result)
	}
}

// ============== 错误类型测试 ==============

func TestMismatch_Error(t *testing.T) {
	m := &Mismatch{
		Expected: "expected",
		Actual:   "actual",
		Message:  "custom message",
	}
	assert.Equal(t, "custom message", m.Error())

	m2 := &Mismatch{
		Expected: "expected",
		Actual:   "actual",
	}
	assert.Contains(t, m2.Error(), "expected")
	assert.Contains(t, m2.Error(), "actual")
}

func TestExitCodeMismatch_Error(t *testing.T) {
	e := &ExitCodeMismatch{
		Expected: 0,
		Actual:   1,
		Stderr:   "error message",
	}
	errMsg := e.Error()
	assert.Contains(t, errMsg, "0")
	assert.Contains(t, errMsg, "1")
	assert.Contains(t, errMsg, "error message")
}

func TestRejectError_Error(t *testing.T) {
	e := &RejectError{Message: "test error"}
	assert.Equal(t, "test error", e.Error())
}

// ============== 边界情况测试 ==============

func TestRun_NotExecuted(t *testing.T) {
	// 测试未执行时调用检查方法
	r := Run(".", "echo", "test")
	r = r.Stdout("test")
	assert.Error(t, r.Error())
	assert.Contains(t, r.Error().Error(), "not yet executed")

	r = Run(".", "echo", "test")
	r = r.Exit(0)
	assert.Error(t, r.Error())
}

func TestRun_CommandNotFound(t *testing.T) {
	// 测试命令不存在
	r := Run(".", "nonexistent_command_12345").Execute()
	assert.Error(t, r.Error())
}

func TestGetStdout_NilResult(t *testing.T) {
	// 测试 result 为 nil 时
	r := Run(".", "echo", "test")
	assert.Equal(t, "", r.GetStdout())
}
