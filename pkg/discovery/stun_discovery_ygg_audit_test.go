package discovery

import (
	"net"
	"testing"
)

// The ygg= value must yield a dialable host and port, both in the documented
// unbracketed form (the port is the text after the last colon) and in the
// standard bracketed [addr]:port form.
func TestRecordHostPortAudit(t *testing.T) {
	for _, tc := range []struct{ txt, host, port string }{
		{"ygg=200:abc:def::1:5000 v=1", "200:abc:def::1", "5000"},
		{"ygg=201:1:2:3::9:6000", "201:1:2:3::9", "6000"},
		{"ygg=200::1:5000 v=2", "200::1", "5000"},
		{"ygg=[200:abc:def::1]:5000 v=1", "200:abc:def::1", "5000"},
	} {
		rec, err := Parse(tc.txt)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.txt, err)
		}
		host, port, err := rec.HostPort()
		if err != nil || host != tc.host || port != tc.port {
			t.Fatalf("Parse(%q).HostPort() = %q, %q, %v; want %q, %q", tc.txt, host, port, err, tc.host, tc.port)
		}
		addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, port))
		if err != nil || !addr.IP.Equal(net.ParseIP(tc.host)) || addr.Port == 0 {
			t.Fatalf("Parse(%q) host/port not dialable: %v, %v", tc.txt, addr, err)
		}
	}
}

// Parse must reject a ygg= value that is not an IP address plus a port, so
// Lookup skips it instead of returning an endpoint nobody can dial.
func TestParseRejectsBadYggAudit(t *testing.T) {
	for _, txt := range []string{
		"ygg=hello v=1",
		"ygg=200:abc::1:99999 v=1",
		"ygg=200:abc::1:0 v=1",
		"ygg=200:abc::1:http v=1",
		"ygg=[200:abc::1] v=1",
		"ygg=:5000 v=1",
		"ygg=example.com:5000 v=1",
	} {
		if rec, err := Parse(txt); err == nil {
			t.Errorf("Parse(%q) = %+v, want error", txt, rec)
		}
	}

	m := MapResolver{Name("dave", "example.com"): {"ygg=garbage v=1", "ygg=[200::7]:5000 v=1"}}
	rec, err := Lookup(m, "dave@example.com")
	if err != nil || rec.Ygg != "[200::7]:5000" {
		t.Fatalf("Lookup dave = %+v, %v; want the valid second record", rec, err)
	}
}
