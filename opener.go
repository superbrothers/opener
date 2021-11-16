package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var version string
var commit string
var date string

type OpenerOptions struct {
	Network string `yaml:"network"`
	Address string `yaml:"address"`

	ErrOut io.Writer
}

func NewOpenerCmd(errOut io.Writer) *cobra.Command {
	var configPath string

	o := &OpenerOptions{
		Network: "unix",
		Address: "~/.opener.sock",
		ErrOut:  errOut,
	}

	cmd := &cobra.Command{
		Use: "opener",
		RunE: func(_ *cobra.Command, args []string) error {
			if err := LoadOpenerOptionsFromConfig(configPath, o); err != nil {
				return err
			}

			if err := o.Validate(); err != nil {
				return err
			}

			return o.Run()
		},
	}

	cmd.Flags().StringVar(&configPath, "config", configPath, "Path to the opener config file (defaults to ~/.config/opener/config.yaml)")

	return cmd
}

func (o *OpenerOptions) Validate() error {
	switch o.Network {
	case "unix":
		address, err := homedir.Expand(o.Address)
		if err != nil {
			return err
		}
		o.Address = address

		syscall.Umask(0077)

		if err := os.RemoveAll(o.Address); err != nil {
			return err
		}
	case "tcp":
	default:
		return errors.New("allowed network are: unix,tcp")
	}

	return nil
}

func (o *OpenerOptions) Run() error {
	fmt.Fprintf(o.ErrOut, "version: %s, commit: %s, date: %s\n", version, commit, date)
	fmt.Fprintf(o.ErrOut, "starting a server at %s\n", o.Address)

	ln, err := net.Listen(o.Network, o.Address)
	if err != nil {
		return err
	}

	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Fprintln(o.ErrOut, err)
				return
			}

			go handleConnection(conn, o.ErrOut)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	sig := <-c
	fmt.Fprintf(o.ErrOut, "got signal %s\n", sig)

	return nil
}

var browserMu sync.Mutex

var openURL = func(line string) (string, error) {
	// We try out best avoiding race-condition on swapping browser.{Stdout,Stderr}.
	// This works in a case when there are two or more consumers exist for this package.
	//
	// Fingers-crossed when github.com/pkg/browser is used concurrently outside of this package...
	browserMu.Lock()

	stdout, stderr := browser.Stdout, browser.Stderr

	defer func() {
		browser.Stdout = stdout
		browser.Stderr = stderr

		browserMu.Unlock()
	}()

	var buf bytes.Buffer

	browser.Stdout = &buf
	browser.Stderr = &buf

	err := browser.OpenURL(line)

	return buf.String(), err
}

func handleConnection(conn net.Conn, errOut io.Writer) {
	defer conn.Close()

	line, err := bufio.NewReader(conn).ReadString('\n')
	line = strings.TrimRight(line, "\n")
	fmt.Fprintf(errOut, "received %q\n", line)
	if err != nil {
		if err != io.EOF {
			fmt.Fprintln(errOut, err)
			return
		}
	}

	logs, err := openURL(line)

	if logs != "" {
		fmt.Fprint(errOut, logs)
	}

	if err != nil {
		fmt.Fprintf(errOut, "failed to open %q: %v\n", line, err)

		// Send back the logs from `open` to the client over e.g. the unix domain socket, so that
		// `open` on the client machine would work more like that on the server.
		//
		// Note that this works only when the client selected the protocol of SOCK_STREAM rather than e.g. SOCK_DGRAM.
		// `socat`, for example, negotiates the protocol to prefer SOCK_STREAM so you won't usually care.
		if _, err := conn.Write([]byte(logs)); err != nil {
			fmt.Fprintf(errOut, "failed to send error to client: %v\n", err)
		}
		return
	}
}
