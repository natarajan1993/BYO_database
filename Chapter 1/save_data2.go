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

// dumps the data to a temporary file, then rename the temporary file to the target file
// doesn’t control when the data is persisted to the disk, 
// the metadata (the size of the file) may be persisted to the disk before the data, potentially corrupting the file after when the system crash.
func SaveData2(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.tmp.%d", path, randomInt(10, 50))
	// os.O_EXCL when used together with os.O_CREATE, it ensures that the file must be newly created
	fp, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)

	if err != nil {
		return err
	}
	defer fp.Close()

	_, err = fp.Write(data)

	// If the system crashed before renaming, the original file remains intact
	if err != nil {
		os.Remove(tmp)
		return err
	}
	
	// the rename operation is atomic
	return os.Rename(tmp, path)
}

func main() {
	SaveData2(".\\Chapter 1\\test_file2.txt", []byte("Test Data 2"))
}