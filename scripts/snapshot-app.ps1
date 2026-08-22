param(
  [string]$Exe = "F:\config-code\cfg4ai-desktop.exe",
  [string]$Theme = "",
  [string]$Page = "",
  [string]$Out = "F:\config-code\snap.png",
  [int]$WaitSec = 6
)
Add-Type -AssemblyName System.Drawing
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class Win32 {
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT lpRect);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  public struct RECT { public int Left, Top, Right, Bottom; }
}
"@
$argList = @()
$argList = @(); if ($Theme -ne "") { $argList += @("-theme", $Theme) }; if ($Page -ne "") { $argList += @("-page", $Page) }
$proc = Start-Process -FilePath $Exe -ArgumentList $argList -PassThru
Start-Sleep -Seconds $WaitSec
$proc.Refresh()
$hwnd = $proc.MainWindowHandle
if ($hwnd -eq [IntPtr]::Zero) { Write-Host "NO WINDOW"; if(-not $proc.HasExited){ Stop-Process -Id $proc.Id -Force }; exit 1 }
[Win32]::SetForegroundWindow($hwnd) | Out-Null
Start-Sleep -Milliseconds 800
$r = New-Object Win32+RECT
[Win32]::GetWindowRect($hwnd, [ref]$r) | Out-Null
$w = $r.Right - $r.Left; $h = $r.Bottom - $r.Top
$bmp = New-Object System.Drawing.Bitmap $w, $h
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.CopyFromScreen($r.Left, $r.Top, 0, 0, $bmp.Size)
$bmp.Save($Out, [System.Drawing.Imaging.ImageFormat]::Png)
$g.Dispose(); $bmp.Dispose()
Write-Host "SNAP OK $Out ${w}x${h}"
if(-not $proc.HasExited){ Stop-Process -Id $proc.Id -Force }