package routing_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/gonstruct/core/routing"
	"github.com/gonstruct/core/routing/response"

	"github.com/gin-gonic/gin"
)

// A server asked to stop takes no new connections but answers the request
// it is already serving, the way a deploy that swaps containers needs.
func TestRunFinishesTheRequestsInFlightWhenStopped(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOST", "127.0.0.1")
	t.Setenv("PORT", fmt.Sprint(port))
	t.Setenv("SHUTDOWN_TIMEOUT", "5s")

	started := make(chan struct{})
	server := routing.New(routing.WithRouters(func(engine *routing.Engine) {
		engine.GET("/slow", func(*gin.Context) response.Response {
			close(started)
			time.Sleep(300 * time.Millisecond)

			return response.Success(response.WithRaw(map[string]any{"done": true}))
		})
	}))

	ran := make(chan error, 1)
	go func() { ran <- server.Run() }()

	address := fmt.Sprintf("http://127.0.0.1:%d/slow", port)
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		connection, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			_ = connection.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the server never started listening")
		}
	}

	answered := make(chan string, 1)
	go func() {
		reply, err := http.Get(address) //nolint:noctx // a plain request is what is being served
		if err != nil {
			answered <- "failed: " + err.Error()
			return
		}
		defer reply.Body.Close()
		body, _ := io.ReadAll(reply.Body)
		answered <- fmt.Sprintf("%d %s", reply.StatusCode, body)
	}()

	<-started
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-answered:
		if got != `200 {"done":true}` {
			t.Fatalf("expected the request in flight answered, got %s", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the request in flight was never answered")
	}
	select {
	case err := <-ran:
		if err != nil {
			t.Fatalf("expected the server to stop cleanly, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the server did not stop")
	}
	if _, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second); err == nil {
		t.Fatal("expected the stopped server to take no new connections")
	}
}
