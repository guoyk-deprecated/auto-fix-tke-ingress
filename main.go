package main

import (
	"context"
	"encoding/json"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	AnnotationKeyEnabled    = "net.guoyk.auto-fix-tke-ingress/enabled"
	AnnotationKeyHTTPRules  = "kubernetes.io/ingress.http-rules"
	AnnotationKeyHTTPsRules = "kubernetes.io/ingress.https-rules"
)

var (
	gConfig *rest.Config
	gClient *kubernetes.Clientset
)

type ShitHTTPRule struct {
	Host    string `json:"host,omitempty"`
	Path    string `json:"path"`
	Backend struct {
		ServiceName string `json:"serviceName"`
		ServicePort string `json:"servicePort"`
	} `json:"backend"`
}

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
	if w, err = gClient.ExtensionsV1beta1().Ingresses("").Watch(ctx, metav1.ListOptions{}); err != nil {
		return
	}
	for e := range w.ResultChan() {
		switch e.Type {
		case watch.Added, watch.Modified:
			ig := e.Object.(*v1beta1.Ingress)
			log.Printf("%s: %s/%s", string(e.Type), ig.Namespace, ig.Name)
			if ig.Annotations == nil {
				continue
			}
			if enabled, _ := strconv.ParseBool(ig.Annotations[AnnotationKeyEnabled]); !enabled {
				continue
			}
			var httpRules []ShitHTTPRule
			for _, r := range ig.Spec.Rules {
				if r.HTTP == nil {
					continue
				}
				for _, p := range r.HTTP.Paths {
					if p.Backend.ServicePort.Type != intstr.Int {
						log.Printf("invalid ingress service port: %s, not a int", p.Backend.ServicePort.String())
					}
					var hr ShitHTTPRule
					hr.Host = r.Host
					hr.Path = p.Path
					if hr.Path == "" {
						hr.Path = "/"
					}
					hr.Backend.ServiceName = p.Backend.ServiceName
					hr.Backend.ServicePort = strconv.Itoa(int(p.Backend.ServicePort.IntVal))
					httpRules = append(httpRules, hr)
				}
			}
			var rawHTTPRules []byte
			if rawHTTPRules, err = json.Marshal(httpRules); err != nil {
				return
			}
			patch := map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						AnnotationKeyHTTPRules: string(rawHTTPRules),
					},
				},
			}
			var rawPatch []byte
			if rawPatch, err = json.Marshal(patch); err != nil {
				return
			}

			if _, err = gClient.ExtensionsV1beta1().Ingresses(ig.Namespace).Patch(
				ctx,
				ig.Name,
				types.StrategicMergePatchType,
				rawPatch,
				metav1.PatchOptions{}); err != nil {
				return
			}
		}
	}
	return
}
