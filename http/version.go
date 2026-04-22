package fbhttp

import (
	"net/http"
	"os"
	"strings"
)

var versionHandler = withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	// Root version file in the workspace
	versionPath := "../version.txt"
	content, err := os.ReadFile(versionPath)
	if err != nil {
		return http.StatusNotFound, err
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(strings.TrimSpace(string(content))))
	return 0, nil
})
