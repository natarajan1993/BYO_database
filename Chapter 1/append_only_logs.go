package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

func LogCreate(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
}

func LogAppend(fp *os.File, line string) error {
	// "2006-01-02 15:04:05" is a special layouts format code Go uses for layout rules
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fullLine := fmt.Sprintf("[%s] %s\n", timestamp, line)

	buf := []byte(fullLine)
	buf = append(buf, '\n')
	_, err := fp.Write(buf)

	if err != nil {
		return err
	}

	// Sync() forces the operating system to flush that cache and write the data onto the actual physical disk right now.
	return fp.Sync()
}

func main() {
	file, err := LogCreate("app.log")

	if err != nil {
		log.Fatalf("Critical Error: Could not open/create log file: %v", err)
	}

	// Close file we opened using defer
	defer file.Close()

	fmt.Println("Successfully created and opened log file. Begin log writing")

	err = LogAppend(file, "Application started")

	if err != nil {
		log.Printf("Failed to write log: %v", err)
	}

	// Add more logs as you need here
	LogAppend(file, "User logged in.")
	LogAppend(file, "Application Shutting Down...")
}
