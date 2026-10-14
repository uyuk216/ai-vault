// Package vault stores AI-neutral notes as ordinary Markdown files.
package vault

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const MaxNoteBytes = 4 << 20

type Vault struct {
	root      *os.Root
	directory string
}
type Note struct {
	Path     string   `json:"path"`
	Content  string   `json:"content"`
	SHA256   string   `json:"sha256"`
	Modified string   `json:"modified"`
	Tags     []string `json:"tags"`
	Links    []Link   `json:"links"`
}
type Entry struct {
	Path     string   `json:"path"`
	Title    string   `json:"title"`
	SHA256   string   `json:"sha256"`
	Bytes    int64    `json:"bytes"`
	Modified string   `json:"modified"`
	Tags     []string `json:"tags"`
}
type Link struct {
	Target   string `json:"target"`
	Resolved string `json:"resolved"`
	Exists   bool   `json:"exists"`
}
type Hit struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}
type Category struct {
	Path  string `json:"path"`
	Notes int    `json:"notes"`
}
type Revision struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
	Modified string `json:"modified"`
}
type snapshot struct {
	Revision
	Content string `json:"content"`
}

func Init(directory string) error {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	r, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, p := range []string{".aivault", ".aivault/history", ".aivault/tmp"} {
		if st, err := r.Lstat(p); err == nil && (!st.IsDir() || st.Mode()&os.ModeSymlink != 0) {
			return fmt.Errorf("unsafe metadata directory: %s", p)
		}
		if err := r.MkdirAll(p, 0700); err != nil {
			return err
		}
	}
	return nil
}

func Open(directory string) (*Vault, error) {
	abs, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("vault not found; run aivault init: %w", err)
	}
	if !st.IsDir() {
		return nil, errors.New("vault must be a directory")
	}
	if err = Init(abs); err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(abs)
	if err != nil {
		return nil, err
	}
	return &Vault{root: r, directory: abs}, nil
}
func (v *Vault) Close() error { return v.root.Close() }
func (v *Vault) Path() string { return v.directory }

// ValidatePath rejects names that cannot be safely shared across all three OSes.
func ValidatePath(p string) error {
	if len(p) == 0 || len(p) > 4096 || !utf8.ValidString(p) || strings.ContainsAny(p, "\\\x00<>:\"|?*\r\n") || path.IsAbs(p) || path.Clean(p) != p || !strings.EqualFold(path.Ext(p), ".md") {
		return fmt.Errorf("invalid portable Markdown path: %q", p)
	}
	for _, part := range strings.Split(p, "/") {
		if strings.HasPrefix(part, ".") || strings.TrimSpace(part) != part || strings.HasSuffix(part, ".") {
			return fmt.Errorf("hidden, traversal or ambiguous path: %q", p)
		}
		for _, r := range part {
			if r < 32 {
				return errors.New("control character in path")
			}
		}
		base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		deviceSuffix := strings.TrimPrefix(strings.TrimPrefix(base, "COM"), "LPT")
		device := (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && (strings.Contains("0123456789¹²³", deviceSuffix) && len([]rune(deviceSuffix)) == 1)
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || base == "CONIN$" || base == "CONOUT$" || device {
			return fmt.Errorf("Windows reserved name: %q", part)
		}
	}
	return nil
}
func ValidateCategory(category string) error {
	if category == "" {
		return errors.New("category must not be empty")
	}
	return ValidatePath(category + "/_category.md")
}
func (v *Vault) check(p string) error {
	if err := ValidatePath(p); err != nil {
		return err
	}
	parts := strings.Split(p, "/")
	for i := range parts {
		st, err := v.root.Lstat(strings.Join(parts[:i+1], "/"))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlinks are not allowed in vault note paths")
		}
		if i < len(parts)-1 && !st.IsDir() {
			return errors.New("note parent is not a directory")
		}
	}
	return nil
}
func hash(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }
func (v *Vault) read(p string) (Note, error) {
	if err := v.check(p); err != nil {
		return Note{}, err
	}
	f, err := v.root.Open(p)
	if err != nil {
		return Note{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Note{}, err
	}
	if !st.Mode().IsRegular() {
		return Note{}, errors.New("note must be a regular file")
	}
	if st.Size() > MaxNoteBytes {
		return Note{}, errors.New("note exceeds 4 MiB")
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxNoteBytes+1))
	if err != nil {
		return Note{}, err
	}
	if len(b) > MaxNoteBytes || !utf8.Valid(b) {
		return Note{}, errors.New("note must be UTF-8 and at most 4 MiB")
	}
	s := string(b)
	return Note{Path: p, Content: s, SHA256: hash(s), Modified: st.ModTime().UTC().Format(time.RFC3339Nano), Tags: tags(s), Links: []Link{}}, nil
}
func (v *Vault) Read(p string) (Note, error) {
	n, err := v.read(p)
	if err != nil {
		return n, err
	}
	paths, err := v.paths()
	if err != nil {
		return n, err
	}
	n.Links = resolveLinks(p, n.Content, paths)
	return n, nil
}
func (v *Vault) paths() ([]string, error) {
	paths, _, err := v.scan()
	return paths, err
}

// IgnoredPaths reports unsafe external entries skipped by the note index.
func (v *Vault) IgnoredPaths() ([]string, error) { _, ignored, err := v.scan(); return ignored, err }
func (v *Vault) scan() ([]string, []string, error) {
	result := []string{}
	ignored := []string{}
	err := fs.WalkDir(v.root.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			ignored = append(ignored, p)
			return nil
		}
		if d.IsDir() {
			if err := ValidateCategory(p); err != nil {
				ignored = append(ignored, p)
				return fs.SkipDir
			}
			return nil
		}
		if strings.EqualFold(path.Ext(p), ".md") {
			if err := ValidatePath(p); err != nil {
				ignored = append(ignored, p)
				return nil
			}
			result = append(result, p)
		}
		return nil
	})
	sort.Strings(result)
	sort.Strings(ignored)
	return result, ignored, err
}

// Categories is derived from folders rather than a predefined subject taxonomy.
// Counts include notes in each category and its subcategories.
func (v *Vault) Categories() ([]Category, error) {
	paths, err := v.paths()
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	err = fs.WalkDir(v.root.FS(), ".", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if p == "." {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			if ValidateCategory(p) != nil {
				return fs.SkipDir
			}
			counts[p] = 0
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		for dir := path.Dir(p); dir != "."; dir = path.Dir(dir) {
			counts[dir]++
		}
	}
	keys := []string{}
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []Category{}
	for _, k := range keys {
		out = append(out, Category{Path: k, Notes: counts[k]})
	}
	return out, nil
}
func entry(n Note) Entry {
	title := strings.TrimSuffix(path.Base(n.Path), path.Ext(n.Path))
	for _, line := range strings.Split(n.Content, "\n") {
		if strings.HasPrefix(line, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			break
		}
	}
	return Entry{Path: n.Path, Title: title, SHA256: n.SHA256, Bytes: int64(len(n.Content)), Modified: n.Modified, Tags: n.Tags}
}
func (v *Vault) List() ([]Entry, error) {
	paths, err := v.paths()
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, p := range paths {
		n, err := v.read(p)
		if err != nil {
			return nil, err
		}
		out = append(out, entry(n))
	}
	return out, nil
}

func newID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return time.Now().UTC().Format("20060102T150405.000000000") + "-" + hex.EncodeToString(b)
}
func (v *Vault) lock() (func(), error) {
	if err := v.root.Mkdir(".aivault/write.lock", 0700); err != nil {
		return nil, fmt.Errorf("vault is busy or write lock remains; retry after other writers exit, inspect .aivault/write.lock after a crash: %w", err)
	}
	return func() { _ = v.root.Remove(".aivault/write.lock") }, nil
}
func (v *Vault) backup(n Note) (Revision, error) {
	r := Revision{ID: newID(), Path: n.Path, SHA256: n.SHA256, Modified: time.Now().UTC().Format(time.RFC3339Nano)}
	b, err := json.Marshal(snapshot{Revision: r, Content: n.Content})
	if err != nil {
		return r, err
	}
	tmp := ".aivault/tmp/" + r.ID + ".json"
	defer v.root.Remove(tmp)
	if err = v.saveNew(tmp, b); err != nil {
		return r, err
	}
	if err = v.root.Rename(tmp, ".aivault/history/"+r.ID+".json"); err != nil {
		return r, err
	}
	return r, nil
}
func (v *Vault) saveNew(p string, b []byte) error {
	f, err := v.root.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(b)
	syncErr := f.Sync()
	closeErr := f.Close()
	err = errors.Join(writeErr, syncErr, closeErr)
	if err != nil {
		_ = v.root.Remove(p)
	}
	return err
}
func (v *Vault) checkCollisions(p string) error {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		parent := "."
		if i > 0 {
			parent = strings.Join(parts[:i], "/")
		}
		entries, err := fs.ReadDir(v.root.FS(), parent)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.Name() != part && strings.EqualFold(norm.NFC.String(e.Name()), norm.NFC.String(part)) {
				return fmt.Errorf("portable path collision: %s and %s", part, e.Name())
			}
		}
	}
	return nil
}
func (v *Vault) write(p, content, expected string) (Note, error) {
	if err := v.check(p); err != nil {
		return Note{}, err
	}
	if len(content) > MaxNoteBytes || !utf8.ValidString(content) {
		return Note{}, errors.New("content must be UTF-8 and at most 4 MiB")
	}
	if err := v.checkCollisions(p); err != nil {
		return Note{}, err
	}
	n, err := v.read(p)
	if err == nil {
		if expected == "" || n.SHA256 != expected {
			return Note{}, errors.New("conflict: existing note requires its current expected_sha256; read it again")
		}
		if n.Content == content {
			return v.Read(p)
		}
		if _, err = v.backup(n); err != nil {
			return Note{}, err
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		if expected != "" {
			return Note{}, errors.New("conflict: note no longer exists")
		}
	} else {
		return Note{}, err
	}
	if err = v.root.MkdirAll(path.Dir(p), 0700); err != nil {
		return Note{}, err
	}
	tmp := ".aivault/tmp/" + newID()
	defer v.root.Remove(tmp)
	if err = v.saveNew(tmp, []byte(content)); err != nil {
		return Note{}, err
	}
	if err = v.root.Rename(tmp, p); err != nil {
		return Note{}, err
	}
	return v.Read(p)
}
func (v *Vault) Write(p, content, expected string) (Note, error) {
	unlock, err := v.lock()
	if err != nil {
		return Note{}, err
	}
	defer unlock()
	return v.write(p, content, expected)
}
func (v *Vault) Append(p, content, expected string) (Note, error) {
	unlock, err := v.lock()
	if err != nil {
		return Note{}, err
	}
	defer unlock()
	n, err := v.read(p)
	if err != nil {
		return Note{}, err
	}
	if expected == "" || expected != n.SHA256 {
		return Note{}, errors.New("conflict: append requires current expected_sha256")
	}
	return v.write(p, n.Content+content, expected)
}
func (v *Vault) Search(query string, limit int) ([]Hit, error) {
	if strings.TrimSpace(query) == "" || len(query) > 4096 || !utf8.ValidString(query) {
		return nil, errors.New("query must be nonempty UTF-8 and at most 4096 bytes")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	paths, err := v.paths()
	if err != nil {
		return nil, err
	}
	hits := []Hit{}
	q := strings.ToLower(query)
	for _, p := range paths {
		n, err := v.read(p)
		if err != nil {
			return nil, err
		}
		for i, line := range strings.Split(n.Content, "\n") {
			if strings.Contains(strings.ToLower(line), q) {
				r := []rune(line)
				if len(r) > 512 {
					line = string(r[:512]) + "…"
				}
				hits = append(hits, Hit{Path: p, Line: i + 1, Text: line})
				if len(hits) >= limit {
					return hits, nil
				}
			}
		}
	}
	return hits, nil
}

var tagPattern = regexp.MustCompile(`(?:^|[\s])#([\p{L}\p{N}_/-]+)`)

func tags(content string) []string {
	set := map[string]bool{}
	for _, m := range tagPattern.FindAllStringSubmatch(content, -1) {
		set[m[1]] = true
	}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[0] == "---" {
		for _, l := range lines[1:] {
			if l == "---" {
				break
			}
			if strings.HasPrefix(l, "tags:") {
				s := strings.Trim(strings.TrimSpace(strings.TrimPrefix(l, "tags:")), "[]")
				for _, t := range strings.Split(s, ",") {
					t = strings.Trim(strings.TrimSpace(t), "\"'")
					if t != "" {
						set[t] = true
					}
				}
			}
		}
	}
	out := []string{}
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

var wikiPattern = regexp.MustCompile(`!?\[\[([^\]\r\n]+)\]\]`)
var mdPattern = regexp.MustCompile(`!?\[[^\]\r\n]*\]\(([^)\r\n]+)\)`)

func resolveLinks(source, content string, paths []string) []Link {
	known := map[string]bool{}
	for _, p := range paths {
		known[p] = true
	}
	targets := []string{}
	for _, m := range wikiPattern.FindAllStringSubmatch(content, -1) {
		targets = append(targets, strings.SplitN(m[1], "|", 2)[0])
	}
	for _, m := range mdPattern.FindAllStringSubmatch(content, -1) {
		t := strings.TrimSpace(m[1])
		if strings.HasPrefix(t, "<") {
			t = strings.TrimSuffix(strings.SplitN(t, ">", 2)[0], ">")
			t = strings.TrimPrefix(t, "<")
		} else {
			t = strings.SplitN(t, " ", 2)[0]
		}
		targets = append(targets, t)
	}
	out := []Link{}
	seen := map[string]bool{}
	for _, raw := range targets {
		t := strings.SplitN(raw, "#", 2)[0]
		if decoded, err := url.PathUnescape(t); err == nil {
			t = decoded
		}
		if t == "" {
			continue
		}
		u, err := url.Parse(t)
		if err != nil || u.IsAbs() || strings.HasPrefix(t, "//") {
			continue
		}
		if !strings.EqualFold(path.Ext(t), ".md") {
			if path.Ext(t) != "" {
				continue
			}
			t += ".md"
		}
		resolved := ""
		for _, candidate := range []string{path.Clean(path.Join(path.Dir(source), t)), path.Clean(t)} {
			if known[candidate] {
				resolved = candidate
				break
			}
		}
		if resolved == "" && !strings.Contains(t, "/") {
			matches := []string{}
			for _, p := range paths {
				if path.Base(p) == t {
					matches = append(matches, p)
				}
			}
			if len(matches) == 1 {
				resolved = matches[0]
			}
		}
		if seen[raw] {
			continue
		}
		seen[raw] = true
		out = append(out, Link{Target: raw, Resolved: resolved, Exists: resolved != ""})
	}
	return out
}
func (v *Vault) Links(p string) ([]Link, error) { n, err := v.Read(p); return n.Links, err }
func (v *Vault) Backlinks(p string) ([]Entry, error) {
	if err := v.check(p); err != nil {
		return nil, err
	}
	paths, err := v.paths()
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, source := range paths {
		n, err := v.read(source)
		if err != nil {
			return nil, err
		}
		for _, l := range resolveLinks(source, n.Content, paths) {
			if l.Resolved == p {
				out = append(out, entry(n))
				break
			}
		}
	}
	return out, nil
}
func (v *Vault) Trash(p, expected string) (Revision, error) {
	unlock, err := v.lock()
	if err != nil {
		return Revision{}, err
	}
	defer unlock()
	n, err := v.read(p)
	if err != nil {
		return Revision{}, err
	}
	if expected == "" || expected != n.SHA256 {
		return Revision{}, errors.New("conflict: trash requires current expected_sha256")
	}
	r, err := v.backup(n)
	if err != nil {
		return r, err
	}
	return r, v.root.Remove(p)
}
func (v *Vault) History(p string) ([]Revision, error) {
	if err := ValidatePath(p); err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(v.root.FS(), ".aivault/history")
	if err != nil {
		return nil, err
	}
	out := []Revision{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		s, err := v.loadSnapshot(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		if s.Path == p {
			out = append(out, s.Revision)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}
func (v *Vault) loadSnapshot(id string) (snapshot, error) {
	if !regexp.MustCompile(`^\d{8}T\d{6}\.\d{9}-[0-9a-f]{24}$`).MatchString(id) {
		return snapshot{}, errors.New("invalid revision ID")
	}
	f, err := v.root.Open(".aivault/history/" + id + ".json")
	if err != nil {
		return snapshot{}, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 6*MaxNoteBytes+8193))
	if err != nil {
		return snapshot{}, err
	}
	if len(b) > 6*MaxNoteBytes+8192 {
		return snapshot{}, errors.New("history file too large")
	}
	var s snapshot
	if err = json.Unmarshal(b, &s); err != nil {
		return s, err
	}
	if s.ID != id || hash(s.Content) != s.SHA256 {
		return s, errors.New("history checksum mismatch")
	}
	if err = ValidatePath(s.Path); err != nil {
		return s, err
	}
	return s, nil
}
func (v *Vault) Restore(id, destination, expected string) (Note, error) {
	s, err := v.loadSnapshot(id)
	if err != nil {
		return Note{}, err
	}
	return v.Write(destination, s.Content, expected)
}
