package main

import (
	"fmt"
	"io/ioutil"
	"path"
)

func migrate(pathToOldData, pathToData string) error {
	files, _ := ioutil.ReadDir(pathToOldData)
	for _, f := range files {
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
		p.Save()
	}
	return nil
}
