// aivault provides a headless Markdown knowledge store through CLI and MCP.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/uyuk216/ai-vault/internal/importer"
	"github.com/uyuk216/ai-vault/internal/mcpserver"
	"github.com/uyuk216/ai-vault/internal/vault"
)

var version = "dev"

const usage = `AI Vault — shared Markdown memory for AI, without a GUI.
Usage: aivault [--vault PATH] COMMAND [arguments]

  init                              initialize a vault (default ~/AI-Vault)
  list                              list notes with hashes and tags
  read PATH                         read a note and its current SHA-256
  write PATH --content TEXT         create a note; update with --expected-sha HASH
  append PATH --content TEXT --expected-sha HASH
  search QUERY [--limit 20]          literal text search, including Japanese
  links PATH | backlinks PATH       inspect note links
  tags                              count tags
  categories                        list existing category/subcategory folders
  trash PATH --expected-sha HASH     move a note into recoverable history
  history PATH                      list previous revisions
  restore ID DEST [--expected-sha HASH]
  import GITHUB_URL [--prefix PATH]  import public GitHub Markdown safely
  doctor                            report vault status
  config [--format json|codex]       print MCP configuration
  serve [--read-only]                MCP over stdin/stdout; no network listener
  version                           print build version

write/append also accept --file FILE, or UTF-8 content from stdin.
write accepts --category KIND/SUBKIND to create a category when saving.
All paths are relative Markdown paths; flags for each command follow arguments.
Set AIVAULT_HOME or pass --vault before COMMAND to share one vault across AIs.
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		os.Exit(1)
	}
}

func run(args []string, in io.Reader, out, errOut io.Writer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	directory := os.Getenv("AIVAULT_HOME")
	if directory == "" {
		directory = filepath.Join(home, "AI-Vault")
	}
	global := flag.NewFlagSet("aivault", flag.ContinueOnError)
	global.SetOutput(errOut)
	global.StringVar(&directory, "vault", directory, "absolute vault directory")
	showVersion := global.Bool("version", false, "show version")
	global.Usage = func() { fmt.Fprint(out, usage) }
	if err = global.Parse(args); errors.Is(err, flag.ErrHelp) {
		return nil
	} else if err != nil {
		return err
	}
	rest := global.Args()
	if *showVersion {
		rest = []string{"version"}
	}
	if len(rest) == 0 {
		fmt.Fprint(out, usage)
		return nil
	}
	command := rest[0]
	rest = rest[1:]
	abs, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	encode := func(x any) error {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		return enc.Encode(x)
	}
	if command == "version" {
		return encode(map[string]string{"name": "AI Vault", "version": version, "os": runtime.GOOS, "arch": runtime.GOARCH})
	}
	if command == "help" {
		fmt.Fprint(out, usage)
		return nil
	}
	if command == "init" {
		if len(rest) != 0 {
			return errors.New("init accepts no arguments; pass --vault before init")
		}
		if err = vault.Init(abs); err != nil {
			return err
		}
		return encode(map[string]any{"vault": abs, "initialized": true})
	}
	if command == "config" {
		f := flag.NewFlagSet(command, flag.ContinueOnError)
		f.SetOutput(errOut)
		format := f.String("format", "json", "json or codex")
		if err = f.Parse(rest); err != nil {
			return err
		}
		if f.NArg() != 0 {
			return errors.New("unexpected config arguments")
		}
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		exe, err = filepath.Abs(exe)
		if err != nil {
			return err
		}
		switch *format {
		case "json":
			return encode(map[string]any{"mcpServers": map[string]any{"ai-vault": map[string]any{"command": exe, "args": []string{"--vault", abs, "serve"}}}})
		case "codex":
			_, err = fmt.Fprintf(out, "[mcp_servers.ai-vault]\ncommand = %q\nargs = [%q, %q, %q]\n", exe, "--vault", abs, "serve")
			return err
		default:
			return errors.New("format must be json or codex")
		}
	}
	known := map[string]bool{"list": true, "read": true, "write": true, "append": true, "search": true, "links": true, "backlinks": true, "tags": true, "categories": true, "trash": true, "history": true, "restore": true, "import": true, "doctor": true, "serve": true}
	if !known[command] {
		return fmt.Errorf("unknown command %q; run aivault help", command)
	}
	v, err := vault.Open(abs)
	if err != nil {
		return err
	}
	defer v.Close()
	parse := func(count int, setup func(*flag.FlagSet)) (*flag.FlagSet, error) {
		if len(rest) < count {
			return nil, fmt.Errorf("%s needs %d argument(s)", command, count)
		}
		f := flag.NewFlagSet(command, flag.ContinueOnError)
		f.SetOutput(errOut)
		setup(f)
		if err := f.Parse(rest[count:]); err != nil {
			return nil, err
		}
		if f.NArg() != 0 {
			return nil, errors.New("unexpected arguments")
		}
		return f, nil
	}
	noFlags := func(*flag.FlagSet) {}
	switch command {
	case "serve":
		var ro bool
		if _, err = parse(0, func(f *flag.FlagSet) { f.BoolVar(&ro, "read-only", false, "disable mutations") }); err != nil {
			return err
		}
		return mcpserver.Run(context.Background(), v, version, ro)
	case "list":
		if _, err = parse(0, noFlags); err != nil {
			return err
		}
		notes, err := v.List()
		if err != nil {
			return err
		}
		return encode(notes)
	case "read":
		if _, err = parse(1, noFlags); err != nil {
			return err
		}
		note, err := v.Read(rest[0])
		if err != nil {
			return err
		}
		return encode(note)
	case "write", "append":
		var content, file, expected, category string
		f, err := parse(1, func(f *flag.FlagSet) {
			f.StringVar(&content, "content", "", "UTF-8 note content")
			f.StringVar(&file, "file", "", "read content from file")
			f.StringVar(&expected, "expected-sha", "", "last read SHA-256")
			if command == "write" {
				f.StringVar(&category, "category", "", "category/subcategory folder")
			}
		})
		if err != nil {
			return err
		}
		notePath := rest[0]
		if category != "" {
			if err = vault.ValidatePath(notePath); err != nil {
				return err
			}
			if err = vault.ValidateCategory(category); err != nil {
				return err
			}
			notePath = path.Join(category, notePath)
		}
		provided := false
		f.Visit(func(f *flag.Flag) {
			if f.Name == "content" {
				provided = true
			}
		})
		if provided && file != "" {
			return errors.New("choose --content or --file")
		}
		if !provided {
			reader := in
			if file != "" {
				input, err := os.Open(file)
				if err != nil {
					return err
				}
				defer input.Close()
				reader = input
			}
			b, err := io.ReadAll(io.LimitReader(reader, vault.MaxNoteBytes+1))
			if err != nil {
				return err
			}
			if len(b) > vault.MaxNoteBytes {
				return errors.New("content exceeds 4 MiB")
			}
			content = string(b)
		}
		var note vault.Note
		if command == "write" {
			note, err = v.Write(notePath, content, expected)
		} else {
			note, err = v.Append(rest[0], content, expected)
		}
		if err != nil {
			return err
		}
		return encode(note)
	case "search":
		var limit int
		if _, err = parse(1, func(f *flag.FlagSet) { f.IntVar(&limit, "limit", 20, "maximum results (100)") }); err != nil {
			return err
		}
		hits, err := v.Search(rest[0], limit)
		if err != nil {
			return err
		}
		return encode(hits)
	case "links":
		if _, err = parse(1, noFlags); err != nil {
			return err
		}
		links, err := v.Links(rest[0])
		if err != nil {
			return err
		}
		return encode(links)
	case "backlinks":
		if _, err = parse(1, noFlags); err != nil {
			return err
		}
		notes, err := v.Backlinks(rest[0])
		if err != nil {
			return err
		}
		return encode(notes)
	case "tags":
		if _, err = parse(0, noFlags); err != nil {
			return err
		}
		notes, err := v.List()
		if err != nil {
			return err
		}
		counts := map[string]int{}
		for _, n := range notes {
			for _, tag := range n.Tags {
				counts[tag]++
			}
		}
		names := []string{}
		for t := range counts {
			names = append(names, t)
		}
		sort.Strings(names)
		tags := []map[string]any{}
		for _, t := range names {
			tags = append(tags, map[string]any{"tag": t, "count": counts[t]})
		}
		return encode(tags)
	case "categories":
		if _, err = parse(0, noFlags); err != nil {
			return err
		}
		categories, err := v.Categories()
		if err != nil {
			return err
		}
		return encode(categories)
	case "trash":
		var expected string
		if _, err = parse(1, func(f *flag.FlagSet) { f.StringVar(&expected, "expected-sha", "", "last read SHA-256") }); err != nil {
			return err
		}
		revision, err := v.Trash(rest[0], expected)
		if err != nil {
			return err
		}
		return encode(revision)
	case "history":
		if _, err = parse(1, noFlags); err != nil {
			return err
		}
		revisions, err := v.History(rest[0])
		if err != nil {
			return err
		}
		return encode(revisions)
	case "restore":
		var expected string
		if _, err = parse(2, func(f *flag.FlagSet) { f.StringVar(&expected, "expected-sha", "", "last read destination SHA-256") }); err != nil {
			return err
		}
		note, err := v.Restore(rest[0], rest[1], expected)
		if err != nil {
			return err
		}
		return encode(note)
	case "import":
		var prefix string
		if _, err = parse(1, func(f *flag.FlagSet) { f.StringVar(&prefix, "prefix", "", "relative destination directory") }); err != nil {
			return err
		}
		report, err := importer.Import(context.Background(), v, rest[0], prefix)
		if err != nil {
			if len(report.Imported) > 0 {
				_ = encode(report)
			}
			return err
		}
		return encode(report)
	case "doctor":
		if _, err = parse(0, noFlags); err != nil {
			return err
		}
		notes, err := v.List()
		if err != nil {
			return err
		}
		_, lockErr := os.Lstat(filepath.Join(abs, ".aivault", "write.lock"))
		locked := lockErr == nil
		ignored, err := v.IgnoredPaths()
		if err != nil {
			return err
		}
		var total int64
		for _, n := range notes {
			total += n.Bytes
		}
		if err = encode(map[string]any{"vault": abs, "version": version, "notes": len(notes), "note_bytes": total, "ignored_paths": ignored, "write_lock_present": locked, "transport": "stdio", "runtime_dependencies": []string{}}); err != nil {
			return err
		}
		if locked {
			return errors.New("write lock exists; wait for active writer, or stop all clients and inspect/remove the stale empty .aivault/write.lock directory after a crash")
		}
		return nil
	}
	return errors.New(strings.Join([]string{"unhandled command", command}, " "))
}
