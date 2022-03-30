package tlsutil

import (
	"errors"
	"net"

	"golang.org/x/net/idna"
)

// SanitizeName converts the given domain to its ASCII form.
func SanitizeName(domain string) (string, error) {
	if domain == "" {
		return "", errors.New("empty server name")
	}

	// Note that this conversion is necessary because some server names in the handshakes
	// started by some clients (such as cURL) are not converted to Punycode, which will
	// prevent us from obtaining certificates for them. In addition, we should also treat
	// example.com and EXAMPLE.COM as equivalent and return the same certificate for them.
	// Fortunately, this conversion also helped us deal with this kind of mixedcase problems.
	//
	// Due to the "σςΣ" problem (see https://unicode.org/faq/idn.html#22), we can't use
	// idna.Punycode.ToASCII (or just idna.ToASCII) here.
	name, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		return "", errors.New("server name contains invalid character")
	}

	return name, nil
}

// SanitizeHost returns the ASCII form of the host part in a host:port address.
func SanitizeHost(host string) (string, error) {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return SanitizeName(h)
	}
	return SanitizeName(host)
}
