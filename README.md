# Candy ![gopher_with_candy](https://raw.githubusercontent.com/egonelbre/gophers/master/.thumb/sketch/misc/with-candy.png)

Candy is a zero-config reverse proxy server.
It makes proxying applications with local top-leveled domains as frictionless as possible.
There is no messing around with `/etc/hosts`, [Dnsmasq](https://en.wikipedia.org/wiki/Dnsmasq), or [Nginx](https://nginx.org/).

## How does it work?

A few simple conventions eliminate the need for tedious configuration.
Candy runs as your user on unprivileged ports, and includes an HTTP, an HTTPS, and a DNS server.
It also sets up a system hook so that all DNS queries for a local top-level domain (`.test`) resolve to your local machine.

To route web traffic to an app, just create a file in the `~/.candy` directory.
Assuming you are developing an app that runs on `http://127.0.0.1:8080`, and you would like to access it at `http://myapp.test`, setting it up is as easy as:

```
$ echo "8080" > ~/.candy/myapp
```

The name of the file (`myapp`) determines the hostname (`myapp.test`) to access the application that it points to (`127.0.0.1:8080`).
Both HTTP and HTTPS request is supported out of the box:

```
$ curl http://myapp.test
$ curl https://myapp.test
```

## Installation

### Mac

```
brew install owenthereal/candy/candy
```

After installing the `candy` binary, you also need to create a [DNS resolver config](https://www.unix.com/man-page/opendarwin/5/resolver/) in `/etc/resolver/CONFIG`.
Creating the `/etc/resolver` directory and the config file require superuser privileges. You can execute with a one-liner:

```
sudo candy setup
```

Alternatively, you can manually execute the followings:

```
sudo mkdir -p /etc/resolver
cat<<EOF | sudo tee /etc/resolver/candy-test > /dev/null
domain test
nameserver 127.0.0.1
port 25353
search_order 1
timeout 5
EOF
```

### Linux

```
go install github.com/owenthereal/candy/cmd/candy@latest
```

After installing the `candy` binary, you also need to create a [network name resolution config](https://www.freedesktop.org/software/systemd/man/resolved.conf.html) in `/usr/lib/systemd/resolved.conf.d/CONFIG`.
Creating the `/usr/lib/systemd/resolved.conf.d` directory and the config file require superuser privileges. You can execute with a one-liner:

```
sudo candy setup
```

Alternatively, you can manually execute the followings:

```
sudo mkdir -p /usr/lib/systemd/resolved.conf.d
cat<<EOF | sudo tee /usr/lib/systemd/resolved.conf.d/01-candy.conf > /dev/null
[Resolve]
DNS=127.0.0.1:25353
Domains=test
EOF
sudo systemctl restart systemd-resolved # Restart systemd-resolved
```

## Usage

### Starting Candy

#### Mac

To have [Launchd](https://en.wikipedia.org/wiki/Launchd) start Candy and restart at login:

```
brew services start candy
```

To restart Candy, run:

```
brew services restart candy
```

To stop Candy, run:

```
brew services stop candy
```

Or, if you don't want/need a background service, you can just run:

```
candy run
```

#### Linux

TODO

### Port/IP proxying

Candy's port/IP proxying feature lets you route all web traffic on a particular hostname to another port or IP address.
To use it, create a file in `~/.candy` with the the destination port number or IP address as its contents:

```
echo "8080" > ~/.candy/app1
curl https://app1.test

echo "1.2.3.4:8080" > ~/.candy/app2
curl https://app2.test
```

### Configuration

Candy provides good defaults that most people will never need to configure it.
However, if you need to adjust a setting or two, you can create a file to override the defaults in `~/.candyconfig`.
See [this file](https://github.com/owenthereal/candy/blob/e5a250f950f9db2d0431805e0a9e3719164352c1/cmd/candy/command/run.go#L28-L36) for a list of settings that you can change.

For example, you may want to have multiple top-leveled domains besides `*.test`:

```json
{
  "domain": ["test","mydomain"]
}
```

Changing the `domain` setting requires resetting DNS resolvers in `/etc/resolver`.
Rerun the [setup step](#setup) and make sure all resolver config files are matching in `/etc/resolver`.

After changing a setting in `~/.candyconfig`, you will also need to [restart](#starting-candy) Candy for the change to take effect:

## Prior Arts

* [pow](https://github.com/basecamp/pow)
* [puma-dev](https://github.com/puma/puma-dev)
