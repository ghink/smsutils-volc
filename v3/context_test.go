package volc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	smserrors "go.gh.ink/smsutils/v3/errors"
	"go.gh.ink/smsutils/v3/model"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func testClient(t *testing.T, tr http.RoundTripper) model.ClientCtx {
	t.Helper()
	c, err := (Driver{}).NewClientCtx(model.DriverClientParam{Credential: map[string]string{AccessKey: "test-id", SecretKey: "test-secret", SmsAccount: "account"}, Marshal: json.Marshal, Unmarshal: json.Unmarshal}, model.ClientOptions{Transport: tr, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func response(body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

var testVars = model.Vars{{Key: "second", Value: "123456"}, {Key: "first", Value: "10"}}

func TestSendMessageCtxTransportAndCancellation(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, testTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		<-r.Context().Done()
		return nil, r.Context().Err()
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.SendMessageCtx(ctx, "+8613800138000", "sign", "template", testVars); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("cancelled request reached HTTP transport")
	}
	ctx, cancel = context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := c.SendMessageCtx(ctx, "+8613800138000", "sign", "template", testVars); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("cancellation did not stop HTTP request")
	}
	if calls.Load() != 1 {
		t.Fatal("unexpected send retries", calls.Load())
	}
}
func TestSendMessageCtxProviderResultsAndLegacyMethod(t *testing.T) {
	var body = `{"ResponseMetadata":{"RequestId":"request"},"Result":{"MessageID":["message"]}}`
	var calls atomic.Int32
	c := testClient(t, testTransport(func(r *http.Request) (*http.Response, error) { calls.Add(1); return response(body), nil }))
	for _, send := range []func() error{
		func() error {
			return c.SendMessageCtx(context.Background(), "+8613800138000", "sign", "template", testVars)
		},
		func() error { return c.SendMessage("+8613800138000", "sign", "template", testVars) },
	} {
		if err := send(); err != nil {
			t.Fatal(err)
		}
	}
	body = `{"ResponseMetadata":{"RequestId":"request","Error":{"Code":"Rejected","Message":"rejected"}}}`
	if err := c.SendMessageCtx(context.Background(), "+8613800138000", "sign", "template", testVars); err == nil {
		t.Fatal("provider failure accepted")
	}
	before := calls.Load()
	if err := c.SendMessageCtx(context.Background(), "+8613800138000", "sign", "template", model.Vars{nil}); !errors.Is(err, smserrors.ErrMessageInvalid) {
		t.Fatal(err)
	}
	if calls.Load() != before {
		t.Fatal("invalid vars reached provider")
	}
}
func TestConcurrentContextsAreIsolated(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	c := testClient(t, testTransport(func(r *http.Request) (*http.Response, error) {
		started <- struct{}{}
		select {
		case <-r.Context().Done():
			return nil, r.Context().Err()
		case <-release:
			return response(`{"ResponseMetadata":{"RequestId":"request"},"Result":{"MessageID":["message"]}}`), nil
		}
	}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first, second := make(chan error, 1), make(chan error, 1)
	go func() { first <- c.SendMessageCtx(ctx, "+8613800138000", "sign", "template", testVars) }()
	go func() {
		second <- c.SendMessageCtx(context.Background(), "+8613800138000", "sign", "template", testVars)
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("transport not reached")
		}
	}
	cancel()
	select {
	case err := <-first:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not return")
	}
	close(release)
	select {
	case err := <-second:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("other request was affected")
	}
}

func TestMalformedProviderResponsesNeverSucceed(t *testing.T) {
	for _, body := range []string{"", "null", "{}", "not-json"} {
		t.Run(body, func(t *testing.T) {
			c := testClient(t, testTransport(func(*http.Request) (*http.Response, error) { return response(body), nil }))
			if err := c.SendMessageCtx(context.Background(), "+8613800138000", "sign", "template", testVars); err == nil {
				t.Fatal("malformed provider response accepted")
			}
		})
	}
}
func TestRejectedRequestIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, testTransport(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		r := response("{}")
		r.StatusCode = 503
		return r, nil
	}))
	if err := c.SendMessageCtx(context.Background(), "+8613800138000", "sign", "template", testVars); err == nil {
		t.Fatal("server failure accepted")
	}
	if calls.Load() != 1 {
		t.Fatal("SMS was retried", calls.Load())
	}
}
