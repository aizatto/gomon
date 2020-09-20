# Readme

My goal was to build a filesystem monitor that only executed the command after a file system change.

Some problem with alternative watchers:

- They rerun the command when the exit code is non zero
- They are a task runner
- Do not kill child processes

I only want to re run the command on file system changes.

# Example

```sh
gomon --help
```

```sh
gomon <paths> -- <command>
```

```
gomon "**" -- go build .
```

# Alternatives

- https://github.com/cortesi/modd
- https://github.com/nathany/looper
- https://github.com/cespare/reflex
- https://github.com/mitranim/gow
- https://github.com/loov/watchrun
- https://github.com/c9s/gomon
- https://github.com/canthefason/go-watcher
- https://github.com/go-godo/godo 
- https://github.com/gravityblast/fresh
- https://github.com/githubnemo/CompileDaemon
- https://github.com/codegangsta/gin
- https://github.com/cespare/reflex

# Notes

https://github.com/cortesi/moddwatch/blob/master/watch.go
https://github.com/cortesi/modd/search?q=notify&unscoped_q=notify
https://github.com/cortesi/modd/blob/06afa96cb8f7fa492c2eb649d1b569f77fa986f5/modd.go#L194
https://gobyexample.com/channel-synchronization
https://golang.org/pkg/os/exec/#Command
https://gobyexample.com/command-line-flags

## Killing Child Processes

https://varunksaini.com/posts/kiling-processes-in-go/