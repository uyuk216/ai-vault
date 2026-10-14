package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLISharedMemoryAndConflicts(t *testing.T) {
	p := filepath.Join(t.TempDir(), "日本語 vault")
	call := func(input string, args ...string) (string, error) {
		t.Helper()
		var out, stderr bytes.Buffer
		err := run(append([]string{"--vault", p}, args...), strings.NewReader(input), &out, &stderr)
		return out.String(), err
	}
	if _, err := call("", "init"); err != nil {
		t.Fatal(err)
	}
	if categories, err := call("", "categories"); err != nil || strings.TrimSpace(categories) != "[]" {
		t.Fatalf("category presets %s %v", categories, err)
	}
	if _, err := call("", "write", "方針.md", "--category", "仕事/決定事項", "--content", "分類の共有"); err != nil {
		t.Fatal(err)
	}
	if categories, err := call("", "categories"); err != nil || !strings.Contains(categories, "仕事/決定事項") {
		t.Fatalf("categories %s %v", categories, err)
	}
	if _, err := call("", "write", "escape.md", "--category", "../outside", "--content", "bad"); err == nil {
		t.Fatal("category traversal via CLI")
	}
	created, err := call("# shared\r\nClaudeの記憶 #memory\n", "write", "shared.md")
	if err != nil {
		t.Fatal(err)
	}
	var note struct {
		SHA     string `json:"sha256"`
		Content string `json:"content"`
	}
	if err = json.Unmarshal([]byte(created), &note); err != nil {
		t.Fatal(err)
	}
	if _, err = call("", "write", "shared.md", "--content", "oops"); err == nil {
		t.Fatal("blind overwrite via CLI")
	}
	if _, err = call("", "append", "shared.md", "--content", "Codexの記憶\n", "--expected-sha", note.SHA); err != nil {
		t.Fatal(err)
	}
	read, err := call("", "read", "shared.md")
	if err != nil || !strings.Contains(read, "Codexの記憶") {
		t.Fatalf("read %s %v", read, err)
	}
	search, err := call("", "search", "記憶", "--limit", "1")
	if err != nil {
		t.Fatal(err)
	}
	var hits []any
	if err = json.Unmarshal([]byte(search), &hits); err != nil || len(hits) != 1 {
		t.Fatalf("hits %s %v", search, err)
	}
	config, err := call("", "config")
	if err != nil {
		t.Fatal(err)
	}
	var settings struct {
		Servers map[string]struct {
			Command string
			Args    []string
		} `json:"mcpServers"`
	}
	if err = json.Unmarshal([]byte(config), &settings); err != nil {
		t.Fatal(err)
	}
	server := settings.Servers["ai-vault"]
	if !filepath.IsAbs(server.Command) || len(server.Args) != 3 || server.Args[1] != p {
		t.Fatalf("config %s", config)
	}
	if _, err = call("", "read", "../escape.md"); err == nil {
		t.Fatal("CLI path escape accepted")
	}
	if _, err = call("", "write", "new.md", "--content", "x", "--file", "x"); err == nil {
		t.Fatal("ambiguous input accepted")
	}
	if _, err = call("", "doctor"); err != nil {
		t.Fatal(err)
	}
}
