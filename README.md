# opener

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
└────────────────────┘                 └────────────────────┘
       localhost                            remote server
```

## Setup

### Local environment

You can install opener with Homebrew. Since opener is a daemon, it is managed by Homebrew-services.

```
$ brew install superbrothers/opener
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

Fake commands use `socat` command, so install it.

```sh
# Ubuntu 20.04
$ sudo apt install socat
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

## License

MIT License
