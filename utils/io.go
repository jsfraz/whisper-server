package utils

import (
	"io"
	"os"
)

// Returns string content of file.
//
//	@param path
//	@return string
//	@return error
func ReadFile(path string) (*string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	str := string(bytes)
	return &str, nil
}
