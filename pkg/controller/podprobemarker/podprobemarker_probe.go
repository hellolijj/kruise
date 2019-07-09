package podprobemarker

import (
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/probe/http"

	"github.com/openkruise/kruise/pkg/client"
)

type prober struct {
	client clientset.Interface
	http   http.Prober
}

func newProber() (prober, error) {
	return prober{
		http:   http.New(),
		client: client.GetGenericClient().KubeClient,
	}, nil
}
