package fbhttp

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"

	"github.com/filebrowser/filebrowser/v2/files"
)

var exifHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	if !d.user.Perm.Download {
		return http.StatusForbidden, nil
	}

	file, err := files.NewFileInfo(&files.FileOptions{
		Fs:         d.user.Fs,
		Path:       r.URL.Path,
		Modify:     false,
		Expand:     false,
		ReadHeader: false,
		CalcImgRes: false,
		Token:      "",
		Checker:    d,
		Content:    false,
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	realPath := file.RealPath()
	if realPath == "" {
		return http.StatusNotFound, nil
	}

	// For non-macOS systems this might fail, but this is a macOS-specific deployment
	cmd := exec.Command("sips", "-g", "all", realPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	lines := strings.Split(string(out), "\n")
	exifData := make(map[string]string)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if key != "" && val != "" {
				exifData[key] = val
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	return 0, json.NewEncoder(w).Encode(exifData)
})
