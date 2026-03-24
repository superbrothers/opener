# opener

![Logo](./sennuki.png)

Open URL in your local web browser from the SSH-connected remote environment.

## How does opener work?

opener is a daemon process that runs locally. When you send a URL to the process, it will execute a command tailored to your local environment (`open` on macOS, `xdg-open` on Linux) with the URL as an argument. As a result, the URL will be opened in your favorite web browser.

You remotely forward the socket file of the opener daemon, `~/.opener.sock`, when you log in to the remote environment via SSH. In a remote environment, you use fake `open` command or` xdg-open` command to send the URL to `~/.opener.sock` being forwarded from your local environment. The result is as if URL was sent to the local opener daemon, which opens the URL in your local web browser.

```
┌────────────────────┐                 ┌────────────────────┐
│                    │                 │                    │
│ ┌────────────────┐ │                 │ ┌────────────────┐ │
│ │   Web Browser  │ │                 │ │  open command  │ │
│ └─▲──────────────┘ │                 │ │     (fake)     │ │
│   │ Open URL       │                 │ └─┬──────────────┘ │
│ ┌─┴──────────────┐ │                 │   │                │
│ │  opener daemon │ │                 │   │ Send URL       │
│ └─┬──────────────┘ │                 │   │                │
│   │                │                 │   │                │
│ ┌─┴──────────────┐ │ SSH connection  │ ┌─▼──────────────┐ │
│ │ ~/.opener.sock │ ├─────────────────► │ ~/.opener.sock │ │
│ └────────────────┘ │ Remote forward  │ └────────────────┘ │
│                    │                 │                    │
│      localhost     │                 │    remote server   │
└────────────────────┘                 └────────────────────┘
```

## Setup

### Local environment

You can install opener with Homebrew. Since opener is a daemon, it is managed by Homebrew-services.

```
$ brew install superbrothers/opener/opener
$ brew services start opener
```

Set ssh config to forward `~/.opener.sock` to the remote environment.

```
Host host.example.org
  RemoteForward /home/me/.opener.sock /Users/me/.opener.sock
```

### Remote environment

Install a fake `open` or` xdg-open` command. Please choose your preference either way.

```sh
$ mkdir ~/bin
# open command
$ curl -L -o ~/bin/open https://raw.githubusercontent.com/superbrothers/opener/master/bin/open
$ chmod 755 ~/bin/open
# xdg-open command
$ curl -L -o ~/bin/xdg-open https://raw.githubusercontent.com/superbrothers/opener/master/bin/xdg-open
$ chmod 755 ~/bin/xdg-open
# Add ~/bin to $PATH and enable it
$ echo 'export PATH="$HOME/bin:$PATH"' >>~/.bashrc
$ source ~/.bashrc
```

Fake commands use `nc` command, so install it if you don't have it.

```sh
# Ubuntu 20.04
$ sudo apt install netcat
```

Add the following settings to sshd. This is an option to delete the socket file when you lose the connection to the remote environment.

```sh
# Add a configuration file
$ echo "StreamLocalBindUnlink yes" | sudo tee /etc/ssh/sshd_config.d/opener.conf
# Restart ssh service
$ sudo systemctl restart ssh
```

## How to use it

If set up correctly, the following command in a remote environment will send the URL through opener and open the URL in your local web browser.

```
$ open https://www.google.com/
```

## Configuration

You can configure opener with a config file. By default, it should be located at `~/.config/opener/config.yaml`. You can also specify a config file with `--config` option.

```yaml
# The network to use opener daemon.
# Allowed networks are: unix or tcp. (defaults to unix)
network: unix

# The address to listen on. (defaults to ~/.opener.sock)
address: ~/.opener.sock

# SSH control socket for automatic port forwarding (optional, see below).
auto-forward-control-socket: ~/.ssh/cm_socket/my-remote

# How long to keep a forwarded port before cleaning it up. (defaults to 1m)
auto-forward-ttl: 60s
```

### Automatic SSH port forwarding

Some URLs opened through opener point to `localhost` on a non-standard port.
This is common with OAuth callback flows where a CLI tool like `az login` or
`gcloud auth login` starts a temporary local HTTP server (for example
`http://localhost:38947/callback`). When you are working on a remote server
over SSH, these URLs fail because the port is not forwarded.

If you set `auto-forward-control-socket` in your config, opener will detect localhost URLs
with non-standard ports and automatically set up SSH port forwards using your
existing SSH connection. This works for both direct localhost URLs and localhost
URLs embedded in query parameters (like `redirect_uri`).

Forwards are cleaned up automatically after `auto-forward-ttl` (default 1 minute).

To use this feature:

1. Enable SSH connection sharing with a static control socket path in your `~/.ssh/config`:

```
Host my-remote
  ControlMaster auto
  ControlPath ~/.ssh/cm_socket/my-remote
  ControlPersist 60m
```

2. Add the same path to your opener config (`~/.config/opener/config.yaml`):

```yaml
auto-forward-control-socket: ~/.ssh/cm_socket/my-remote
auto-forward-ttl: 60s
```

When opener receives a URL like
`https://login.example.com/?redirect_uri=http%3A%2F%2Flocalhost%3A54123%2Fcallback`,
it will run `ssh -S ~/.ssh/cm_socket/my-remote -O forward -L 54123:localhost:54123 none`
before opening the URL in your browser. After `auto-forward-ttl` elapses the port
forward is removed with `ssh -O cancel`.

### Example: Open a URL from inside a container

If you want to open a URL from inside a container, you can use `tcp` network instead of `unix`.

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│ ┌────────────────┐                                      │
│ │   Web Browser  │                                      │
│ └─▲──────────────┘                 ┌──────────────────┐ │
│   │                                │     container    │ │
│   │ Open URL                       │                  │ │
│   │                                │ ┌──────────────┐ │ │
│ ┌─┴──────────────┐    Send a URL   │ │ open command │ │ │
│ │  opener daemon │◄────────────────┼─┤    (fake)    │ │ │
│ └────────────────┘   (TCP request) │ └──────────────┘ │ │
│   127.0.0.1:9999                   │                  │ │
│                                    └──────────────────┘ │
│                       localhost                         │
└─────────────────────────────────────────────────────────┘
```

Create the following config at `~/.config/opener/config.yaml`:

```yaml
network: tcp
address: 127.0.0.1:9999
```

Restart the opener daemon:

```
$ brew services restart opener
```

Send a URL to the opener daemon from inside a container:

```
$ docker run --rm -it busybox /bin/sh
# echo https://www.google.com/ | nc host.docker.internal 9999
```

The following script is useful as a fake `open` command.

```sh
#!/bin/sh
echo "$@" | nc host.docker.internal 9999
```

## License

MIT License
