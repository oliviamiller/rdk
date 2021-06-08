package client_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/edaniels/golog"
	"github.com/miekg/dns"
	"go.viam.com/test"
	"google.golang.org/grpc"

	pb "go.viam.com/core/proto/rpc/examples/echo/v1"
	"go.viam.com/core/rpc/client"
	"go.viam.com/core/rpc/dialer"
	echoserver "go.viam.com/core/rpc/examples/echo/server"
	"go.viam.com/core/rpc/server"
	"go.viam.com/core/testutils"
	"go.viam.com/core/utils"
)

func TestDial(t *testing.T) {
	testutils.SkipUnlessInternet(t)
	logger := golog.NewTestLogger(t)

	// pure failure cases
	_, err := client.Dial(context.Background(), "::", client.DialOptions{Insecure: true}, logger)
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, "too many")

	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel1()
	_, err = client.Dial(ctx1, "127.0.0.1:1", client.DialOptions{Insecure: true}, logger)
	test.That(t, err, test.ShouldResemble, context.DeadlineExceeded)
	cancel1()

	// working and fallbacks

	rpcServer, err := server.New(logger)
	test.That(t, err, test.ShouldBeNil)

	es := echoserver.Server{}
	err = rpcServer.RegisterServiceServer(
		context.Background(),
		&pb.EchoService_ServiceDesc,
		&es,
		pb.RegisterEchoServiceHandlerFromEndpoint,
	)
	test.That(t, err, test.ShouldBeNil)

	httpListener, err := net.Listen("tcp", "localhost:0")
	test.That(t, err, test.ShouldBeNil)

	errChan := make(chan error)
	go func() {
		errChan <- rpcServer.Serve(httpListener)
	}()

	conn, err := client.Dial(context.Background(), httpListener.Addr().String(), client.DialOptions{Insecure: true}, logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, conn.Close(), test.ShouldBeNil)

	test.That(t, rpcServer.Stop(), test.ShouldBeNil)
	err = <-errChan
	test.That(t, err, test.ShouldBeNil)

	rpcServer, err = server.NewWithOptions(server.Options{server.WebRTCOptions{
		Enable:        true,
		Insecure:      true,
		SignalingHost: "yeehaw",
	}}, logger)
	test.That(t, err, test.ShouldBeNil)

	err = rpcServer.RegisterServiceServer(
		context.Background(),
		&pb.EchoService_ServiceDesc,
		&es,
		pb.RegisterEchoServiceHandlerFromEndpoint,
	)
	test.That(t, err, test.ShouldBeNil)

	httpListener, err = net.Listen("tcp", "localhost:0")
	test.That(t, err, test.ShouldBeNil)

	errChan = make(chan error)
	go func() {
		errChan <- rpcServer.Serve(httpListener)
	}()

	conn, err = client.Dial(context.Background(), httpListener.Addr().String(), client.DialOptions{Insecure: true}, logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, conn.Close(), test.ShouldBeNil)

	conn, err = client.Dial(context.Background(), "yeehaw", client.DialOptions{
		SignalingServer: httpListener.Addr().String(),
		Insecure:        true,
	}, logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, conn.Close(), test.ShouldBeNil)

	port, err := utils.TryReserveRandomPort()
	test.That(t, err, test.ShouldBeNil)
	mux := dns.NewServeMux()
	httpIP := httpListener.Addr().(*net.TCPAddr).IP
	httpPort := httpListener.Addr().(*net.TCPAddr).Port

	mux.HandleFunc("local.something.", func(rw dns.ResponseWriter, r *dns.Msg) {
		m := &dns.Msg{Compress: false}
		m.SetReply(r)

		switch r.Opcode {
		case dns.OpcodeQuery:
			for _, q := range m.Question {
				switch q.Qtype {
				case dns.TypeA:
					rr := &dns.A{
						Hdr: dns.RR_Header{
							Name:   q.Name,
							Rrtype: dns.TypeA,
							Class:  dns.ClassINET,
							Ttl:    60,
						},
						A: httpIP,
					}
					m.Answer = append(m.Answer, rr)
				}
			}
		}

		utils.UncheckedError(rw.WriteMsg(m))
	})
	dnsServer := &dns.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Net:     "udp",
		Handler: mux,
	}
	go dnsServer.ListenAndServe()

	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return net.Dial("udp", dnsServer.Addr)
		},
	}
	ctx := dialer.ContextWithResolver(context.Background(), resolver)
	ctx = dialer.ContextWithDialer(ctx, &staticDialer{httpListener.Addr().String()})
	conn, err = client.Dial(ctx, fmt.Sprintf("something:%d", httpPort), client.DialOptions{Insecure: true}, logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, conn.Close(), test.ShouldBeNil)

	test.That(t, rpcServer.Stop(), test.ShouldBeNil)
	err = <-errChan
	test.That(t, err, test.ShouldBeNil)
	test.That(t, dnsServer.Shutdown(), test.ShouldBeNil)
}

type staticDialer struct {
	address string
}

func (sd *staticDialer) Dial(ctx context.Context, target string, opts ...grpc.DialOption) (dialer.ClientConn, error) {
	return grpc.DialContext(ctx, sd.address, opts...)
}

func (sd *staticDialer) Close() error {
	return nil
}
