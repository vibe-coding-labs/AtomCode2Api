//go:build !linux && !darwin
// +build !linux,!darwin

package main

import "syscall"

func procAttr() *syscall.SysProcAttr {
	return nil
}
