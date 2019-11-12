package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

type CLI struct {
	OriginalBinaryPath string
	BinaryPath         string
	Rest               []string

	Replacement string
	Count       int
}

func parseArgs() (*CLI, error) {
	if len(os.Args) < 2 {
		return nil, fmt.Errorf("must provide at least 1 argument")
	}

	cli := &CLI{
		OriginalBinaryPath: os.Args[1],
		BinaryPath:         os.Args[1],
		Rest:               os.Args[2:],
	}
	cli.Replacement = os.Getenv("BINSWAP_REPLACEMENT")
	if cli.Replacement == "" {
		cli.Replacement = "/tmp/binswap"
	}
	return cli, nil
}

func (cli *CLI) lastMod() (time.Time, error) {
	info, err := os.Lstat(cli.Replacement)
	if err != nil && !os.IsNotExist(err) {
		return time.Unix(0, 0), fmt.Errorf("failed to stat replacement location: %v", cli.Replacement)
	} else if os.IsNotExist(err) {
		return time.Unix(0, 0), nil
	}

	return info.ModTime(), nil
}

func (cli *CLI) watch() chan struct{} {
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

func (cli *CLI) cmd() *exec.Cmd {
	cmd := exec.Command(cli.BinaryPath, cli.Rest...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func (cli *CLI) reap(swaps chan struct{}, cmd *exec.Cmd) {
	<-swaps
	log.Println("Noticed binary swap, killing")
	if err := cmd.Process.Kill(); err != nil {
		log.Printf("failed to kill main process: %v", err)
		return
	}
}

func (cli *CLI) swap() error {
	if _, err := os.Lstat(cli.Replacement); err != nil {
		return fmt.Errorf("failed to lstat replacement: %v", err)
	}

	if cli.BinaryPath != cli.OriginalBinaryPath {
		for {
			if err := os.Remove(cli.BinaryPath); err != nil {
				log.Printf("failed to remove old binary: %v", err)
				continue
			}
			break
		}
	}

	cli.BinaryPath = fmt.Sprintf("%s-%d", cli.OriginalBinaryPath, cli.Count)
	cli.Count++
	for {
		err := os.Rename(cli.Replacement, cli.BinaryPath)
		if err == nil {
			return nil
		}
		log.Printf("failed to swap binaries: %v", err)
		time.Sleep(500 * time.Millisecond)
	}
}

func main() {
	cli, err := parseArgs()
	if err != nil {
		log.Fatalf("failed to parse arguments: %v", err)
	}

	swaps := cli.watch()
	for {
		log.Printf("starting binary: %v\n", cli.BinaryPath)
		cmd := cli.cmd()
		if err := cmd.Start(); err != nil {
			log.Printf("failed to start command: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		go cli.reap(swaps, cmd)
		if err := cmd.Wait(); err != nil {
			log.Printf("command failed: %v\n", err)
		}

		log.Println("Binary killed. Performing swap")
		if err := cli.swap(); err != nil {
			log.Printf("failed to perform swap: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
	}
}
