package main

import (
	"context"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	AnnotationKeyEnabled = "net.guoyk.auto-fix-tke-ingress/enabled"
)

var (
	gConfig *rest.Config
	gClient *kubernetes.Clientset
)

func exit(err *error) {
	if *err != nil {
		log.Println("exited with error:", (*err).Error())
		os.Exit(1)
	} else {
		log.Println("exited")
	}
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime | log.Lmsgprefix)

	var err error
	defer exit(&err)

	if gConfig, err = rest.InClusterConfig(); err != nil {
		return
	}
	if gClient, err = kubernetes.NewForConfig(gConfig); err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error, 1)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		errChan <- routineWatchIngresses(ctx)
	}()

	select {
	case err = <-errChan:
		return
	case sig := <-sigChan:
		log.Printf("signal caught: %s", sig.String())
		cancel()
		<-errChan
	}
}

func routineWatchIngresses(ctx context.Context) (err error) {
	for {
		if err = onceWatchIngress(ctx); err != nil {
			log.Printf("failed watching ingresses: %s", err.Error())
		}

		select {
		case <-time.After(time.Second * 5):
		case <-ctx.Done():
			return
		}
	}
}

func onceWatchIngress(ctx context.Context) (err error) {
	var w watch.Interface
	if w, err = gClient.NetworkingV1beta1().Ingresses("").Watch(ctx, metav1.ListOptions{}); err != nil {
		return
	}
	for e := range w.ResultChan() {
		switch e.Type {
		case watch.Added, watch.Modified:
			ig := e.Object.(*v1beta1.Ingress)
			log.Printf("%s: %s/%s", string(e.Type), ig.Namespace, ig.Name)
		}
	}
	return
}
