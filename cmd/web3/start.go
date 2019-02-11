package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/urfave/cli"
)

func start(c *cli.Context) error {
	// var dDir string
	// home := config.GetHomeDir()
	// if c.String("data-dir") != "" {
	// 	dDir = c.String("data-dir")
	// } else {
	// 	dDir = filepath.Join(home, ".fn")
	// }

	// check if the container already exists
	// docker ps -a --filter name=gochain --format "{{.Names}}"
	cmd := exec.Command("docker", "ps", "-a", "--filter", "name=gochain", "--format", "{{.Names}}")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	if len(stdoutStderr) != 0 {
		// then already exists, so just start it again
		fmt.Println("Restarting existing container 'gochain'...")
		cmd = exec.Command("docker", "start", "gochain")
	} else {

		args := []string{"run",
			// todo: // should use the `--rm` flag if we allow user to mount a local data dir
			// It's a much better experience than having to do docker rm or switch to docker start.
			// We could also check to see if the container exists and if it does, automatically do a `start` rather than a `run`
			// "--rm",
			// "-v", fmt.Sprintf("%s/data:/app/data", dDir),
			"-i",
			"--name", "gochain",
			"-v", "/var/run/docker.sock:/var/run/docker.sock",
			// "--privileged", // if we to run docker-in-docker
			"-p", fmt.Sprintf("%d:8545", 8545), // fmt'd so we an let use pass these in
			"-p", fmt.Sprintf("%d:8546", 8546),
			"--entrypoint", "gochain",
		}
		// if c.String("log-level") != "" {
		// 	args = append(args, "-e", fmt.Sprintf("FN_LOG_LEVEL=%v", c.String("log-level")))
		// }
		if c.String("env-file") != "" {
			args = append(args, "--env-file", c.String("env-file"))
		}
		if c.Bool("detach") {
			args = append(args, "-d")
		}
		args = append(args, "gochain/gochain", "--local")
		cmd = exec.Command("docker", args...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Fatalln("Starting command failed:", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	// catch ctrl-c and kill
	sigC := make(chan os.Signal, 2)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sigC:
			log.Println("Interrupt caught, exiting")
			err = cmd.Process.Signal(syscall.SIGTERM)
			if err != nil {
				log.Println("Error: could not kill process:", err)
				return err
			}
		case err := <-done:
			if err != nil {
				log.Println("Error: processed finished with error", err)
			}
		}
		return err
	}
}