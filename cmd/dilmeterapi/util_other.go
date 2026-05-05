//go:build !windows

package main

func messagebox(msg string) {
	logger.Println("messagebox", msg)
}
