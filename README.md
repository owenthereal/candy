# Candy

Candy is a zero-config reverse proxy.

# Manual setup

## Mac OS

Create a file in `/etc/resolver/candy`

```
domain test
nameserver 127.0.0.1
port 8080
search_order 1
timeout 5
```
Replace `port` with the `candy` DNS server port.

Each time a file is created or a change is made to a file in `/etc/resolver` you may need to run the following to reload Mac OS mDNS resolver.

```
sudo launchctl unload -w /System/Library/LaunchDaemons/com.apple.mDNSResponder.plist
sudo launchctl load -w /System/Library/LaunchDaemons/com.apple.mDNSResponder.plist
```

Ref: https://www.unix.com/man-page/opendarwin/5/resolver/
