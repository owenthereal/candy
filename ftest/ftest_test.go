package ftest

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/owenthereal/candy"
	"github.com/owenthereal/candy/caddy"
	"github.com/owenthereal/candy/dns"
	"github.com/owenthereal/candy/fswatch"
	"go.uber.org/zap"
)

func Test_Server(t *testing.T) {
	var (
		hostRoot  = t.TempDir()
		httpAddr  = ":" + randomPort(t)
		httpsAddr = ":" + randomPort(t)
		adminAddr = "127.0.0.1:" + randomPort(t)
		dnsAddr   = "127.0.0.1:" + randomPort(t)
		tlds      = []string{"go-test"}
	)

	if err := ioutil.WriteFile(filepath.Join(hostRoot, "app"), []byte(adminAddr), 0o644); err != nil {
		t.Fatal(err)
	}

	caddyCfg := caddy.Config{
		HTTPAddr:  httpAddr,
		HTTPSAddr: httpsAddr,
		AdminAddr: adminAddr,
		TLDs:      tlds,
		HostRoot:  hostRoot,
		Logger:    zap.NewNop(),
	}
	dnsCfg := dns.Config{
		Addr:   dnsAddr,
		TLDs:   tlds,
		Logger: zap.NewNop(),
	}
	svr := candy.Server{
		Proxy: caddy.New(caddyCfg),
		DNS:   dns.New(dnsCfg),
		Watcher: fswatch.New(fswatch.Config{
			HostRoot: hostRoot,
			Logger:   zap.NewNop(),
		}),
	}

	go func() {
		if err := svr.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	t.Run("http addr", func(t *testing.T) {
		waitUntil(t, 3, func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s/config/apps/http/servers/http/listen/0", adminAddr))
			if err != nil {
				return err
			}

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			gotAddr := strings.Trim(strings.TrimSpace(string(b)), "\"")

			if diff := cmp.Diff(httpAddr, gotAddr); diff != "" {
				return fmt.Errorf("Unexpected http listener addr (-want +got): %s", diff)
			}

			return nil
		})
	})

	t.Run("https addr", func(t *testing.T) {
		waitUntil(t, 3, func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s/config/apps/http/servers/https/listen/0", adminAddr))
			if err != nil {
				return err
			}

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			gotAddr := strings.Trim(strings.TrimSpace(string(b)), "\"")

			if diff := cmp.Diff(httpsAddr, gotAddr); diff != "" {
				return fmt.Errorf("Unexpected https listener addr (-want +got): %s", diff)
			}

			return nil
		})
	})

	t.Run("tls subjects", func(t *testing.T) {
		waitUntil(t, 3, func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s/config/apps/tls/automation/policies/0/subjects", adminAddr))
			if err != nil {
				return nil
			}

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil
			}

			gotSubjects := strings.TrimSpace(string(b))

			if diff := cmp.Diff(`["app.go-test"]`, gotSubjects); diff != "" {
				return fmt.Errorf("Unexpected tls subjects (-want +got): %s", diff)
			}

			return nil
		})

	})

	t.Run("resolve dns", func(t *testing.T) {
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return net.Dial("udp", dnsAddr)
			},
		}

		ips, err := r.LookupHost(context.Background(), "app.go-test")
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff([]string{"127.0.0.1"}, ips); diff != "" {
			t.Fatalf("Unexpected IPs (-want +got): %s", diff)
		}
	})

	t.Run("add new domain", func(t *testing.T) {
		if err := ioutil.WriteFile(filepath.Join(hostRoot, "app2"), []byte(adminAddr), 0o644); err != nil {
			t.Fatal(err)
		}

		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return net.Dial("udp", dnsAddr)
			},
		}

		ips, err := r.LookupHost(context.Background(), "app2.go-test")
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff([]string{"127.0.0.1"}, ips); diff != "" {
			t.Fatalf("Unexpected IPs (-want +got): %s", diff)
		}

		waitUntil(t, 3, func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s/config/apps/tls/automation/policies/0/subjects", adminAddr))
			if err != nil {
				return nil
			}

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil
			}

			gotSubjects := strings.TrimSpace(string(b))

			if diff := cmp.Diff(`["app.go-test","app2.go-test"]`, gotSubjects); diff != "" {
				return fmt.Errorf("Unexpected tls subjects (-want +got): %s", diff)
			}

			return nil
		})
	})
}

func randomPort(t *testing.T) string {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	return strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
}

func waitUntil(tb testing.TB, times int, fn func() error) {
	tb.Helper()

	var err error
	for tries := 0; tries < times; tries++ {
		err = fn()
		if err == nil {
			return
		}

		tb.Logf("Failing to execute wait func: %s", err.Error())

		time.Sleep(time.Duration(tries*tries) * time.Second)
	}

	tb.Fatal(err)
}
