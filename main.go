package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

type CLI struct {
	BinaryPath string
	Rest       []string

	Replacement string
}

func parseArgs() (CLI, error) {
	if len(os.Args) < 2 {
		return CLI{}, fmt.Errorf("must provide at least 1 argument")
	}

	cli := CLI{
		BinaryPath: os.Args[1],
		Rest:       os.Args[2:],
	}
	cli.Replacement = os.Getenv("BINSWAP_REPLACEMENT")
	if cli.Replacement == "" {
		cli.Replacement = "/tmp/binswap"
	}
	return cli, nil
}

func (cli CLI) lastMod() (time.Time, error) {
	info, err := os.Lstat(cli.Replacement)
	if err != nil && !os.IsNotExist(err) {
		return time.Unix(0, 0), fmt.Errorf("failed to stat replacement location: %v", cli.Replacement)
	} else if os.IsNotExist(err) {
		return time.Unix(0, 0), nil
	}

	return info.ModTime(), nil
}

func (cli CLI) watch() chan struct{} {
	swaps := make(chan struct{})
	go func() {
		lastMod, err := cli.lastMod()
		if err != nil {
			log.Printf("failed to get last mod time of replacement binary: %v", err)
			return
		}

		ticker := time.Tick(200 * time.Millisecond)
		for range ticker {
			t, err := cli.lastMod()
			if err != nil {
				log.Printf("failed to get last mod time of replacement binary: %v", err)
				return
			}

			if t.After(lastMod) {
				swaps <- struct{}{}
				lastMod = t
			}
		}
	}()
	return swaps
}

func (cli CLI) cmd() *exec.Cmd {
	cmd := exec.Command(cli.BinaryPath, cli.Rest...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func (cli CLI) reap(swaps chan struct{}, cmd *exec.Cmd) {
	<-swaps
	log.Println("Noticed binary swap, killing")
	if err := cmd.Process.Kill(); err != nil {
		log.Printf("failed to kill main process: %v", err)
		return
	}
}

func (cli CLI) swap() error {
	if _, err := os.Lstat(cli.Replacement); err != nil {
		return err
	}
	return os.Rename(cli.Replacement, cli.BinaryPath)
}

func main() {
	cli, err := parseArgs()
	if err != nil {
		log.Fatalf("failed to parse arguments: %v", err)
	}

	swaps := cli.watch()
	for {
		cmd := cli.cmd()
		if err := cmd.Start(); err != nil {
			log.Fatalf("failed to start command: %v", err)
		}

		go cli.reap(swaps, cmd)
		if err := cmd.Wait(); err != nil {
			log.Printf("command failed: %v\n", err)
		}

		log.Println("Binary killed. Performing swap")
		if err := cli.swap(); err != nil {
			log.Fatalf("failed to start command: %v", err)
		}
	}
}