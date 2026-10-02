package servers

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Aether-Launcher/Aether/pkg/fs"
)

// ServerRow is one servers.dat entry with its live ping result attached.
// Field names are lowercase JSON by construction (see the sandbox field
// mapper) so extensions can read row.name / row.online directly.
type ServerRow struct {
	Name          string `json:"name"`
	IP            string `json:"ip"`
	Hidden        bool   `json:"hidden,omitempty"`
	Online        bool   `json:"online"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	MOTD          string `json:"motd,omitempty"`
	PlayersOnline int    `json:"playersOnline,omitempty"`
	PlayersMax    int    `json:"playersMax,omitempty"`
	Version       string `json:"version,omitempty"`
	LatencyMs     int64  `json:"latencyMs,omitempty"`
}

// statusCacheTTL bounds how long a ping result is reused. servers.dat is
// also checked by mtime+size, so in-game edits invalidate immediately while
// mere page revisits stay instant.
var statusCacheTTL = 45 * time.Second

// maxStatusParallel caps concurrent outbound pings for one list call.
const maxStatusParallel = 6

type statusCacheEntry struct {
	mtime   time.Time
	size    int64
	expires time.Time
	rows    []ServerRow
}

var (
	statusCacheMu sync.Mutex
	statusCache   = map[string]statusCacheEntry{}
)

// ListServersWithStatus reads an instance's servers.dat and pings every
// entry concurrently (bounded), so one dead server can't stall the list.
// Results are cached per servers.dat content; revisits are instant.
// perPingTimeout bounds each ping (clamped to 500ms–10s by the caller).
func ListServersWithStatus(instanceID string, perPingTimeout time.Duration) ([]ServerRow, error) {
	dir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
	if err != nil {
		return nil, err
	}
	return listServersWithStatusIn(filepath.Join(dir, "servers.dat"), perPingTimeout)
}

func listServersWithStatusIn(datPath string, perPingTimeout time.Duration) ([]ServerRow, error) {
	if perPingTimeout <= 0 {
		perPingTimeout = 3 * time.Second
	}
	st, err := os.Stat(datPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ServerRow{}, nil
		}
		return nil, err
	}

	statusCacheMu.Lock()
	if e, ok := statusCache[datPath]; ok && e.mtime.Equal(st.ModTime()) && e.size == st.Size() && time.Now().Before(e.expires) {
		rows := e.rows
		statusCacheMu.Unlock()
		return rows, nil
	}
	statusCacheMu.Unlock()

	data, err := os.ReadFile(datPath)
	if err != nil {
		return nil, err
	}
	entries, err := ParseServersDat(data)
	if err != nil {
		return nil, err
	}

	rows := make([]ServerRow, len(entries))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxStatusParallel)
	for i, e := range entries {
		wg.Add(1)
		go func(i int, e ServerEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res, err := PingWithTimeout(e.IP, perPingTimeout, perPingTimeout)
			row := ServerRow{Name: e.Name, IP: e.IP, Hidden: e.Hidden}
			if err == nil {
				row.Online = res.Online
				row.Host = res.Host
				row.Port = res.Port
				row.MOTD = res.MOTD
				row.PlayersOnline = res.PlayersOnline
				row.PlayersMax = res.PlayersMax
				row.Version = res.Version
				row.LatencyMs = res.LatencyMs
			} else {
				// Unparseable address: surface the raw entry so the UI can
				// still show (and attempt) it.
				row.Host = e.IP
				row.Port = 25565
			}
			rows[i] = row
		}(i, e)
	}
	wg.Wait()

	statusCacheMu.Lock()
	statusCache[datPath] = statusCacheEntry{mtime: st.ModTime(), size: st.Size(), expires: time.Now().Add(statusCacheTTL), rows: rows}
	statusCacheMu.Unlock()
	return rows, nil
}

// InvalidateStatusCache drops cached rows, e.g. after the launcher itself
// modifies a servers.dat (currently nothing does — the game owns the file —
// but the hook exists so future writers stay correct).
func InvalidateStatusCache() {
	statusCacheMu.Lock()
	statusCache = map[string]statusCacheEntry{}
	statusCacheMu.Unlock()
}
