package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/owenthereal/candy"
	"go.uber.org/zap"
)

func Test_Server(t *testing.T) {
	var (
		hostRoot  = t.TempDir()
		httpAddr  = randomAddr(t)
		httpsAddr = randomAddr(t)
		adminAddr = randomAddr(t)
		dnsAddr   = randomAddr(t)
		tlds      = []string{"go-test"}
	)

	if err := os.WriteFile(filepath.Join(hostRoot, "app"), []byte(adminAddr), 0o644); err != nil {
		t.Fatal(err)
	}

	svr := New(Config{
		HostRoot:  hostRoot,
		Domain:    tlds,
		HttpAddr:  httpAddr,
		HttpsAddr: httpsAddr,
		AdminAddr: adminAddr,
		DnsAddr:   dnsAddr,
	})
	errch := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := svr.Run(ctx)
		if err != nil {
			candy.Log().Error("error running server", zap.Error(err))
		}

		errch <- err
	}()

	t.Run("http addr", func(t *testing.T) {
		waitUntil(t, 5*time.Second, 10, func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s/config/apps/http/servers/http/listen/0", adminAddr))
			if err != nil {
				return err
			}

			b, err := io.ReadAll(resp.Body)
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
		waitUntil(t, 5*time.Second, 10, func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s/config/apps/http/servers/https/listen/0", adminAddr))
			if err != nil {
				return err
			}

			b, err := io.ReadAll(resp.Body)
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
		waitUntil(t, 5*time.Second, 10, func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s/config/apps/tls/automation/policies/0/subjects", adminAddr))
			if err != nil {
				return nil
			}

			b, err := io.ReadAll(resp.Body)
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
		if err := os.WriteFile(filepath.Join(hostRoot, "app2"), []byte(adminAddr), 0o644); err != nil {
			t.Fatal(err)
		}

		waitUntil(t, 5*time.Second, 10, func() error {
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

			resp, err := http.Get(fmt.Sprintf("http://%s/config/apps/tls/automation/policies/0/subjects", adminAddr))
			if err != nil {
				return nil
			}

			b, err := io.ReadAll(resp.Body)
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

	t.Run("remove host root", func(t *testing.T) {
		if err := os.RemoveAll(hostRoot); err != nil {
			t.Fatal(err)
		}

		select {
		case <-time.After(5 * time.Second):
			t.Fatal("error wait time out")
		case err := <-errch:
			if want, got := fmt.Sprintf("host root %s was removed", hostRoot), err.Error(); want != got {
				t.Fatalf("unexpected error: want=%s got=%s", want, got)
			}
		}
	})
}

func Test_Server_Shutdown(t *testing.T) {
	var (
		hostRoot = t.TempDir()
		tlds     = []string{"go-test"}
	)

	cases := []struct {
		Name       string
		Config     Config
		WantErrMsg string
	}{
		{
			Name: "invalid dns addr",
			Config: Config{
				HostRoot:  hostRoot,
				Domain:    tlds,
				HttpAddr:  randomAddr(t),
				HttpsAddr: randomAddr(t),
				AdminAddr: randomAddr(t),
				DnsAddr:   "invalid-addr",
			},
			WantErrMsg: "address invalid-addr: missing port in address",
		},
		{
			Name: "invalid http addr",
			Config: Config{
				HostRoot:  hostRoot,
				Domain:    tlds,
				HttpAddr:  "",
				HttpsAddr: randomAddr(t),
				AdminAddr: randomAddr(t),
				DnsAddr:   randomAddr(t),
			},
			WantErrMsg: "loading new config: loading http app module: http: invalid configuration: invalid listener address '': missing port in address",
		},
		{
			Name: "invalid admin addr",
			Config: Config{
				HostRoot:  hostRoot,
				Domain:    tlds,
				HttpAddr:  randomAddr(t),
				HttpsAddr: randomAddr(t),
				AdminAddr: "invalid-addr",
				DnsAddr:   randomAddr(t),
			},
			WantErrMsg: "loading new config: starting caddy administration endpoint: listen tcp: lookup invalid-addr",
		},
		{
			Name: "invalid host root",
			Config: Config{
				HostRoot:  "/invalid-host-root",
				Domain:    tlds,
				HttpAddr:  randomAddr(t),
				HttpsAddr: randomAddr(t),
				AdminAddr: randomAddr(t),
				DnsAddr:   randomAddr(t),
			},
			WantErrMsg: "no such file or directory",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			errch := make(chan error)
			srv := New(c.Config)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				err := srv.Run(ctx)
				if err != nil {
					candy.Log().Error("error running server", zap.Error(err))
				}

				errch <- err
			}()

			select {
			case <-time.After(10 * time.Second):
				t.Fatal("error wait time out")
			case err := <-errch:
				if want, got := c.WantErrMsg, err.Error(); !strings.Contains(got, want) {
					t.Fatalf("got error does not contain want string: want=%s, got=%s", want, got)
				}
			}
		})
	}
}

func randomAddr(t *testing.T) string {
	return "127.0.0.1:" + randomPort(t)
}

func randomPort(t *testing.T) string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	return strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
}

func waitUntil(tb testing.TB, waitInterval time.Duration, times int, fn func() error) {
	tb.Helper()

	var err error
	for tries := 0; tries < times; tries++ {
		err = fn()
		if err == nil {
			return
		}

		time.Sleep(waitInterval)
	}

	tb.Fatal(err)
}
