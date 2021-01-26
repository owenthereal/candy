package candy

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_AppService_FindApps(t *testing.T) {
	cases := []struct {
		Name     string
		Hosts    map[string]string
		TLDs     []string
		WantApps []App
		WantErr  error
	}{
		{
			Name: "valid hosts",
			Hosts: map[string]string{
				"app1": "8080",
				"app2": "192.168.0.1:9090",
				"app3": "https://192.168.0.2:9091",
				"app4": "https://owenou.com",
				"app5": "https://owenou.dev/path",
			},
			TLDs: []string{"test", "dev"},
			WantApps: []App{
				{
					Host:     "app1.test",
					Protocol: "http",
					Addr:     "127.0.0.1:8080",
				},
				{
					Host:     "app1.dev",
					Protocol: "http",
					Addr:     "127.0.0.1:8080",
				},
				{
					Host:     "app2.test",
					Protocol: "http",
					Addr:     "192.168.0.1:9090",
				},
				{
					Host:     "app2.dev",
					Protocol: "http",
					Addr:     "192.168.0.1:9090",
				},
				{
					Host:     "app3.test",
					Protocol: "https",
					Addr:     "192.168.0.2:9091",
				},
				{
					Host:     "app3.dev",
					Protocol: "https",
					Addr:     "192.168.0.2:9091",
				},
				{
					Host:     "app4.test",
					Protocol: "https",
					Addr:     "owenou.com",
				},
				{
					Host:     "app4.dev",
					Protocol: "https",
					Addr:     "owenou.com",
				},
				{
					Host:     "app5.test",
					Protocol: "https",
					Addr:     "owenou.dev",
				},
				{
					Host:     "app5.dev",
					Protocol: "https",
					Addr:     "owenou.dev",
				},
			},
			WantErr: nil,
		},
		{
			Name: "invalid hosts",
			Hosts: map[string]string{
				"app1": "invalid",
			},
			TLDs:     []string{"test"},
			WantApps: nil,
			WantErr:  nil,
		},
		{
			Name: "ignore invalid hosts",
			Hosts: map[string]string{
				"app1": "invalid",
				"app2": "8080",
			},
			TLDs: []string{"test"},
			WantApps: []App{
				{
					Host:     "app2.test",
					Protocol: "http",
					Addr:     "127.0.0.1:8080",
				},
			},
			WantErr: nil,
		},
	}

	for _, c := range cases {
		cc := c
		t.Run(cc.Name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			for k, v := range cc.Hosts {
				if err := ioutil.WriteFile(filepath.Join(dir, k), []byte(v), 0o0644); err != nil {
					t.Fatalf("error writing test hosts: %s", err)
				}
			}

			svc := NewAppService(AppServiceConfig{
				TLDs:     cc.TLDs,
				HostRoot: dir,
			})

			gotApps, gotErr := svc.FindApps()

			if !cmp.Equal(cc.WantErr, gotErr) {
				t.Fatalf("mismatch error: want=%s got=%s", cc.WantErr, gotErr)
			}

			if diff := cmp.Diff(cc.WantApps, gotApps); diff != "" {
				t.Fatalf("mismatch apps (-want +got): %s", diff)
			}
		})
	}

}
