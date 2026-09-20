package icon

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/123cdxcc/SilentOpen/pkg/icondata"
)

// platformProvider resolves icons through the Windows shell.
type platformProvider struct{}

// NewPlatformProvider returns the Windows Provider.
func NewPlatformProvider() Provider { return platformProvider{} }

// Icon extracts the icon associated with executable.
func (platformProvider) Icon(ctx context.Context, executable string) string {
	// The script is fixed; paths cross the boundary as data, never PowerShell code.
	const script = `$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing
$icon = [System.Drawing.Icon]::ExtractAssociatedIcon($env:SILENTOPEN_ICON_EXE)
if ($null -eq $icon) { exit }
$bitmap = $null
$stream = $null
try {
  $bitmap = $icon.ToBitmap()
  $stream = New-Object System.IO.MemoryStream
  $bitmap.Save($stream, [System.Drawing.Imaging.ImageFormat]::Png)
  [Convert]::ToBase64String($stream.ToArray())
} finally {
  if ($stream) { $stream.Dispose() }
  if ($bitmap) { $bitmap.Dispose() }
  $icon.Dispose()
}`
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(entry), "SILENTOPEN_ICON_EXE=") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "SILENTOPEN_ICON_EXE="+executable)
	out, err := cmd.Output()
	if err != nil || len(out) > base64.StdEncoding.EncodedLen(icondata.MaxBytes)+2 {
		return ""
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(out)))
	if err != nil {
		return ""
	}
	return icondata.PNGURL(data)
}
