// Package importer copies Markdown from a public GitHub repository into a vault.
// Repository contents are untrusted data and are never executed.
package importer

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/uyuk216/ai-vault/internal/vault"
	"golang.org/x/text/unicode/norm"
)

const (
	maxCompressedBytes   = 32 << 20
	maxUncompressedBytes = 64 << 20
	maxNoteBytes         = 4 << 20
	maxArchiveEntries    = 10000
	maxNotes             = 2000
)

var (
	ownerPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)
	repoPattern   = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,100}$`)
	commitPattern = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
)

// Report identifies the immutable source and each resulting vault-relative path.
// On a write failure, Imported still lists every successfully created note.
type Report struct {
	Source   string   `json:"source"`
	Commit   string   `json:"commit"`
	Imported []string `json:"imported"`
	Skipped  []string `json:"skipped"`
}

// Import downloads the default branch at an immutable commit and imports its
// Markdown. Existing identical notes are skipped; different notes are never
// overwritten. The entire archive and all conflicts are checked before writes.
func Import(ctx context.Context, v *vault.Vault, repoURL, prefix string) (Report, error) {
	return importWithClient(ctx, v, repoURL, prefix, &http.Client{
		Timeout:       90 * time.Second,
		CheckRedirect: safeRedirect,
	})
}

func importWithClient(ctx context.Context, v *vault.Vault, repoURL, prefix string, client *http.Client) (Report, error) {
	report := Report{Imported: []string{}, Skipped: []string{}}
	owner, repo, err := parseRepository(repoURL)
	if err != nil {
		return report, err
	}
	report.Source = "https://github.com/" + owner + "/" + repo
	if prefix == "" {
		prefix = "sources/" + owner + "/" + repo
	}
	if err := portablePath(prefix, false); err != nil {
		return report, fmt.Errorf("invalid import prefix: %w", err)
	}
	base := "https://api.github.com/repos/" + owner + "/" + repo
	response, err := download(ctx, client, base+"/commits/HEAD", 1<<20, "application/vnd.github+json")
	if err != nil {
		return report, fmt.Errorf("resolve public GitHub repository: %w", err)
	}
	var commit struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(response, &commit); err != nil || !commitPattern.MatchString(commit.SHA) {
		return report, errors.New("GitHub returned an invalid commit")
	}
	report.Commit = strings.ToLower(commit.SHA)
	archive, err := download(ctx, client, base+"/zipball/"+report.Commit, maxCompressedBytes, "application/zip")
	if err != nil {
		return report, fmt.Errorf("download GitHub archive: %w", err)
	}
	notes, skipped, err := readArchive(archive, prefix)
	if err != nil {
		return report, err
	}
	report.Skipped = append(report.Skipped, skipped...)
	pending, identical, err := preflight(v, notes)
	if err != nil {
		return report, err
	}
	report.Skipped = append(report.Skipped, identical...)
	sort.Strings(report.Skipped)
	for _, note := range pending {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		if _, err := v.Write(note.path, note.content, ""); err != nil {
			return report, fmt.Errorf("create %s (already imported %d notes): %w", note.path, len(report.Imported), err)
		}
		report.Imported = append(report.Imported, note.path)
	}
	return report, nil
}

func parseRepository(raw string) (string, string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, "github.com") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.EscapedPath() != u.Path {
		return "", "", errors.New("use a public repository URL such as https://github.com/owner/repo (no branch, query, or credentials)")
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), "/"), "/")
	if len(parts) != 2 {
		return "", "", errors.New("GitHub import accepts repository root URLs only")
	}
	owner, repo := parts[0], strings.TrimSuffix(parts[1], ".git")
	if !ownerPattern.MatchString(owner) || !repoPattern.MatchString(repo) || repo == "." || repo == ".." {
		return "", "", errors.New("invalid GitHub owner or repository name")
	}
	return owner, repo, nil
}

func safeRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return errors.New("too many GitHub redirects")
	}
	if req.URL.Scheme != "https" || req.URL.User != nil {
		return errors.New("GitHub redirect must use HTTPS without credentials")
	}
	switch strings.ToLower(req.URL.Host) {
	case "api.github.com", "github.com", "codeload.github.com":
		return nil
	default:
		return errors.New("GitHub redirect points outside official GitHub hosts")
	}
}

func download(ctx context.Context, client *http.Client, address string, limit int64, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "AI-Vault")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub HTTP %d (repository must be public; unauthenticated API rate limits apply)", resp.StatusCode)
	}
	if resp.ContentLength > limit {
		return nil, fmt.Errorf("download exceeds %d bytes", limit)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download exceeds %d bytes", limit)
	}
	return data, nil
}

type importedNote struct{ path, content string }

func readArchive(data []byte, prefix string) ([]importedNote, []string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid GitHub ZIP archive: %w", err)
	}
	if len(zr.File) == 0 || len(zr.File) > maxArchiveEntries {
		return nil, nil, fmt.Errorf("archive must contain 1 to %d entries", maxArchiveEntries)
	}
	var total uint64
	root := ""
	seen := map[string]string{}
	notes := []importedNote{}
	skipped := []string{}
	for _, f := range zr.File {
		name := strings.TrimSuffix(f.Name, "/")
		if err := archivePath(name); err != nil {
			return nil, nil, fmt.Errorf("unsafe archive entry %q: %w", f.Name, err)
		}
		parts := strings.SplitN(name, "/", 2)
		if root == "" {
			root = parts[0]
		}
		if parts[0] != root {
			return nil, nil, errors.New("archive contains multiple root directories")
		}
		if f.Mode()&os.ModeSymlink != 0 || (f.Mode().Type() != 0 && !f.Mode().IsDir()) {
			return nil, nil, fmt.Errorf("archive contains a symlink or special file: %s", f.Name)
		}
		if f.UncompressedSize64 > maxUncompressedBytes-total {
			return nil, nil, fmt.Errorf("archive exceeds %d uncompressed bytes", maxUncompressedBytes)
		}
		total += f.UncompressedSize64
		if f.FileInfo().IsDir() {
			continue
		}
		if len(parts) != 2 {
			return nil, nil, errors.New("archive file has no repository root directory")
		}
		rel := parts[1]
		if !strings.EqualFold(path.Ext(rel), ".md") {
			continue
		}
		if hiddenPath(rel) {
			skipped = append(skipped, prefix+"/"+rel)
			continue
		}
		destination := prefix + "/" + rel
		if err := portablePath(destination, true); err != nil {
			return nil, nil, fmt.Errorf("unsupported Markdown path %q: %w", rel, err)
		}
		if err := registerPath(seen, destination); err != nil {
			return nil, nil, err
		}
		if f.UncompressedSize64 > maxNoteBytes {
			return nil, nil, fmt.Errorf("Markdown file exceeds %d bytes: %s", maxNoteBytes, rel)
		}
		r, err := f.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", rel, err)
		}
		content, readErr := io.ReadAll(io.LimitReader(r, maxNoteBytes+1))
		closeErr := r.Close()
		if readErr != nil {
			return nil, nil, fmt.Errorf("read %s: %w", rel, readErr)
		}
		if closeErr != nil {
			return nil, nil, closeErr
		}
		if len(content) > maxNoteBytes {
			return nil, nil, fmt.Errorf("Markdown file too large: %s", rel)
		}
		if !utf8.Valid(content) || bytes.ContainsRune(content, 0) {
			return nil, nil, fmt.Errorf("Markdown must be UTF-8 text without NUL bytes: %s", rel)
		}
		notes = append(notes, importedNote{destination, string(content)})
		if len(notes) > maxNotes {
			return nil, nil, fmt.Errorf("archive exceeds %d Markdown notes", maxNotes)
		}
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].path < notes[j].path })
	return notes, skipped, nil
}

func archivePath(value string) error {
	if value == "" || !utf8.ValidString(value) || strings.ContainsAny(value, "\\\x00") || strings.HasPrefix(value, "/") || len(value) > 4096 {
		return errors.New("invalid relative path")
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.Contains(segment, ":") {
			return errors.New("traversal or absolute path")
		}
	}
	return nil
}

func hiddenPath(value string) bool {
	for _, segment := range strings.Split(value, "/") {
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}
	return false
}

func portablePath(value string, note bool) error {
	if err := archivePath(value); err != nil {
		return err
	}
	if hiddenPath(value) {
		return errors.New("hidden paths are unsupported")
	}
	if note && !strings.EqualFold(path.Ext(value), ".md") {
		return errors.New("notes must use .md")
	}
	for _, segment := range strings.Split(value, "/") {
		if len(segment) > 255 || strings.HasSuffix(segment, ".") || strings.TrimSpace(segment) != segment || strings.ContainsAny(segment, `<>"|?*`) {
			return errors.New("path is not portable across Windows, Linux, and macOS")
		}
		for _, r := range segment {
			if unicode.IsControl(r) {
				return errors.New("control character in path")
			}
		}
		base := strings.ToUpper(strings.SplitN(segment, ".", 2)[0])
		switch base {
		case "CON", "PRN", "AUX", "NUL", "CLOCK$", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9", "COM¹", "COM²", "COM³", "LPT¹", "LPT²", "LPT³":
			return errors.New("Windows reserved filename")
		}
	}
	if note {
		return vault.ValidatePath(value)
	}
	return vault.ValidatePath(value + "/aivault-import.md")
}

// registerPath also compares directory spelling: A/x.md and a/y.md would merge
// on default Windows/macOS filesystems and diverge on Linux.
func registerPath(seen map[string]string, value string) error {
	segments := strings.Split(value, "/")
	for i := 1; i <= len(segments); i++ {
		part := strings.Join(segments[:i], "/")
		key := collisionKey(part)
		if previous, exists := seen[key]; exists && (previous != part || i == len(segments)) {
			return fmt.Errorf("archive has duplicate or case-colliding paths: %s and %s", previous, part)
		}
		seen[key] = part
	}
	return nil
}

// SimpleFold handles Unicode case pairs that lowercase alone misses; NFC also
// detects decomposed filenames that alias on common macOS filesystems.
func collisionKey(value string) string {
	return strings.Map(func(r rune) rune {
		minimum := r
		for folded := unicode.SimpleFold(r); folded != r; folded = unicode.SimpleFold(folded) {
			if folded < minimum {
				minimum = folded
			}
		}
		return minimum
	}, norm.NFC.String(value))
}

func sameSpelling(a, b string) bool { return norm.NFC.String(a) == norm.NFC.String(b) }

func preflight(v *vault.Vault, notes []importedNote) ([]importedNote, []string, error) {
	entries, err := v.List()
	if err != nil {
		return nil, nil, err
	}
	root, err := os.OpenRoot(v.Path())
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()
	directoryCache := map[string][]os.DirEntry{}
	existing := map[string]string{}
	allPaths := map[string]string{}
	for _, entry := range entries {
		existing[collisionKey(entry.Path)] = entry.Path
		parts := strings.Split(entry.Path, "/")
		for i := 1; i <= len(parts); i++ {
			part := strings.Join(parts[:i], "/")
			key := collisionKey(part)
			if old, ok := allPaths[key]; ok && !sameSpelling(old, part) {
				return nil, nil, fmt.Errorf("existing vault has case-colliding paths: %s and %s", old, part)
			}
			allPaths[key] = part
		}
	}
	pending := []importedNote{}
	skipped := []string{}
	for _, note := range notes {
		if err := checkFilesystem(root, note.path, directoryCache); err != nil {
			return nil, nil, err
		}
		parts := strings.Split(note.path, "/")
		for i := 1; i <= len(parts); i++ {
			part := strings.Join(parts[:i], "/")
			key := collisionKey(part)
			if old, ok := allPaths[key]; ok && !sameSpelling(old, part) {
				return nil, nil, fmt.Errorf("import path conflicts with vault path: %s and %s", old, part)
			}
			if i < len(parts) {
				if old, ok := existing[key]; ok {
					return nil, nil, fmt.Errorf("import directory conflicts with note: %s", old)
				}
			}
			allPaths[key] = part
		}
		key := collisionKey(note.path)
		for previous := range existing {
			if strings.HasPrefix(previous, key+"/") {
				return nil, nil, fmt.Errorf("import note conflicts with existing directory: %s", note.path)
			}
		}
		if old, found := existing[key]; found {
			current, err := v.Read(old)
			if err != nil {
				return nil, nil, err
			}
			if current.Content != note.content {
				return nil, nil, fmt.Errorf("import conflicts with existing note: %s; use another prefix or resolve it explicitly", note.path)
			}
			skipped = append(skipped, note.path)
		} else {
			pending = append(pending, note)
		}
		// Include pending files in ancestor checks before any mutation.
		existing[key] = note.path
	}
	return pending, skipped, nil
}

// The vault index lists Markdown only. Also check physical directories and
// non-Markdown files so such conflicts cannot cause an avoidable partial import.
func checkFilesystem(root *os.Root, destination string, cache map[string][]os.DirEntry) error {
	parts := strings.Split(destination, "/")
	parent := "."
	for i, segment := range parts {
		entries, cached := cache[parent]
		if !cached {
			dir, err := root.Open(filepath.FromSlash(parent))
			if err != nil {
				return fmt.Errorf("inspect import directory %s: %w", parent, err)
			}
			entries, err = dir.ReadDir(-1)
			dir.Close()
			if err != nil {
				return err
			}
			cache[parent] = entries
		}
		found := false
		for _, entry := range entries {
			if collisionKey(entry.Name()) == collisionKey(segment) {
				if !sameSpelling(entry.Name(), segment) {
					return fmt.Errorf("import path case conflicts with existing filesystem entry: %s and %s", entry.Name(), segment)
				}
				found = true
			}
		}
		if !found {
			return nil
		}
		component := strings.Join(parts[:i+1], "/")
		info, err := root.Lstat(filepath.FromSlash(component))
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("import path contains a symlink: %s", component)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("import directory conflicts with a file: %s", component)
		}
		if i == len(parts)-1 && !info.Mode().IsRegular() {
			return fmt.Errorf("import note conflicts with a directory or special file: %s", component)
		}
		parent = component
	}
	return nil
}
