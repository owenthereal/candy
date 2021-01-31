package candy

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

type App struct {
	Host string
	Addr string
}

type AppServiceConfig struct {
	TLDs     []string
	HostRoot string
}

func NewAppService(cfg AppServiceConfig) *AppService {
	return &AppService{cfg: cfg}
}

type AppService struct {
	cfg AppServiceConfig
}

func (f *AppService) FindApps() ([]App, error) {
	files, err := ioutil.ReadDir(f.cfg.HostRoot)
	if err != nil {
		return nil, err
	}

	var result []App

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		b, err := ioutil.ReadFile(filepath.Join(f.cfg.HostRoot, file.Name()))
		if err != nil {
			return nil, err
		}

		apps, err := f.parseApps(file.Name(), strings.TrimSpace(string(b)))
		if err != nil {
			continue
		}

		result = append(result, apps...)
	}

	return result, nil
}

func (f *AppService) parseApps(domain, data string) ([]App, error) {
	// port
	port, err := strconv.Atoi(data)
	if err == nil {
		return f.buildApps(domain, fmt.Sprintf("127.0.0.1:%d", port)), nil
	}

	// http://ip:port
	u, err := url.ParseRequestURI(data)
	if err == nil {
		return f.buildApps(domain, u.Host), nil
	}

	// ip:port
	host, sport, err := net.SplitHostPort(data)
	if err == nil {
		return f.buildApps(domain, host+":"+sport), nil
	}

	// TODO: json
	return nil, fmt.Errorf("invalid domain for file: %s", filepath.Join(f.cfg.HostRoot, domain))
}

func (f *AppService) buildApps(domain, addr string) []App {
	var apps []App
	for _, tld := range f.cfg.TLDs {
		apps = append(apps, App{
			Host: domain + "." + tld, // e.g., app.test
			Addr: addr,
		})
	}

	return apps
}
