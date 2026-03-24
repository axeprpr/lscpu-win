//go:build !amd64

package main

func detectFlags() ([]string, string, string, bool) {
	return nil, "", "", false
}
