// +build debug

package server

import "github.com/jcelliott/lumber"

func init() {
	hotTemplateReloading = true
  LogLevel = lumber.TRACE
}
