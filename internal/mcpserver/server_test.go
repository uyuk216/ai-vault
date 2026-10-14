package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/uyuk216/ai-vault/internal/vault"
)

// The subprocess runs the public Run entrypoint over actual OS stdio pipes.
// os.Exit avoids the test runner printing PASS into the MCP stdout stream.
func TestMCPHelperProcess(t *testing.T) {
	if os.Getenv("AIVAULT_MCP_TEST_HELPER") != "1" {
		return
	}
	v, err := vault.Open(os.Getenv("AIVAULT_MCP_TEST_VAULT"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	err = Run(context.Background(), v, "test", os.Getenv("AIVAULT_MCP_TEST_READ_ONLY") == "1")
	closeErr := v.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if closeErr != nil {
		fmt.Fprintln(os.Stderr, closeErr)
		os.Exit(2)
	}
	os.Exit(0)
}

func TestStdioSharedVaultAndConflictProtection(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "共有 vault")
	if err := vault.Init(directory); err != nil {
		t.Fatal(err)
	}
	ctx, first := connectStdio(t, directory, false)
	_, second := connectStdio(t, directory, false)
	if got := first.InitializeResult().ProtocolVersion; got != "2026-07-28" {
		t.Fatalf("negotiated protocol = %q, want current protocol", got)
	}
	if !strings.Contains(first.InitializeResult().Instructions, "untrusted") {
		t.Fatal("server instructions must identify note content as untrusted")
	}
	tools, err := first.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 12 {
		t.Fatalf("tool count = %d, want 12", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		if tool.Annotations == nil || tool.Annotations.OpenWorldHint == nil || tool.Annotations.DestructiveHint == nil {
			t.Fatalf("%s is missing safety annotations", tool.Name)
		}
		if *tool.Annotations.OpenWorldHint != (tool.Name == "vault_import") {
			t.Fatalf("%s open-world hint is incorrect", tool.Name)
		}
	}
	var created noteOutput
	callOK(t, ctx, first, "vault_write", map[string]any{"path": "notes/共有.md", "content": "# 共有\n日本語の記憶 #memory\n"}, &created)
	if len(created.Note.SHA256) != 64 {
		t.Fatalf("invalid create hash: %q", created.Note.SHA256)
	}
	var shared noteOutput
	callOK(t, ctx, second, "vault_read", map[string]any{"path": "notes/共有.md"}, &shared)
	if shared.Note.SHA256 != created.Note.SHA256 {
		t.Fatal("second client cannot see the shared note")
	}
	var appended noteOutput
	callOK(t, ctx, second, "vault_append", map[string]any{"path": "notes/共有.md", "content": "追加\n", "expected_sha256": shared.Note.SHA256}, &appended)
	callError(t, ctx, first, "vault_write", map[string]any{"path": "notes/共有.md", "content": "lost update", "expected_sha256": created.Note.SHA256})
	callError(t, ctx, first, "vault_write", map[string]any{"path": "notes/共有.md", "content": "blind overwrite"})
	var afterConflict noteOutput
	callOK(t, ctx, first, "vault_read", map[string]any{"path": "notes/共有.md"}, &afterConflict)
	if afterConflict.Note.Content != appended.Note.Content || !strings.Contains(afterConflict.Note.Content, "追加") {
		t.Fatal("a rejected mutation damaged the shared note")
	}
	callOK(t, ctx, first, "vault_write", map[string]any{"path": "index.md", "content": "[[notes/共有]]\n"}, nil)
	var links linksOutput
	callOK(t, ctx, first, "vault_links", map[string]any{"path": "index.md"}, &links)
	if len(links.Links) != 1 || !links.Links[0].Exists || links.Links[0].Resolved != "notes/共有.md" {
		t.Fatalf("links = %#v", links.Links)
	}
	var backlinks listOutput
	callOK(t, ctx, first, "vault_backlinks", map[string]any{"path": "notes/共有.md"}, &backlinks)
	if len(backlinks.Notes) != 1 || backlinks.Notes[0].Path != "index.md" {
		t.Fatalf("backlinks = %#v", backlinks.Notes)
	}
	var found searchOutput
	callOK(t, ctx, first, "vault_search", map[string]any{"query": "日本語", "limit": 1}, &found)
	if len(found.Hits) != 1 || found.Hits[0].Line != 2 {
		t.Fatalf("search = %#v", found.Hits)
	}
	var listed listOutput
	callOK(t, ctx, first, "vault_list", map[string]any{}, &listed)
	if len(listed.Notes) != 2 {
		t.Fatalf("listed notes = %#v", listed.Notes)
	}
	var history historyOutput
	callOK(t, ctx, first, "vault_history", map[string]any{"path": "notes/共有.md"}, &history)
	if len(history.Revisions) == 0 {
		t.Fatal("append did not save prior content")
	}
	var trashed revisionOutput
	callOK(t, ctx, first, "vault_trash", map[string]any{"path": "notes/共有.md", "expected_sha256": appended.Note.SHA256}, &trashed)
	callError(t, ctx, second, "vault_read", map[string]any{"path": "notes/共有.md"})
	var restored noteOutput
	callOK(t, ctx, second, "vault_restore", map[string]any{"id": trashed.Revision.ID, "destination": "notes/共有.md"}, &restored)
	if restored.Note.Content != appended.Note.Content {
		t.Fatal("restore did not recover the trashed content")
	}
}

func TestStdioReadOnlyAndInputRejection(t *testing.T) {
	directory := t.TempDir()
	if err := vault.Init(directory); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	largeContent := strings.Repeat("日本語", 15000) // Exceeds scanners' usual 64 KiB limit.
	if _, err := v.Write("large.md", largeContent, ""); err != nil {
		t.Fatal(err)
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, readOnly := connectStdio(t, directory, true)
	tools, err := readOnly.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 7 {
		t.Fatalf("read-only tool count = %d, want 7", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Fatalf("read-only server registered %s", tool.Name)
		}
	}
	if _, err := readOnly.CallTool(ctx, &mcp.CallToolParams{Name: "vault_write", Arguments: map[string]any{"path": "blocked.md", "content": "blocked"}}); err == nil {
		t.Fatal("read-only server accepted an unregistered mutation tool")
	}
	var large noteOutput
	callOK(t, ctx, readOnly, "vault_read", map[string]any{"path": "large.md"}, &large)
	if large.Note.Content != largeContent {
		t.Fatal("stdio truncated a note larger than 64 KiB")
	}
	callError(t, ctx, readOnly, "vault_read", map[string]any{"path": "../outside.md"})
	callError(t, ctx, readOnly, "vault_search", map[string]any{"query": "x", "limit": 101})
	callError(t, ctx, readOnly, "vault_search", map[string]any{"query": strings.Repeat("x", maxArgumentBytes+1)})
	badType, err := readOnly.CallTool(ctx, &mcp.CallToolParams{Name: "vault_read", Arguments: map[string]any{"path": 123}})
	if err == nil && !badType.IsError {
		t.Fatal("wrong argument type was not rejected by the SDK schema")
	}
	_, writable := connectStdio(t, directory, false)
	callError(t, ctx, writable, "vault_append", map[string]any{"path": "large.md", "content": "x", "expected_sha256": "invalid"})
	// A non-GitHub URL must fail validation without making a network request.
	callError(t, ctx, writable, "vault_import", map[string]any{"url": "file:///etc/passwd"})
	if _, err := os.Stat(filepath.Join(directory, "blocked.md")); !os.IsNotExist(err) {
		t.Fatalf("read-only mutation affected disk: %v", err)
	}
}

func TestStdioDynamicCategories(t *testing.T) {
	directory := t.TempDir()
	if err := vault.Init(directory); err != nil {
		t.Fatal(err)
	}
	ctx, readOnly := connectStdio(t, directory, true)
	var initial categoriesOutput
	callOK(t, ctx, readOnly, "vault_categories", map[string]any{}, &initial)
	if len(initial.Categories) != 0 {
		t.Fatalf("new vault must have no preset categories: %#v", initial.Categories)
	}
	_, writable := connectStdio(t, directory, false)
	var created noteOutput
	callOK(t, ctx, writable, "vault_write", map[string]any{
		"path": "実験.md", "category": "調査/研究メモ", "content": "# 実験\n観測結果\n",
	}, &created)
	if created.Note.Path != "調査/研究メモ/実験.md" {
		t.Fatalf("categorized path = %q", created.Note.Path)
	}
	var categories categoriesOutput
	callOK(t, ctx, readOnly, "vault_categories", map[string]any{}, &categories)
	assertCategory(t, categories.Categories, "調査/研究メモ", 1)
	var shared noteOutput
	callOK(t, ctx, readOnly, "vault_read", map[string]any{"path": created.Note.Path}, &shared)
	if shared.Note.Content != created.Note.Content {
		t.Fatal("read-only client cannot read a categorized note saved by another client")
	}
	callOK(t, ctx, writable, "vault_write", map[string]any{
		"path": "測定.md", "category": "調査/研究メモ/気象", "content": "# 測定\n",
	}, nil)
	callOK(t, ctx, writable, "vault_write", map[string]any{
		"path": "あとで整理.md", "category": "未分類", "content": "分類を後で決める\n",
	}, nil)
	callOK(t, ctx, readOnly, "vault_categories", map[string]any{}, &categories)
	assertCategory(t, categories.Categories, "調査/研究メモ/気象", 1)
	assertCategory(t, categories.Categories, "未分類", 1)
	for _, arguments := range []map[string]any{
		{"path": "../逃走.md", "category": "調査", "content": "blocked"},
		{"path": "逃走.md", "category": "調査/../外", "content": "blocked"},
		{"path": "逃走.md", "category": "/外", "content": "blocked"},
		{"path": "逃走.md", "category": "CON", "content": "blocked"},
	} {
		callError(t, ctx, writable, "vault_write", arguments)
	}
	var all listOutput
	callOK(t, ctx, readOnly, "vault_list", map[string]any{}, &all)
	if len(all.Notes) != 3 {
		t.Fatalf("rejected categorized writes affected the vault: %#v", all.Notes)
	}
}

func assertCategory(t *testing.T, categories []vault.Category, categoryPath string, notes int) {
	t.Helper()
	for _, category := range categories {
		if category.Path == categoryPath {
			if category.Notes != notes {
				t.Fatalf("category %q has %d notes, want %d", categoryPath, category.Notes, notes)
			}
			return
		}
	}
	t.Fatalf("category %q missing from %#v", categoryPath, categories)
}

func TestLegacyStdioHandshake(t *testing.T) {
	directory := t.TempDir()
	if err := vault.Init(directory); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	cmd := helperCommand(ctx, directory, true)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = stdin.Close()
		if err := cmd.Wait(); err != nil {
			t.Errorf("legacy subprocess exit: %v; stderr: %s", err, stderr.String())
		}
	})
	enc, dec := json.NewEncoder(stdin), json.NewDecoder(stdout)
	if err := enc.Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "legacy-test", "version": "1"}}}); err != nil {
		t.Fatal(err)
	}
	var initialized struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := dec.Decode(&initialized); err != nil {
		t.Fatal(err)
	}
	if initialized.Error != nil || initialized.Result.ProtocolVersion != "2025-11-25" {
		t.Fatalf("legacy initialize = %#v", initialized)
	}
	if err := enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"}); err != nil {
		t.Fatal(err)
	}
	if err := enc.Encode(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	var listed struct {
		Result struct {
			Tools []mcp.Tool `json:"tools"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := dec.Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if listed.Error != nil || len(listed.Result.Tools) != 7 {
		t.Fatalf("legacy list = %#v", listed)
	}
}

func TestValidationBounds(t *testing.T) {
	if _, err := validateWrite("ok.md", strings.Repeat("x", maxContentBytes+1), "", false); err == nil {
		t.Fatal("oversized content accepted")
	}
	if _, err := validateWrite("ok.md", string([]byte{0xff}), "", false); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
	if _, err := validateWrite("ok.md", "x", "", true); err == nil {
		t.Fatal("append without an expected hash accepted")
	}
	if err := validatePath(strings.Repeat("x", maxArgumentBytes+1)); err == nil {
		t.Fatal("oversized path accepted")
	}
	if err := validatePath("bad\x00.md"); err == nil {
		t.Fatal("NUL path accepted")
	}
	if got, err := validateSHA(strings.Repeat("A", 64), true); err != nil || got != strings.Repeat("a", 64) {
		t.Fatalf("SHA normalization = %q, %v", got, err)
	}
	if err := Run(context.Background(), nil, "test", false); err == nil {
		t.Fatal("nil vault accepted")
	}
}

func helperCommand(ctx context.Context, directory string, readOnly bool) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMCPHelperProcess$")
	cmd.Env = append(os.Environ(), "AIVAULT_MCP_TEST_HELPER=1", "AIVAULT_MCP_TEST_VAULT="+directory)
	if readOnly {
		cmd.Env = append(cmd.Env, "AIVAULT_MCP_TEST_READ_ONLY=1")
	}
	cmd.WaitDelay = 2 * time.Second
	return cmd
}

func connectStdio(t *testing.T, directory string, readOnly bool) (context.Context, *mcp.ClientSession) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	cmd := helperCommand(ctx, directory, readOnly)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	client := mcp.NewClient(&mcp.Implementation{Name: "ai-vault-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd, TerminateDuration: 3 * time.Second}, nil)
	if err != nil {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
		t.Fatalf("MCP stdio connect: %v; stderr: %s", err, stderr.String())
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("MCP stdio close: %v; stderr: %s", err, stderr.String())
		}
	})
	return ctx, session
}

func callOK(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, arguments map[string]any, output any) {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("%s protocol error: %v", name, err)
	}
	if result.IsError {
		t.Fatalf("%s tool error: %s", name, toolText(result))
	}
	if output == nil {
		return
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, output); err != nil {
		t.Fatalf("%s structured output: %v (%s)", name, err, data)
	}
}

func callError(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, arguments map[string]any) {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("%s should return a readable tool error, got protocol error: %v", name, err)
	}
	if !result.IsError || strings.TrimSpace(toolText(result)) == "" {
		t.Fatalf("%s did not return a readable tool error: %#v", name, result)
	}
}

func toolText(result *mcp.CallToolResult) string {
	var text strings.Builder
	for _, content := range result.Content {
		if item, ok := content.(*mcp.TextContent); ok {
			text.WriteString(item.Text)
		}
	}
	return text.String()
}
