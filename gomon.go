package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/cortesi/moddwatch"
	"github.com/fatih/color"
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
		[]string{},
		time.Duration(debounce)*time.Millisecond,
		chs,
	)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	log.Println(">> start")
	TimeTrigger(command)

	for ch := range chs {
		if ch == nil {
			break
		}

		paths := append(ch.Added, ch.Changed...)
		log.Println(">> start")
		PrintPaths(color.Green, "Changed/Added", paths)
		PrintPaths(color.Red, "Deleted", ch.Deleted)

		TimeTrigger(command)
	}

	return nil
}

func PrintPaths(fn func(string, ...interface{}), title string, paths []string) {
	if len(paths) == 0 {
		return
	}

	if len(paths) > 10 {
		PrintPaths(fn, title, paths[0:10])
		fmt.Printf("Showing %d of %d", 10, len(paths))
		return
	}

	fn(title)
	for _, path := range paths {
		fn("- " + path)
	}
}

func TimeTrigger(command string) error {
	start := time.Now()
	fmt.Println("")
	err := Trigger(command)
	log.Printf(">> done: %s\n\n", time.Since(start))
	if err != nil {
		panic(err)
	}
	return nil
}

func Trigger(command string) error {
	log.Printf("Running: %s", command)
	cmd := exec.Command("sh", "-c", command)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}

	eret := cmd.Wait()

	switch cmd.ProcessState.ExitCode() {
	case 0:
		color.Green(cmd.ProcessState.String())
	default:
		color.Red(cmd.ProcessState.String())
	}

	if err != nil {
		fmt.Print(eret)
	}

	return nil
}
