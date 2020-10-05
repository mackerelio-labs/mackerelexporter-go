package mackerel

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mackerelio/mackerel-client-go"
)

func TestHandler_ServeHTTP(t *testing.T) {
	var c handlerClient
	c.snapshot = []*mackerel.HostMetricValue{
		{
			HostID: "1234",
			MetricValue: &mackerel.MetricValue{
				Name:  "custom.request_latencies.index",
				Value: 12345678.91234,
				Time:  1601862222,
			},
		},
		{
			HostID: "1234",
			MetricValue: &mackerel.MetricValue{
				Name:  "custom.requests.count",
				Value: 1000,
				Time:  1601862222,
			},
		},
	}

	r := httptest.NewRequest("GET", "http://localhost/metrics", nil)
	w := httptest.NewRecorder()
	c.ServeHTTP(w, r)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d; want %d", resp.StatusCode, http.StatusOK)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"request_latencies.index\t12345678.912340\t1601862222",
		"requests.count\t1000\t1601862222",
		"",
	}, "\n")
	if s := string(b); s != want {
		t.Errorf("Body = %q; want %q", s, want)
	}
}
