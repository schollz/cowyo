package cowyo

import (
	"strings"

	log "github.com/schollz/logger"
)

func pageOperation(operation string) string {
	if operation == "" {
		return "edit"
	}
	return operation
}

func logPageEdit(title, source, operation string, byteCount int) {
	path := "/" + strings.TrimPrefix(title, "/")
	log.Debugf(
		"page edited: path=%q source=%q operation=%q bytes=%d",
		path,
		source,
		operation,
		byteCount,
	)
}
