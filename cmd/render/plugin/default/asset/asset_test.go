package asset

import (
	"net"
	"strings"
	"testing"
)

type f struct {
	M string
}

func (f *f) String() string {
	return f.M
}

func TestJoinStringsFromSliceOrSingle(t *testing.T) {
	var out string

	testSingle := &f{M: "hello"}
	emptySingle := &f{M: ""}
	testSlice := []string{"Hello", "World"}

	if out = joinStringsFromSliceOrSingle(nil, testSingle); out != testSingle.String() {
		t.Errorf("single-only test failed: expected '%s', got '%s'", testSingle, out)
	}

	if out = joinStringsFromSliceOrSingle(testSlice, emptySingle); out != strings.Join(testSlice, ",") {
		t.Errorf("slice-only test failed: expected '%s', got '%s'", strings.Join(testSlice, ","), out)
	}

	if out = joinStringsFromSliceOrSingle(testSlice, testSingle); out != strings.Join(testSlice, ",") {
		t.Errorf("single+slice test failed: expected '%s', got '%s'", strings.Join(testSlice, ","), out)
	}

	if out = joinStringsFromSliceOrSingle(nil, emptySingle); out != "" {
		t.Errorf("empty test failed: expected '%s', got '%s'", "", out)
	}
}

func TestContainsNonLocalIPv6(t *testing.T) {
	ipv4Loopback := net.ParseIP("127.0.0.1")
	ipv4PublicUnicast := net.ParseIP("8.8.8.8")
	ipv4PrivateUnicast := net.ParseIP("192.168.1.2")
	ipv4Multicast := net.ParseIP("224.10.20.30")

	ipv6Loopback := net.ParseIP("::1")
	ipv6Unicast := net.ParseIP("2001:db8::1")
	ipv6LinkLocal := net.ParseIP("fe80::db8:2")
	ipv6Multicast := net.ParseIP("ff00::db8:3")

	if containsNonLocalIPv6(nil) {
		t.Errorf("empty set failed")
	}
	if containsNonLocalIPv6([]net.IP{ipv4Loopback, ipv4PublicUnicast, ipv4PrivateUnicast, ipv4Multicast, ipv6Loopback}) {
		t.Errorf("all-false set check failed")
	}
	if !containsNonLocalIPv6([]net.IP{ipv6Unicast}) {
		t.Errorf("single-true set failed")
	}
	if !containsNonLocalIPv6([]net.IP{ipv6LinkLocal, ipv4Loopback}) {
		t.Errorf("true+v4Loop set failed")
	}
	if !containsNonLocalIPv6([]net.IP{ipv6Loopback, ipv6Multicast}) {
		t.Errorf("true+v6Loop set failed")
	}
}

func TestStringerSlice(t *testing.T) {
	var out []string

	if out = stringerSlice(nil); out != nil {
		t.Errorf("nil input test did not have nil output: %v", out)
	}

	testNilSlice := []net.IP{}
	out = stringerSlice(testNilSlice)
	if len(out) != len(testNilSlice) {
		t.Errorf("output mismatch (%d) for testNilSlice (%d)", len(out), len(testNilSlice))
	}

	testNonStringerSlice := []int{1, 2, 3, 4}
	out = stringerSlice(testNonStringerSlice)
	for i, s := range out {
		if s != "" {
			t.Errorf("nonStringerSlice[%d] produced non-nil output: %s", i, s)
		}
	}

	testNormal1 := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("0.0.0.0")}
	out = stringerSlice(testNormal1)
	if len(out) != len(testNormal1) {
		t.Errorf("output mismatch (%d) for testNormal1 (%d)", len(out), len(testNormal1))
	}
	if out[0] != testNormal1[0].String() {
		t.Errorf("output index 0 mismatch (%s) on testNormal (%s)", out[0], testNormal1[0].String())
	}
	if out[1] != testNormal1[1].String() {
		t.Errorf("output index 1 mismatch (%s) on testNormal (%s)", out[1], testNormal1[1].String())
	}
}
