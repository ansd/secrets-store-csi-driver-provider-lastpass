package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ansd/secrets-store-csi-driver-provider-lastpass/server"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	socketPath := "/etc/kubernetes/secrets-store-csi-providers/lastpass.sock"
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		klog.Fatalf("Failed to listen on unix socket %s: %v", socketPath, err)
	}
	defer listener.Close()

	grpcSrv := grpc.NewServer()
	v1alpha1.RegisterCSIDriverProviderServer(grpcSrv, &server.CSIDriverProviderServer{})

	//Gracefully terminate server on shutdown unix signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigs
		klog.Infof("received signal %s to terminate", sig)
		grpcSrv.GracefulStop()
	}()

	klog.Infof("listening for connections on address: %v", listener.Addr())
	if err := grpcSrv.Serve(listener); err != nil {
		klog.Fatalf("Failure serving incoming mount requests. error: %v", err)
	}
}
