package desktopui

import (
	"gioui.org/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// IconSet 应用图标集（Material 图标，链接器只打包用到的）。
type IconSet struct {
	Dashboard *widget.Icon
	List      *widget.Icon
	Download  *widget.Icon
	Sync      *widget.Icon
	Snapshot  *widget.Icon
	Search    *widget.Icon
	Settings  *widget.Icon
	Check     *widget.Icon
	Warn      *widget.Icon
	Add       *widget.Icon
	Delete    *widget.Icon
	Edit      *widget.Icon
	History   *widget.Icon
	Key       *widget.Icon
	Close     *widget.Icon
	// 导航补齐（12 页）
	CompareArrows *widget.Icon // 一致性
	Timeline      *widget.Icon // 活动
	Explore       *widget.Icon // 发现
	Hub           *widget.Icon // 关系
	Renew         *widget.Icon // 同步
	// 操作补齐
	Refresh    *widget.Icon // 刷新
	Star       *widget.Icon // 收藏（实）
	StarBorder *widget.Icon // 收藏（空）
	Restore    *widget.Icon // 恢复
	Upload     *widget.Icon // 下发/推送
	Info       *widget.Icon // 信息
}

// MustIcons 构造图标集（失败 panic——图标数据是编译期常量，失败即编程错误）。
func MustIcons() IconSet {
	mk := func(data []byte) *widget.Icon {
		ic, err := widget.NewIcon(data)
		if err != nil {
			panic(err)
		}
		return ic
	}
	return IconSet{
		Dashboard: mk(icons.ActionDashboard),
		List:      mk(icons.ActionList),
		Download:  mk(icons.FileFileDownload),
		Sync:      mk(icons.ActionSwapHoriz),
		Snapshot:  mk(icons.ActionBackup),
		Search:    mk(icons.ActionSearch),
		Settings:  mk(icons.ActionSettings),
		Check:     mk(icons.ActionCheckCircle),
		Warn:      mk(icons.AlertWarning),
		Add:       mk(icons.ContentAdd),
		Delete:    mk(icons.ActionDelete),
		Edit:      mk(icons.ImageEdit),
		History:   mk(icons.ActionHistory),
		Key:       mk(icons.CommunicationVPNKey),
		Close:     mk(icons.NavigationClose),
		// 导航补齐
		CompareArrows: mk(icons.ActionCompareArrows),
		Timeline:      mk(icons.ActionTimeline),
		Explore:       mk(icons.ActionExplore),
		Hub:           mk(icons.HardwareDeviceHub),
		Renew:         mk(icons.ActionAutorenew),
		// 操作补齐
		Refresh:    mk(icons.NavigationRefresh),
		Star:       mk(icons.ToggleStar),
		StarBorder: mk(icons.ToggleStarBorder),
		Restore:    mk(icons.ActionRestore),
		Upload:     mk(icons.FileCloudUpload),
		Info:       mk(icons.ActionInfo),
	}
}
