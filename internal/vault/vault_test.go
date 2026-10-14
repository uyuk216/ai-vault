package vault

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func testVault(t *testing.T) *Vault {
	t.Helper()
	p := t.TempDir()
	if err := Init(p); err != nil {
		t.Fatal(err)
	}
	v, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { v.Close() })
	return v
}
func TestNotesHistoryTrashRestore(t *testing.T) {
	v := testVault(t)
	text := "---\r\ntags: [共有, memory]\r\n---\r\n# 日本語のメモ\r\nhello #仕事\r\n"
	n, err := v.Write("仕事/共有.md", text, "")
	if err != nil {
		t.Fatal(err)
	}
	if n.Content != text || len(n.SHA256) != 64 || len(n.Tags) != 3 {
		t.Fatalf("unexpected note: %+v", n)
	}
	if _, err = v.Write(n.Path, "lost", ""); err == nil {
		t.Fatal("blind overwrite accepted")
	}
	if _, err = v.Append(n.Path, "lost", "outdated"); err == nil {
		t.Fatal("stale append accepted")
	}
	updated, err := v.Append(n.Path, "Claudeから追記\n", n.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	h, err := v.History(n.Path)
	if err != nil || len(h) != 1 {
		t.Fatalf("history: %v %v", h, err)
	}
	if h[0].SHA256 != n.SHA256 {
		t.Fatal("history not original bytes")
	}
	rev, err := v.Trash(n.Path, updated.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = v.Read(n.Path); !os.IsNotExist(err) {
		t.Fatalf("trash did not remove: %v", err)
	}
	restored, err := v.Restore(rev.ID, n.Path, "")
	if err != nil || restored.Content != updated.Content {
		t.Fatalf("restore: %v", err)
	}
	if _, err = v.Restore("../../escape", n.Path, restored.SHA256); err == nil {
		t.Fatal("unsafe revision accepted")
	}
	if _, err = v.Restore(h[0].ID, n.Path, restored.SHA256); err != nil {
		t.Fatal(err)
	}
	original, err := v.Read(n.Path)
	if err != nil || original.Content != text {
		t.Fatal("original content not recoverable")
	}
}
func TestGraphAndJapaneseSearch(t *testing.T) {
	v := testVault(t)
	for p, s := range map[string]string{"設計.md": "# 設計\n記憶の共有\n", "daily/日記.md": "[[設計#記憶|仕様]] [説明](../設計.md) [[missing]] [web](https://example.com)\n記憶の共有 #日本語\n"} {
		if _, err := v.Write(p, s, ""); err != nil {
			t.Fatal(err)
		}
	}
	links, err := v.Links("daily/日記.md")
	if err != nil || len(links) != 3 || !links[0].Exists || links[1].Exists || !links[2].Exists {
		t.Fatalf("links: %+v %v", links, err)
	}
	backs, err := v.Backlinks("設計.md")
	if err != nil || len(backs) != 1 || backs[0].Path != "daily/日記.md" {
		t.Fatalf("backlinks: %+v %v", backs, err)
	}
	hits, err := v.Search("記憶", 1000)
	if err != nil || len(hits) != 3 || hits[0].Line != 1 {
		t.Fatalf("search %+v %v", hits, err)
	}
	if _, err = v.Search("  ", 20); err == nil {
		t.Fatal("empty query accepted")
	}
}
func TestPortablePathsAndLimits(t *testing.T) {
	v := testVault(t)
	for _, p := range []string{"../x.md", "/tmp/x.md", "a/../../x.md", ".git/config.md", ".aivault/x.md", "foo/.hidden.md", "C:/x.md", `a\x.md`, "CON.md", "LPT1.md", "COM¹.md", "LPT³.md", "NUL/note.md", "x.md ", "x.txt", "a//x.md", "./x.md", "x\n.md"} {
		if _, err := v.Write(p, "bad", ""); err == nil {
			t.Errorf("accepted %q", p)
		}
	}
	if _, err := v.Write("big.md", strings.Repeat("x", MaxNoteBytes+1), ""); err == nil {
		t.Fatal("large write accepted")
	}
	if _, err := v.Write("invalid.md", string([]byte{0xff}), ""); err == nil {
		t.Fatal("non UTF8 accepted")
	}
	if _, err := v.Write("Case.md", "ok", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Write("case.md", "no", ""); err == nil {
		t.Fatal("case collision accepted")
	}
}
func TestCategoriesAreDynamicAndSurviveEmptyFolders(t *testing.T) {
	v := testVault(t)
	categories, err := v.Categories()
	if err != nil || len(categories) != 0 {
		t.Fatalf("unexpected presets %+v %v", categories, err)
	}
	n, err := v.Write("仕事/決定事項/方針.md", "任意の分類", "")
	if err != nil {
		t.Fatal(err)
	}
	categories, err = v.Categories()
	if err != nil || len(categories) != 2 || categories[0].Notes != 1 || categories[1].Path != "仕事/決定事項" {
		t.Fatalf("categories %+v %v", categories, err)
	}
	if _, err = v.Trash(n.Path, n.SHA256); err != nil {
		t.Fatal(err)
	}
	categories, err = v.Categories()
	if err != nil || len(categories) != 2 || categories[0].Notes != 0 {
		t.Fatalf("empty categories lost %+v %v", categories, err)
	}
	if _, err = v.Write("Work/a.md", "first", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = v.Write("work/b.md", "case collision", ""); err == nil {
		t.Fatal("case-variant parent created")
	}
	for _, category := range []string{"../escape", ".hidden", "a//b", "C:/x"} {
		if ValidateCategory(category) == nil {
			t.Fatalf("accepted category %q", category)
		}
	}
}
func TestCrossProcessStyleConflict(t *testing.T) {
	v := testVault(t)
	second, err := Open(v.Path())
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	n, err := v.Write("shared.md", "start", "")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i, writer := range []*Vault{v, second} {
		wg.Add(1)
		go func(v *Vault, i int) {
			defer wg.Done()
			_, err := v.Append("shared.md", string(rune('a'+i)), n.SHA256)
			errs <- err
		}(writer, i)
	}
	wg.Wait()
	close(errs)
	success := 0
	for err := range errs {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("expected one writer, got %d", success)
	}
	after, err := v.Read(n.Path)
	if err != nil || len(after.Content) != 6 {
		t.Fatalf("lost or duplicate write: %+v %v", after, err)
	}
}
func TestLockAndSymlinkConfinement(t *testing.T) {
	v := testVault(t)
	if err := os.Mkdir(filepath.Join(v.Path(), ".aivault", "write.lock"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Write("locked.md", "no", ""); err == nil {
		t.Fatal("ignored cross process lock")
	}
	os.Remove(filepath.Join(v.Path(), ".aivault", "write.lock"))
	outside := t.TempDir()
	external := filepath.Join(outside, "secret.md")
	os.WriteFile(external, []byte("secret"), 0600)
	if err := os.Symlink(external, filepath.Join(v.Path(), "secret.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := v.Read("secret.md"); err == nil {
		t.Fatal("read symlink outside vault")
	}
	if _, err := v.Write("secret.md", "changed", ""); err == nil {
		t.Fatal("wrote symlink outside vault")
	}
	b, _ := os.ReadFile(external)
	if string(b) != "secret" {
		t.Fatal("external file modified")
	}
	if err := os.Symlink(outside, filepath.Join(v.Path(), "outside")); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Write("outside/escape.md", "no", ""); err == nil {
		t.Fatal("directory symlink escaped")
	}
}
func TestHistoryChecksumAndMaximumEscaping(t *testing.T) {
	v := testVault(t)
	content := strings.Repeat("<", MaxNoteBytes)
	n, err := v.Write("escaped.md", content, "")
	if err != nil {
		t.Fatal(err)
	}
	rev, err := v.Trash(n.Path, n.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = v.Restore(rev.ID, n.Path, ""); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(v.Path(), ".aivault", "history", rev.ID+".json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	b = []byte(strings.Replace(string(b), n.SHA256, strings.Repeat("0", 64), 1))
	if err = os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = v.Restore(rev.ID, "tampered.md", ""); err == nil {
		t.Fatal("corrupt history accepted")
	}
}
