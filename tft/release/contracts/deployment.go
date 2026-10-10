package contracts

import "time"

// DeploymentManifest 描述一次部署的兼容组合身份（ADR013）。
//
// 原子选择兼容组合：应用镜像、数据 release、prompt、策略与模型配置
// 版本一起构成 deployment_id。禁止把模型密钥写进清单。
type DeploymentManifest struct {
	DeploymentID       string    `json:"deployment_id" yaml:"deployment_id"`
	ImageDigest        string    `json:"image_digest" yaml:"image_digest"`
	AppCommit          string    `json:"app_commit" yaml:"app_commit"`
	DataReleaseID      string    `json:"data_release_id" yaml:"data_release_id"`
	PromptVersion      string    `json:"prompt_version" yaml:"prompt_version"`
	PolicyVersion      string    `json:"policy_version" yaml:"policy_version"`
	ModelConfigVersion string    `json:"model_config_version" yaml:"model_config_version"`
	CreatedAt          time.Time `json:"created_at" yaml:"created_at"`
}

// Validate 校验部署清单结构。与 DataManifest 的兼容性组合校验
//（数据 schema 在应用 reader 范围内）由 ComposeWith 完成。
func (d *DeploymentManifest) Validate() error {
	if d == nil {
		return newErr(ErrCategoryFormat, "deployment", "", "deployment 为 nil")
	}
	for field, v := range map[string]string{
		"deployment_id":        d.DeploymentID,
		"image_digest":         d.ImageDigest,
		"app_commit":           d.AppCommit,
		"data_release_id":       d.DataReleaseID,
		"prompt_version":       d.PromptVersion,
		"policy_version":       d.PolicyVersion,
		"model_config_version": d.ModelConfigVersion,
	} {
		if v == "" {
			return newErr(ErrCategoryIdentity, field, "", "%s 不能为空", field)
		}
	}
	if d.CreatedAt.IsZero() {
		return newErr(ErrCategoryTimeOrder, "created_at", "", "created_at 为零值")
	}
	return nil
}

// ComposeWith 校验"应用 + 数据"组合兼容：数据契约版本必须落在
// 应用声明的 reader 范围内（ADR013：允许的组合必须在部署清单中
// 通过兼容性检查）。
func (d *DeploymentManifest) ComposeWith(m *DataManifest) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if m == nil {
		return newErr(ErrCategoryFormat, "manifest", "", "data manifest 为 nil")
	}
	if d.DataReleaseID != m.DataReleaseID {
		return newErr(ErrCategoryIdentity, "data_release_id", "",
			"部署清单与数据 release 不匹配：deployment=%s manifest=%s", d.DataReleaseID, m.DataReleaseID)
	}
	if err := SupportsSchema(d.readerRange(), m.SchemaVersion); err != nil {
		return newErr(ErrCategorySchema, "reader_schema_range", "",
			"应用不支持数据 schema %s：%v", m.SchemaVersion, err)
	}
	return nil
}

// readerRange 部署侧读取兼容范围的占位实现：v1 由应用构建时注入。
// DATA-01 先固定为当前契约版本范围，CORE-01 接管应用侧声明。
func (d *DeploymentManifest) readerRange() string { return "^1.0" }
