package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LoadManifest 从磁盘读取并校验一份 manifest.json。
// 内容 hash 与文件存在性校验用 VerifyFiles（供 DATA-02 复用）。
func LoadManifest(path string) (*DataManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, newErr(ErrCategoryFileMissing, "manifest", path, "读取失败：%v", err)
	}
	var m DataManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, newErr(ErrCategoryFormat, "manifest", path, "JSON 解析失败：%v", err)
	}
	if err := m.Validate(); err != nil {
		if ce, ok := err.(*ContractError); ok && ce.Path == "" {
			ce.Path = path
		}
		return nil, err
	}
	return &m, nil
}

// VerifyFiles 校验 manifest.Files 相对 baseDir 的存在性、hash 与大小。
// 符号链接解析后必须仍在 baseDir 内（拒绝逃逸）。
// DATA-01 提供该能力供跨语言 fixture 验证与测试复用；
// 生产构建门禁的接线属于 DATA-02。
func (m *DataManifest) VerifyFiles(baseDir string) error {
	baseReal, err := filepath.EvalSymlinks(baseDir)
	if err != nil {
		return newErr(ErrCategoryFilePath, "baseDir", baseDir, "基目录不可解析：%v", err)
	}
	for i := range m.Files {
		p := filepath.Join(baseReal, filepath.FromSlash(m.Files[i].RelativePath))
		real, err := filepath.EvalSymlinks(p)
		if err != nil {
			return newErr(ErrCategoryFileMissing, "files", m.Files[i].RelativePath, "文件不存在：%v", err)
		}
		if !withinDir(real, baseReal) {
			return newErr(ErrCategoryFilePath, "files", m.Files[i].RelativePath,
				"路径越界（符号链接逃逸出基目录）：%s", real)
		}
		sum, n, err := fileSHA256(real)
		if err != nil {
			return newErr(ErrCategoryHash, "sha256", m.Files[i].RelativePath, "读取失败：%v", err)
		}
		if sum != m.Files[i].SHA256 {
			return newErr(ErrCategoryHash, "sha256", m.Files[i].RelativePath,
				"hash 不符：manifest=%s actual=%s", m.Files[i].SHA256, sum)
		}
		if n != m.Files[i].Size {
			return newErr(ErrCategoryHash, "size", m.Files[i].RelativePath,
				"size 不符：manifest=%d actual=%d", m.Files[i].Size, n)
		}
	}
	return nil
}

// withinDir 判断 path 解析后是否位于 dir 内（不含 dir 本身）。
func withinDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// fileSHA256 计算文件 sha256 与字节数。
func fileSHA256(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
