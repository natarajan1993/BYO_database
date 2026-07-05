package main

import (
			"os"
			"fmt"
			"math/rand/v2"
		)

func randomInt(min, max int) int {
	data := min + rand.IntN(max-min)
	return data
}

func SaveData3(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.tmp.%d", path, randomInt(10, 50))
	fp, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)

	if err != nil {
		return err
	}
	defer fp.Close()

	_, err = fp.Write(data)

	if err != nil {
		os.Remove(tmp)
		return err
	}

	// we must flush the data to the disk before renaming it
	err = fp.Sync() // fsync

	if err != nil {
		os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, path)
}

func main() {
	SaveData3(".\\Chapter 1\\test_file3.txt", []byte("Test Data 3"))
}