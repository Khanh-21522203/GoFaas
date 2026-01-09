package storage

import (
	"io/ioutil"
	"os"
)

// SaveFunction saves the function code to a file
func SaveFunction(name string, code string) error {
	filePath := "functions/" + name + ".go"
	return ioutil.WriteFile(filePath, []byte(code), os.ModePerm)
}
