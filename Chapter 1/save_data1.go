package main

import "os"

func SaveData1(path string, data []byte) error {
	// OpenFile(name string, flag int, perm FileMode) (*File, error)
	// os.O_RDONLY: Read-only access.os.O_WRONLY: Write-only access.os.O_RDWR: Both read and write access.
	// os.O_CREATE: If the file does not exist at the path you provided, create a brand new, empty file.
	// os.O_TRUNC: If the file already exists, completely erase everything inside it
	fp, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)

	if err != nil {
		return err
	}

	defer fp.Close() // defer is a unique Go keyword that delays (defers) the execution of a function until the surrounding function finishes and returns

	_, err = fp.Write(data)

	return err
}

func main() {
	SaveData1(".\\Chapter 1\\test_file.txt", []byte("Test Data"))
}