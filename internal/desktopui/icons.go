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
	}
}
