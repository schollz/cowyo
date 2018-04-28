package server

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/jcelliott/lumber"
)

func Migrate(pathToOldData, pathToData string, logger *lumber.ConsoleLogger) error {
	log = logger
	files, err := ioutil.ReadDir(pathToOldData)
	if len(files) == 0 {
		return err
	}
	for _, f := range files {
		if f.Mode().IsDir() {
			continue
		}
		fmt.Printf("Migrating %s", f.Name())
		p := Open(f.Name())
		bData, err := ioutil.ReadFile(path.Join(pathToOldData, f.Name()))
		if err != nil {
			return err
		}
		err = p.Update(string(bData))
		if err != nil {
			return err
		}
		if err = p.Save(); err != nil {
			return err
		}
	}
	return nil
}
