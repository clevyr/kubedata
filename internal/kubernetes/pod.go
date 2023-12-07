package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

var ErrNoPods = errors.New("no pods in namespace")

var ErrPodNotFound = errors.New("no pods with matching label")

func (client KubeClient) GetNamespacedPods(ctx context.Context) (*v1.PodList, error) {
	pods, err := client.Pods().List(ctx, metav1.ListOptions{})
	if err != nil {
		return pods, err
	}

	if len(pods.Items) == 0 {
		return pods, fmt.Errorf("%w: %s", ErrNoPods, client.Namespace)
	}

	return pods, nil
}

type ExecOptions struct {
	Pod            v1.Pod
	Cmd            string
	Stdin          io.Reader
	Stdout, Stderr io.Writer
	TTY            bool
	SizeQueue      remotecommand.TerminalSizeQueue
	DisablePing    bool
}

func (client KubeClient) Exec(ctx context.Context, opt ExecOptions) error {
	req := client.ClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(client.Namespace).
		Name(opt.Pod.Name).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command: []string{"sh", "-c", opt.Cmd},
			Stdin:   opt.Stdin != nil,
			Stdout:  opt.Stdout != nil,
			Stderr:  opt.Stderr != nil,
			TTY:     opt.TTY,
		}, scheme.ParameterCodec)

	tlsConfig, err := rest.TLSConfigFor(client.ClientConfig)
	if err != nil {
		return err
	}
	proxy := http.ProxyFromEnvironment
	if client.ClientConfig.Proxy != nil {
		proxy = client.ClientConfig.Proxy
	}

	pingPeriod := 5 * time.Second
	if opt.DisablePing {
		pingPeriod = 0
	}
	upgradeRoundTripper := spdy.NewRoundTripperWithConfig(spdy.RoundTripperConfig{
		TLS:     tlsConfig,
		Proxier: proxy,
		// Needs to be 0 for dump/restore to prevent unexpected EOF.
		// See https://github.com/kubernetes/kubernetes/issues/60140#issuecomment-1411477275
		PingPeriod: pingPeriod,
	})
	wrapper, err := rest.HTTPWrappersForConfig(client.ClientConfig, upgradeRoundTripper)
	if err != nil {
		return err
	}

	exec, err := remotecommand.NewSPDYExecutorForTransports(wrapper, upgradeRoundTripper, "POST", req.URL())
	if err != nil {
		return err
	}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             opt.Stdin,
		Stdout:            opt.Stdout,
		Stderr:            opt.Stderr,
		Tty:               opt.TTY,
		TerminalSizeQueue: opt.SizeQueue,
	})

	return err
}

func (client KubeClient) GetPodsFiltered(ctx context.Context, queries []LabelQueryable) ([]v1.Pod, error) {
	pods, err := client.GetNamespacedPods(ctx)
	if err != nil {
		return []v1.Pod{}, err
	}
	return client.FilterPodList(pods, queries)
}

func (client KubeClient) FilterPodList(pods *v1.PodList, queries []LabelQueryable) (foundPods []v1.Pod, err error) {
	for _, query := range queries {
		var p []v1.Pod
		p, err = query.FindPods(pods)
		if errors.Is(err, ErrPodNotFound) {
			log.WithField("query", query).Trace(err)
			continue
		}
		log.WithFields(log.Fields{
			"query": query,
			"count": len(p),
		}).Trace("query returned podlist")
		foundPods = append(foundPods, p...)
	}

	if len(foundPods) == 0 {
		if errors.Is(err, ErrPodNotFound) {
			err = ErrPodNotFound
		}

		return []v1.Pod{}, err
	}
	return foundPods, nil
}
