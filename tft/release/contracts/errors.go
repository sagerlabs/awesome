package contracts

import (
	"fmt"
	"strings"
)

// 错误类别：结构化、可编程判断，校验器与调用方不得用字符串匹配代替。
const (
	// ErrCategoryFormat JSON 解析失败或基本类型不符。
	ErrCategoryFormat = "format"
	// ErrCategorySchema schema_version 不在 reader 兼容范围或未知。
	ErrCategorySchema = "schema"
	// ErrCategoryScope DataScope 非法（region/queue/patch/时间等）。
	ErrCategoryScope = "scope"
	// ErrCategoryFileMissing 清单声明的必需文件缺失。
	ErrCategoryFileMissing = "file_missing"
	// ErrCategoryFilePath 越界路径、绝对路径、路径逃逸或符号链接。
	ErrCategoryFilePath = "file_path"
	// ErrCategoryHash hash 字段格式非法（长度/字符集）。内容比对属于 DATA-02。
	ErrCategoryHash = "hash"
	// ErrCategorySelfReference manifest 出现在自身 files 列表（哈希自引用）。
	ErrCategorySelfReference = "self_reference"
	// ErrCategoryTimeOrder 时间语义矛盾（fetched/built/window 顺序）。
	ErrCategoryTimeOrder = "time_order"
	// ErrCategoryCapability 声明的 capability 缺少对应必需文件。
	ErrCategoryCapability = "capability"
	// ErrCategoryIdentity ID 字段缺失或格式非法（release/deployment/session 等）。
	ErrCategoryIdentity = "identity"
)

// ContractError 是契约校验的结构化错误。
// Category 供程序判断（见 ErrCategory* 常量），Field/Path/Reason 供人定位。
type ContractError struct {
	Category string `json:"category"`
	Field    string `json:"field,omitempty"`
	Path     string `json:"path,omitempty"`
	Reason   string `json:"reason"`
}

func (e *ContractError) Error() string {
	var b strings.Builder
	b.WriteString("contract: ")
	b.WriteString(e.Category)
	if e.Field != "" {
		fmt.Fprintf(&b, " field=%s", e.Field)
	}
	if e.Path != "" {
		fmt.Fprintf(&b, " path=%s", e.Path)
	}
	if e.Reason != "" {
		fmt.Fprintf(&b, ": %s", e.Reason)
	}
	return b.String()
}

// newErr 构造结构化错误。
func newErr(category, field, path, format string, args ...interface{}) *ContractError {
	return &ContractError{
		Category: category,
		Field:    field,
		Path:     path,
		Reason:   fmt.Sprintf(format, args...),
	}
}
