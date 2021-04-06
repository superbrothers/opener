package main

import (
	"bufio"
	"errors"
	"io"
	"net"
	"testing"
)

func TestHandleConnection(t *testing.T) {
	tt := []struct {
		test        string
		openURLFunc func(string) (string, error)
		res         string
	}{
		{
			"Sending back the logs",
			func(line string) (string, error) {
				return "pong\n", errors.New("exit status 1")
			},
			"pong\n",
		},
	}

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()

	for _, tc := range tt {
		t.Run(tc.test, func(t *testing.T) {
			openURL = tc.openURLFunc

			go func() {
				conn, _ := ln.Accept()
				go handleConnection(conn, io.Discard)
			}()

			client, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()

			if _, err := client.Write([]byte("ping\n")); err != nil {
				t.Fatal(err)
			}

			res, err := bufio.NewReader(client).ReadString('\n')
			if tc.res != res {
				t.Errorf("expect %q, but actual %q", tc.res, res)
			}
		})
	}
}
