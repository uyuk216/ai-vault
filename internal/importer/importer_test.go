package importer

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/uyuk216/ai-vault/internal/vault"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

type zipEntry struct {
	name, content string
	mode          os.FileMode
}

func archive(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	w := zip.NewWriter(&buffer)
	for _, entry := range entries {
		h := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		if entry.mode != 0 {
			h.SetMode(entry.mode)
		}
		f, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(f, entry.content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func testVault(t *testing.T) *vault.Vault {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "vault")
	if err := vault.Init(directory); err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Error(err)
		}
	})
	return v
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func githubClient(t *testing.T, data []byte) *http.Client {
	t.Helper()
	return &http.Client{CheckRedirect: safeRedirect, Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Scheme != "https" || req.URL.Host != "api.github.com" || req.Method != "GET" {
			return nil, fmt.Errorf("unexpected request: %s", req.URL)
		}
		if req.Header.Get("Authorization") != "" {
			return nil, errors.New("unexpected credentials")
		}
		if req.Header.Get("User-Agent") != "AI-Vault" {
			return nil, errors.New("missing user agent")
		}
		var body []byte
		switch req.URL.Path {
		case "/repos/example/notes/commits/HEAD":
			body = []byte(`{"sha":"` + testCommit + `"}`)
		case "/repos/example/notes/zipball/" + testCommit:
			body = data
		default:
			return nil, fmt.Errorf("unexpected endpoint: %s", req.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)), Request: req}, nil
	})}
}

func TestImportPinnedCommitAndIdempotence(t *testing.T) {
	v := testVault(t)
	data := archive(t,
		zipEntry{name: "repo-sha/README.md", content: "# Hello\n[[日本語]]"},
		zipEntry{name: "repo-sha/日本語.md", content: "# 日本語\nメモ"},
		zipEntry{name: "repo-sha/tools/run.sh", content: "echo never executed"},
		zipEntry{name: "repo-sha/.github/README.md", content: "hidden"},
	)
	client := githubClient(t, data)
	first, err := importWithClient(context.Background(), v, "https://github.com/example/notes.git", "", client)
	if err != nil {
		t.Fatal(err)
	}
	if first.Source != "https://github.com/example/notes" || first.Commit != testCommit {
		t.Fatalf("bad provenance: %+v", first)
	}
	expected := []string{"sources/example/notes/README.md", "sources/example/notes/日本語.md"}
	if !reflect.DeepEqual(first.Imported, expected) {
		t.Fatalf("imported %v", first.Imported)
	}
	if !reflect.DeepEqual(first.Skipped, []string{"sources/example/notes/.github/README.md"}) {
		t.Fatalf("skipped %v", first.Skipped)
	}
	note, err := v.Read(expected[1])
	if err != nil || note.Content != "# 日本語\nメモ" {
		t.Fatalf("content = %+v, err = %v", note, err)
	}
	second, err := importWithClient(context.Background(), v, "https://github.com/example/notes/", "", client)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Imported) != 0 || len(second.Skipped) != 3 {
		t.Fatalf("not idempotent: %+v", second)
	}
}

func TestConflictsPreflightBeforeAnyWrite(t *testing.T) {
	v := testVault(t)
	if _, err := v.Write("imported/z.md", "keep me", ""); err != nil {
		t.Fatal(err)
	}
	data := archive(t, zipEntry{name: "root/a.md", content: "new"}, zipEntry{name: "root/z.md", content: "different"})
	report, err := importWithClient(context.Background(), v, "https://github.com/example/notes", "imported", githubClient(t, data))
	if err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected conflict, got %v", err)
	}
	if len(report.Imported) != 0 {
		t.Fatalf("partial import: %+v", report)
	}
	entries, err := v.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("unexpected vault: %v, %v", entries, err)
	}
	note, _ := v.Read("imported/z.md")
	if note.Content != "keep me" {
		t.Fatal("overwrote existing content")
	}
}

func TestImportRejectsURLsAndPrefixesBeforeNetwork(t *testing.T) {
	v := testVault(t)
	never := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		t.Error("unexpected network request")
		return nil, errors.New("unexpected request")
	})}
	for _, address := range []string{
		"http://github.com/example/notes", "https://evil.example/example/notes", "https://github.com.evil.example/example/notes",
		"https://github.com:443/example/notes", "https://token@github.com/example/notes", "https://github.com/example/notes/tree/main",
		"https://github.com/example/notes?q=x", "https://github.com/example/notes#readme", "https://github.com/example/%2e%2e", "https://github.com/example/..",
	} {
		t.Run(address, func(t *testing.T) {
			if _, err := importWithClient(context.Background(), v, address, "", never); err == nil {
				t.Fatal("accepted URL")
			}
		})
	}
	for _, prefix := range []string{"../escape", "/absolute", "C:/escape", ".hidden", "safe//bad", "CON", "trailing.", "safe\\bad"} {
		t.Run(prefix, func(t *testing.T) {
			if _, err := importWithClient(context.Background(), v, "https://github.com/example/notes", prefix, never); err == nil {
				t.Fatal("accepted prefix")
			}
		})
	}
}

func TestArchiveRejectsUnsafePathsAndContent(t *testing.T) {
	tests := []struct {
		name    string
		entries []zipEntry
	}{
		{"traversal", []zipEntry{{name: "root/../escape.md"}}},
		{"absolute", []zipEntry{{name: "/root/note.md"}}},
		{"windows drive", []zipEntry{{name: "C:/root/note.md"}}},
		{"backslash", []zipEntry{{name: "root\\..\\escape.md"}}},
		{"empty segment", []zipEntry{{name: "root//note.md"}}},
		{"multiple roots", []zipEntry{{name: "one/a.md"}, {name: "two/b.md"}}},
		{"no root", []zipEntry{{name: "note.md"}}},
		{"duplicate", []zipEntry{{name: "root/a.md"}, {name: "root/a.md"}}},
		{"case collision", []zipEntry{{name: "root/a.md"}, {name: "root/A.md"}}},
		{"directory case collision", []zipEntry{{name: "root/A/one.md"}, {name: "root/a/two.md"}}},
		{"Unicode normalization collision", []zipEntry{{name: "root/が.md"}, {name: "root/か\u3099.md"}}},
		{"Unicode case folding collision", []zipEntry{{name: "root/Σ.md"}, {name: "root/ς.md"}}},
		{"symlink non Markdown", []zipEntry{{name: "root/link", content: "../../escape", mode: os.ModeSymlink | 0777}}},
		{"special file", []zipEntry{{name: "root/fifo", mode: os.ModeNamedPipe | 0600}}},
		{"reserved Windows", []zipEntry{{name: "root/NUL.md"}}},
		{"trailing dot", []zipEntry{{name: "root/dir./note.md"}}},
		{"trailing space", []zipEntry{{name: "root/dir /note.md"}}},
		{"wildcard", []zipEntry{{name: "root/what?.md"}}},
		{"invalid UTF8", []zipEntry{{name: "root/note.md", content: string([]byte{0xff, 0xfe})}}},
		{"NUL content", []zipEntry{{name: "root/note.md", content: "before\x00after"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := readArchive(archive(t, tc.entries...), "imported"); err == nil {
				t.Fatal("accepted unsafe archive")
			}
		})
	}
}

func TestUnsafeArchiveDoesNotImportEarlierValidNotes(t *testing.T) {
	v := testVault(t)
	data := archive(t, zipEntry{name: "root/a.md", content: "valid"}, zipEntry{name: "root/../escape.md", content: "unsafe"})
	report, err := importWithClient(context.Background(), v, "https://github.com/example/notes", "", githubClient(t, data))
	if err == nil || len(report.Imported) != 0 {
		t.Fatalf("report = %+v, err = %v", report, err)
	}
	entries, err := v.List()
	if err != nil || len(entries) != 0 {
		t.Fatalf("archive wrote notes before validation: %v, %v", entries, err)
	}
}

func TestCancellationStopsBeforeWrites(t *testing.T) {
	v := testVault(t)
	ctx, cancel := context.WithCancel(context.Background())
	data := archive(t, zipEntry{name: "root/a.md", content: "valid"})
	client := githubClient(t, data)
	original := client.Transport
	client.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
		response, err := original.RoundTrip(req)
		if strings.Contains(req.URL.Path, "/zipball/") {
			cancel()
		}
		return response, err
	})
	defer cancel()
	report, err := importWithClient(ctx, v, "https://github.com/example/notes", "", client)
	if !errors.Is(err, context.Canceled) || len(report.Imported) != 0 {
		t.Fatalf("report = %+v, err = %v", report, err)
	}
	entries, err := v.List()
	if err != nil || len(entries) != 0 {
		t.Fatalf("canceled import wrote notes: %v, %v", entries, err)
	}
}

func TestArchiveLimits(t *testing.T) {
	for _, tc := range []struct {
		name, file string
		size       uint32
	}{
		{"note bytes", "root/note.md", maxNoteBytes + 1},
		{"all archive bytes including non Markdown", "root/image.bin", maxUncompressedBytes + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := archive(t, zipEntry{name: tc.file, content: "small"})
			central := bytes.Index(data, []byte{'P', 'K', 1, 2})
			if central < 0 {
				t.Fatal("no central directory")
			}
			binary.LittleEndian.PutUint32(data[central+24:central+28], tc.size)
			if _, _, err := readArchive(data, "imported"); err == nil {
				t.Fatal("accepted oversized archive entry")
			}
		})
	}
	t.Run("notes", func(t *testing.T) {
		entries := make([]zipEntry, maxNotes+1)
		for i := range entries {
			entries[i] = zipEntry{name: fmt.Sprintf("root/%04d.md", i)}
		}
		if _, _, err := readArchive(archive(t, entries...), "imported"); err == nil {
			t.Fatal("accepted too many notes")
		}
	})
	t.Run("entries", func(t *testing.T) {
		entries := make([]zipEntry, maxArchiveEntries+1)
		for i := range entries {
			entries[i] = zipEntry{name: fmt.Sprintf("root/%05d.bin", i)}
		}
		if _, _, err := readArchive(archive(t, entries...), "imported"); err == nil {
			t.Fatal("accepted too many entries")
		}
	})
}

func TestPreflightPhysicalConflicts(t *testing.T) {
	for _, tc := range []struct {
		name, physical, target string
		directory              bool
	}{
		{"file ancestor", "imported/folder", "imported/folder/note.md", false},
		{"directory leaf", "imported/note.md", "imported/note.md", true},
		{"directory case", "Imported", "imported/note.md", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := testVault(t)
			physical := filepath.Join(v.Path(), filepath.FromSlash(tc.physical))
			if tc.directory {
				if err := os.MkdirAll(physical, 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.MkdirAll(filepath.Dir(physical), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(physical, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := preflight(v, []importedNote{{tc.target, "new"}}); err == nil {
				t.Fatal("accepted physical conflict")
			}
		})
	}
	t.Run("pending ancestor", func(t *testing.T) {
		v := testVault(t)
		if _, _, err := preflight(v, []importedNote{{"imported/a.md", "a"}, {"imported/a.md/child.md", "b"}}); err == nil {
			t.Fatal("accepted note as directory")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		v := testVault(t)
		if err := os.Symlink(t.TempDir(), filepath.Join(v.Path(), "imported")); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, _, err := preflight(v, []importedNote{{"imported/note.md", "a"}}); err == nil {
			t.Fatal("accepted symlink")
		}
	})
}

func TestSafeRedirects(t *testing.T) {
	for _, address := range []string{"https://api.github.com/x", "https://github.com/x", "https://codeload.github.com/x"} {
		u, _ := url.Parse(address)
		if err := safeRedirect(&http.Request{URL: u}, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, address := range []string{"http://codeload.github.com/x", "https://evil.example/x", "https://github.com:443/x", "https://token@github.com/x", "https://github.com.evil.example/x"} {
		u, _ := url.Parse(address)
		if err := safeRedirect(&http.Request{URL: u}, nil); err == nil {
			t.Fatalf("accepted %s", address)
		}
	}
	u, _ := url.Parse("https://github.com/x")
	if err := safeRedirect(&http.Request{URL: u}, make([]*http.Request, 5)); err == nil {
		t.Fatal("accepted redirect loop")
	}
}

func TestDownloadBoundsAndHTTPFailure(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		length int64
		body   string
	}{
		{"content length", 200, 100, "tiny"},
		{"stream length", 200, -1, strings.Repeat("x", 17)},
		{"rate limited", 403, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, ContentLength: tc.length, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(tc.body)), Request: req}, nil
			})}
			if _, err := download(context.Background(), client, "https://api.github.com/test", 16, "application/zip"); err == nil {
				t.Fatal("accepted response")
			}
		})
	}
}

func TestInvalidCommitNeverDownloadsArchive(t *testing.T) {
	v := testVault(t)
	requests := 0
	client := &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"sha":"main"}`)), Request: req}, nil
	})}
	if _, err := importWithClient(context.Background(), v, "https://github.com/example/notes", "", client); err == nil {
		t.Fatal("accepted mutable commit")
	}
	if requests != 1 {
		t.Fatalf("made %d requests", requests)
	}
}
