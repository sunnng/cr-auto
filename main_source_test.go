package main

import (
	"os"
	"strings"
	"testing"
)

func TestMainLoopExitsOnlyOnCommandExit(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(source)
	if strings.Contains(s, "if command.Type == ui.CommandStop {\n\t\t\t\thost.stop()\n\t\t\t\treturn") {
		t.Fatal("CommandStop must not return from main")
	}
	if !strings.Contains(s, "ui.CommandExit") {
		t.Fatal("main must handle CommandExit")
	}
}
