package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	filepath "path"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"github.com/cortesi/moddwatch"
)

var CLI struct {
	Watch struct {
		Paths   []string `arg name:"path" help:"Paths to watch." type:"path" required`
		Command []string `arg name:"command" help:"Command to run. required"`
	} `cmd help:"Watch paths."`
}

func main() {
	ctx := kong.Parse(&CLI)
	switch ctx.Command() {
	case "watch <path> <command>":
		for _, path := range CLI.Watch.Paths {
			PreRun(path, strings.Join(CLI.Watch.Command, " "))
		}
	default:
		fmt.Print(ctx.Command())
	}
}

func PreRun(path string, command string) {
	var err error

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if !strings.HasPrefix(path, "/") {
		path = filepath.Join(wd, path)
	}

	if _, err = os.Stat(path); os.IsNotExist(err) {
		log.Printf("Path does not exist: %s", path)
		panic(err)
	}

	log.Printf("Watching %s", path)
	Run(path, command)
}

func Run(path string, command string) error {
	chs := make(chan *moddwatch.Mod, 2)

	for {
		watcher, err := moddwatch.Watch(
			path,
			[]string{"*"},
			[]string{},
			time.Millisecond*200,
			chs,
		)
		if err != nil {
			return err
		}
		defer watcher.Stop()

		TimeTrigger(command)

		for ch := range chs {
			if ch == nil {
				break
			}

			TimeTrigger(command)
		}
	}
}

func TimeTrigger(command string) error {
	start := time.Now()
	err := Trigger(command)
	log.Println(">> done (%s)", time.Since(start))
	return err
}

func Trigger(command string) error {
	fmt.Println(command)
	cmd := exec.Command("sh", "-c", command)

	stdo, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stde, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	buff := new(bytes.Buffer)

	bufferr := true
	wg := sync.WaitGroup{}
	wg.Add(2)
	buflock := sync.Mutex{}
	go logOutput(
		&wg, stde,
		func(s string, args ...interface{}) {
			// log.Warn(s, args...)
			if bufferr {
				buflock.Lock()
				defer buflock.Unlock()
				fmt.Fprintf(buff, "%s\n", args...)
			}
		},
	)
	go logOutput(&wg, stdo, log.Printf)

	wg.Wait()

	eret := cmd.Wait()

	fmt.Printf("state %s\n", cmd.ProcessState.String())
	fmt.Print(buff.String())

	return eret
}

func logOutput(wg *sync.WaitGroup, fp io.ReadCloser, out func(string, ...interface{})) {
	defer wg.Done()
	r := bufio.NewReader(fp)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return
		}
		out("%s", string(line))
	}
}
