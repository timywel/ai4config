; cfg4ai 安装向导（Inno Setup 6）
; 生成向导式安装包：欢迎 → 许可 → 选目录 → 桌面图标 → 完成

#define MyAppName "cfg4ai 配置治理"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "cfg4ai"
#define MyAppURL "https://github.com/timywel/ai4config"
#define MyAppExe "cfg4ai-desktop.exe"

[Setup]
AppId={{8F3A2B1C-4D5E-6F70-8192-A3B4C5D6E7F8}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
; 免 UAC 用户级安装（与 VS Code/QQ 同级）
PrivilegesRequired=lowest
DefaultDirName={localappdata}\Programs\cfg4ai
DefaultGroupName=cfg4ai
OutputDir=..\dist
OutputBaseFilename=cfg4ai-Setup-{#MyAppVersion}-windows-amd64
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
DisableProgramGroupPage=yes
UninstallDisplayIcon={app}\{#MyAppExe}
UninstallDisplayName={#MyAppName}
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "chinesesimp"; MessagesFile: "compiler:Languages\ChineseSimplified.isl"

[Tasks]
Name: "desktopicon"; Description: "创建桌面快捷方式(&D)"; GroupDescription: "附加任务："

[Files]
Source: "..\dist\cfg4ai-desktop_windows_amd64_v1\cfg4ai-desktop.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\cfg4ai_windows_amd64_v1\cfg4ai.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\cfg4ai 配置治理"; Filename: "{app}\{#MyAppExe}"
Name: "{autodesktop}\cfg4ai"; Filename: "{app}\{#MyAppExe}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExe}"; Description: "立即启动 cfg4ai(&L)"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
; 卸载时清理应用目录残留（用户配置仓库 ~/.config/cfg4ai 保留，防误删配置）
Type: filesandordirs; Name: "{app}"