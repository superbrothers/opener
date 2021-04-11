package main

import (
	"errors"
	"io"
	"net"
	"testing"
)

func TestHandleConnection(t *testing.T) {
	tt := []struct {
		test        string
		openURLFunc func(string) (string, error)
		data        string
		err         error
	}{
		{
			"Say nothing when successful",
			func(line string) (string, error) {
				return "pong\n", nil
			},
			"",
			io.EOF,
		},
		{
			"Sending back the logs when failure",
			func(line string) (string, error) {
				return "pong\n", errors.New("exit status 1")
			},
			"pong\n",
			nil,
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

			buf := make([]byte, 1024)
			n, err := client.Read(buf)
			data := string(buf[:n])
			if tc.data != data {
				t.Errorf("expect %q, but actual %q", tc.data, data)
			}

			if tc.err != err {
				t.Errorf("expect %v, but actual %v", tc.err, err)
			}
		})
	}
}
