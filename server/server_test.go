package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func Test_Server(t *testing.T) {
	var (
		hostRoot  = t.TempDir()
		httpAddr  = "127.0.0.1:" + randomPort(t)
		httpsAddr = "127.0.0.1:" + randomPort(t)
		adminAddr = "127.0.0.1:" + randomPort(t)
		dnsAddr   = "127.0.0.1:" + randomPort(t)
		tlds      = []string{"go-test"}
	)

	if err := ioutil.WriteFile(filepath.Join(hostRoot, "app"), []byte(adminAddr), 0o644); err != nil {
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
	go func() {
		errch <- svr.Run(context.Background())
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
		hostRoot  = t.TempDir()
		httpAddr  = "127.0.0.1:" + randomPort(t)
		httpsAddr = "127.0.0.1:" + randomPort(t)
		adminAddr = "127.0.0.1:" + randomPort(t)
		dnsAddr   = "127.0.0.1:" + randomPort(t)
		tlds      = []string{"go-test"}
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
				HttpAddr:  httpAddr,
				HttpsAddr: httpsAddr,
				AdminAddr: adminAddr,
				DnsAddr:   "invalid-addr",
			},
			WantErrMsg: "address invalid-addr: missing port in address",
		},
		{
			Name: "invalid http addr",
			Config: Config{
				HostRoot:  hostRoot,
				Domain:    tlds,
				HttpAddr:  "invalid-addr",
				HttpsAddr: httpsAddr,
				AdminAddr: adminAddr,
				DnsAddr:   dnsAddr,
			},
			WantErrMsg: "address invalid-addr: missing port in address",
		},
		{
			Name: "invalid admin addr",
			Config: Config{
				HostRoot:  hostRoot,
				Domain:    tlds,
				HttpAddr:  httpAddr,
				HttpsAddr: httpsAddr,
				AdminAddr: "invalid-addr",
				DnsAddr:   dnsAddr,
			},
			WantErrMsg: "address invalid-addr: missing port in address",
		},
		{
			Name: "invalid host root",
			Config: Config{
				HostRoot:  "invalid-host-root",
				Domain:    tlds,
				HttpAddr:  httpAddr,
				HttpsAddr: httpsAddr,
				AdminAddr: adminAddr,
				DnsAddr:   dnsAddr,
			},
			WantErrMsg: "invalid-host-root: no such file or directory",
		},
	}

	for _, c := range cases {
		cc := c
		t.Run(cc.Name, func(t *testing.T) {
			errch := make(chan error)
			srv := New(cc.Config)
			go func() {
				errch <- srv.Run(context.Background())
			}()

			select {
			case <-time.After(5 * time.Second):
				t.Fatal("error wait time out")
			case err := <-errch:
				if want, got := cc.WantErrMsg, err.Error(); !strings.Contains(got, want) {
					t.Fatalf("got error does not contain want string: want=%s, got=%s", want, got)
				}
			}
		})
	}
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
