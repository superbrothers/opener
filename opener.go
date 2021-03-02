package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var version string
var commit string
var date string

type OpenerOptions struct {
	Address string

	ErrOut io.Writer
}

func NewOpenerCmd(errOut io.Writer) *cobra.Command {
	o := &OpenerOptions{
		Address: "~/.opener.sock",
		ErrOut:  errOut,
	}

	cmd := &cobra.Command{
		Use: "opener",
		RunE: func(_ *cobra.Command, args []string) error {
			if err := o.Validate(); err != nil {
				return err
			}

			return o.Run()
		},
	}

	return cmd
}

func (o *OpenerOptions) Validate() error {
	address, err := homedir.Expand(o.Address)
	if err != nil {
		return err
	}
	o.Address = address

	return nil
}

func (o *OpenerOptions) Run() error {
	fmt.Fprintf(o.ErrOut, "version: %s, commit: %s, date: %s\n", version, commit, date)

	syscall.Umask(0077)

	if err := os.RemoveAll(o.Address); err != nil {
		return err
	}

	fmt.Fprintf(o.ErrOut, "starting UNIX domain socket server at %s\n", o.Address)

	ln, err := net.Listen("unix", o.Address)
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

	if err := browser.OpenURL(line); err != nil {
		fmt.Fprintf(errOut, "failed to open %q: %v\n", line, err)
		return
	}
}
