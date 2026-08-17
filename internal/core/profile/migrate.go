package profile

import "fmt"

// CurrentIRVersion 当前实现支持的 IR 结构版本。
const CurrentIRVersion = 1

// MigrateManifest 对 manifest 执行 ir_version 链式迁移（IR-SCHEMA §2.2）。
// 实体/manifest 版本高于实现版本时拒绝并提示升级。
func MigrateManifest(m *Manifest) error {
	if m.IRVersion > CurrentIRVersion {
		return fmt.Errorf("profile: manifest ir_version %d 高于实现版本 %d，请升级 cfg4ai", m.IRVersion, CurrentIRVersion)
	}
	if m.IRVersion < CurrentIRVersion {
		// 链式迁移占位：v1 恒等。未来 v1→v2 等迁移函数在此追加。
		m.IRVersion = CurrentIRVersion
	}
	return nil
}
