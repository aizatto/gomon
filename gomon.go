package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/cortesi/moddwatch"
	"github.com/fatih/color"
	"github.com/go-git/go-billy/osfs"
	"github.com/go-git/go-git/plumbing/format/gitignore"
	"github.com/pkg/errors"
)

var CLI struct {
	Debounce int      `name:"debounce" help:"Delay in milliseconds before last change" default:"500" short:"d"`
	Patterns []string `arg name:"patterns" help:"Patterns to watch." default:"**"`
	Command  []string `arg name:"command" help:"Command to run" default:" "`
}

func main() {
	kong.Parse(&CLI)

	args := CLI
	patterns := args.Patterns
	if len(patterns) == 0 {
		patterns = []string{"**"}
	}

	color.Green("Watching: %s", strings.Join(patterns, " "))
	err := Run(patterns, strings.Join(args.Command, " "), args.Debounce)
	if err != nil {
		panic(err)
	}
}

func Run(patterns []string, command string, debounce int) error {
	chs := make(chan *moddwatch.Mod, 1024)

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	watcher, err := moddwatch.Watch(
		wd,
		patterns,
		[]string{
			".git/*",
		},
		time.Duration(debounce)*time.Millisecond,
		chs,
	)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	log.Println(">> start")
	cmd := TriggerCommand(command)

	killsignal := make(chan os.Signal, 1)
	signal.Notify(killsignal, os.Interrupt)
	go func() {
		<-killsignal
		KillProcessGroup(cmd)
		os.Exit(1)
	}()

	matcher, err := LoadGitignore()

	for ch := range chs {
		changedpaths := append(ch.Added, ch.Changed...)
		allpaths := append(changedpaths, ch.Deleted...)

		ruleschanged := false
		for _, path := range allpaths {
			if path == ".gitignore" || strings.HasSuffix(path, "/.gitignore") {
				ruleschanged = true
				break
			}
		}

		if ruleschanged {
			matcher, err = LoadGitignore()
		}

		PrintPaths(color.Green, "Changed/Added", changedpaths)
		PrintPaths(color.Red, "Deleted", ch.Deleted)

		if matcher != nil {
			matcher := *matcher
			run := false
			for _, path := range allpaths {
				if matcher.Match(filepath.SplitList(path), false) == false {
					run = true
					break
				}
			}

			if !run {
				log.Println(">> changed files are ignored by git")
				continue
			}
		}

		if cmd != nil {
			KillProcessGroup(cmd)
		}

		if ch == nil {
			break
		}

		log.Println(">> start")

		cmd = TriggerCommand(command)
	}

	return nil
}

func LoadGitignore() (*gitignore.Matcher, error) {
	dirs := GetGitignoreDirs()

	var patterns []gitignore.Pattern
	for _, dir := range dirs {
		fs := osfs.New(dir)

		patterns2, err := gitignore.ReadPatterns(fs, []string{})
		if err != nil {
			log.Println(errors.Wrapf(err, "Problem with dir: %s", dir))
			continue
		}
		patterns = append(patterns, patterns2...)
	}

	matcher := gitignore.NewMatcher(patterns)
	return &matcher, nil
}

func GetGitignoreDirs() []string {
	dirs := []string{}

	dir, err := os.Getwd()
	if err == nil {
		dirs = append(dirs, dir)
	} else {
		log.Println(errors.Wrap(err, "Unable to get current working directory"))
	}

	return dirs
}

// https://varunksaini.com/posts/kiling-processes-in-go/
func KillProcessGroup(cmd *exec.Cmd) {
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
			log.Println(err)
		}
	} else {
		log.Println(err)
	}
}

func PrintPaths(fn func(string, ...interface{}), title string, paths []string) {
	if len(paths) == 0 {
		return
	}

	if len(paths) > 10 {
		PrintPaths(fn, title, paths[0:10])
		fmt.Printf("Showing %d of %d\n", 10, len(paths))
		return
	}

	fn(title)
	for _, path := range paths {
		fn("- " + path)
	}
}

func TriggerCommand(command string) *exec.Cmd {
	log.Printf("Running: %s", command)
	start := time.Now()
	cmd := exec.Command("sh", "-c", command)
	// We need this to kill child processes
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		log.Println(err)
		return cmd
	}

	go func() {
		cmd.Wait()
		log.Printf(">> done: %s\n\n", time.Since(start))

		switch cmd.ProcessState.ExitCode() {
		case 0:
			color.Green(cmd.ProcessState.String())
		default:
			color.Red(cmd.ProcessState.String())
		}
	}()

	return cmd
}
