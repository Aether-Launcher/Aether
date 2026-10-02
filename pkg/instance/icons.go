package instance

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Aether-Launcher/Aether/pkg/fs"
)

// IconFileName is the conventional icon file stored inside an instance
// directory. No instance.json migration is needed: presence of the file
// means a custom icon, absence means the default gradient art.
const IconFileName = "icon.png"

// MaxIconBytes caps custom icons at 5 MiB.
const MaxIconBytes = 5 * 1024 * 1024

// allowedIconHosts whitelists where modpack icons may be downloaded from.
// User-picked files from disk bypass this check (the user chose them).
var allowedIconHosts = []string{
	"cdn.modrinth.com",
	"media.forgecdn.net",
	"edge.forgecdn.net",
}

func iconPath(id string) (string, error) {
	dir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, IconFileName), nil
}

func isAllowedIconURL(rawURL string) bool {
	u, err := neturl.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, h := range allowedIconHosts {
		if host == h || strings.HasSuffix(host, "."+h) {
			return true
		}
	}
	return false
}

// imageMIME sniffs PNG/JPEG magic bytes. Anything else is rejected.
func imageMIME(data []byte) (string, error) {
	if len(data) > MaxIconBytes {
		return "", fmt.Errorf("icon exceeds the 5 MB size limit")
	}
	if len(data) >= 8 &&
		data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		return "image/png", nil
	}
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg", nil
	}
	return "", fmt.Errorf("icon must be a PNG or JPEG image")
}

// writeIconAtomically validates image bytes and stores them as icon.png.
func writeIconAtomically(id string, data []byte) error {
	if _, err := imageMIME(data); err != nil {
		return err
	}
	dest, err := iconPath(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// SetIconFromFile validates a user-picked image and saves it as the
// instance icon.
func SetIconFromFile(id, srcPath string) error {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(srcPath) == "" {
		return fmt.Errorf("instance id and file path are required")
	}
	if st, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("cannot read icon file: %w", err)
	} else if st.Size() > MaxIconBytes {
		return fmt.Errorf("icon exceeds the 5 MB size limit")
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("cannot read icon file: %w", err)
	}
	return writeIconAtomically(id, data)
}

// SetIconFromURL downloads a modpack icon from an allowlisted host and
// saves it as the instance icon. Failures are returned to the caller,
// which should treat them as non-fatal (the pack installs regardless).
func SetIconFromURL(ctx context.Context, id, rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("empty icon URL")
	}
	if !isAllowedIconURL(rawURL) {
		return fmt.Errorf("icon host is not allowlisted")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("icon download failed with status %s", resp.Status)
	}
	// Read at most MaxIconBytes+1 so oversized files are rejected, not buffered.
	buf := make([]byte, 0, 64*1024)
	tmp := make([]byte, 32*1024)
	for {
		n, rerr := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) > MaxIconBytes {
				return fmt.Errorf("icon exceeds the 5 MB size limit")
			}
		}
		if rerr != nil {
			break
		}
	}
	return writeIconAtomically(id, buf)
}

// RemoveInstanceIcon deletes a custom icon; missing icon is not an error.
func RemoveInstanceIcon(id string) error {
	dest, err := iconPath(id)
	if err != nil {
		return err
	}
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetInstanceIcon returns the custom icon as a data: URL, or "" when the
// instance uses default art.
func GetInstanceIcon(id string) (string, error) {
	dest, err := iconPath(id)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	mime, err := imageMIME(data)
	if err != nil {
		return "", nil
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// GetInstanceIcons returns id -> data URL for every instance that has a
// custom icon, so lists can render icons with a single backend call.
func GetInstanceIcons() map[string]string {
	out := map[string]string{}
	for _, inst := range GetInstances() {
		if url, err := GetInstanceIcon(inst.ID); err == nil && url != "" {
			out[inst.ID] = url
		}
	}
	return out
}
