// TiCS: disabled // This is generating completion and doc, not production code.

//go:build tools

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/ubuntu/ubuntu-insights/insights/cmd/insights/commands"
)

const usage = `Usage of %s:

   completion DIRECTORY
     Create completions files in a structured hierarchy in DIRECTORY.
   man DIRECTORY
     Create man pages files in a structured hierarchy in DIRECTORY.
`

func main() {
	if len(os.Args) < 2 {
		log.Fatalf(usage, os.Args[0])
	}
	i, err := commands.New()
	if err != nil {
		log.Fatalf("Couldn't create new command: %v", err)
	}

	commands := []cobra.Command{i.RootCmd()}

	if len(os.Args) < 3 {
		log.Fatalf(usage, os.Args[0])
	}
	dir := filepath.Join(os.Args[2], "usr", "share")
	switch os.Args[1] {
	case "completion":
		genCompletions(commands, dir)
	case "man":
		genManPages(commands, dir)
	default:
		log.Fatalf(usage, os.Args[0])
	}
}

// genCompletions for bash and zsh directories
func genCompletions(cmds []cobra.Command, dir string) {
	bashCompDir := filepath.Join(dir, "bash-completion", "completions")
	zshCompDir := filepath.Join(dir, "zsh", "site-functions")
	for _, d := range []string{bashCompDir, zshCompDir} {
		if err := cleanDirectory(filepath.Dir(d)); err != nil {
			log.Fatalln(err)
		}
		if err := createDirectory(d, 0755); err != nil {
			log.Fatalf("Couldn't create bash completion directory: %v", err)
		}
	}

	for _, cmd := range cmds {
		if err := cmd.GenBashCompletionFileV2(filepath.Join(bashCompDir, cmd.Name()), true); err != nil {
			log.Fatalf("Couldn't create bash completion for %s: %v", cmd.Name(), err)
		}
		if err := cmd.GenZshCompletionFile(filepath.Join(zshCompDir, cmd.Name())); err != nil {
			log.Fatalf("Couldn't create zsh completion for %s: %v", cmd.Name(), err)
		}
	}
}

func genManPages(cmds []cobra.Command, dir string) {
	manBaseDir := filepath.Join(dir, "man")
	if err := cleanDirectory(manBaseDir); err != nil {
		log.Fatalln(err)
	}

	out := filepath.Join(manBaseDir, "man1")
	if err := createDirectory(out, 0755); err != nil {
		log.Fatalf("Couldn't create man pages directory: %v", err)
	}

	for _, cmd := range cmds {
		cmd := cmd
		// Run ExecuteC to install completion and help commands
		_, _ = cmd.ExecuteC()
		opts := doc.GenManTreeOptions{
			Header: &doc.GenManHeader{
				Title: fmt.Sprintf("Ubuntu-insights: %s", cmd.Name()),
			},
			Path: out,
		}
		if err := genManTreeFromOpts(&cmd, opts); err != nil {
			log.Fatalf("Couldn't generate man pages for %s: %v", cmd.Name(), err)
		}
	}
}

func mustWriteLine(w io.Writer, msg string) {
	if _, err := w.Write([]byte(msg + "\n")); err != nil {
		log.Fatalf("Couldn't write %s: %v", msg, err)
	}
}

// genManTreeFromOpts generates a man page for the command and all descendants.
// The pages are written to the opts.Path directory.
// This is a copy from cobra, but it will include Hidden commands.
func genManTreeFromOpts(cmd *cobra.Command, opts doc.GenManTreeOptions) error {
	header := opts.Header
	if header == nil {
		header = &doc.GenManHeader{}
	}
	for _, c := range cmd.Commands() {
		if (!c.IsAvailableCommand() && !c.Hidden) || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := genManTreeFromOpts(c, opts); err != nil {
			return err
		}
	}
	section := "1"
	if header.Section != "" {
		section = header.Section
	}

	separator := "_"
	if opts.CommandSeparator != "" {
		separator = opts.CommandSeparator
	}
	basename := strings.Replace(cmd.CommandPath(), " ", separator, -1)
	filename := filepath.Join(opts.Path, basename+"."+section)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	headerCopy := *header
	return doc.GenMan(cmd, &headerCopy, f)
}

// cleanDirectory removes a directory and recreates it.
func cleanDirectory(p string) error {
	if err := os.RemoveAll(p); err != nil {
		return fmt.Errorf("couldn't delete %q: %w", p, err)
	}
	if err := createDirectory(p, 0750); err != nil {
		return fmt.Errorf("couldn't create %q: %w", p, err)
	}
	return nil
}

// createDirectory creates a directory with the given permissions.
// If the directory already exists, it is left untouched.
// If the directory cannot be created, an error is returned.
//
// Prefer this way of creating directories instead of os.Mkdir as the latter
// could bypass fakeroot and cause unexpected confusion.
//
// The additional os.MkdirAll is for compatibility with Windows.
func createDirectory(dir string, perm uint32) error {
	// First attempt with os.MkdirAll
	if err := os.MkdirAll(dir, os.FileMode(perm)); err != nil {
		// #nosec:G204 - we control the mode and directory we run mkdir on
		cmd := exec.Command("mkdir", "-m", fmt.Sprintf("%o", perm), "-p", dir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("Couldn't create dest directory: %v", string(output))
		}
	}

	return nil
}
