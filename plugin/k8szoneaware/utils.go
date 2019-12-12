package k8szoneaware

import (
	"sync"

	clog "github.com/coredns/coredns/plugin/pkg/log"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
)

//func GetK8sClient() (*kubernetes.Clientset, error) {
//	kubeconfig := os.Getenv("KUBECONFIG")
//	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
//	if err != nil {
//		panic(err.Error())
//	}
//
//	client, err := kubernetes.NewForConfig(config)
//	if err != nil {
//		return nil, err
//	}
//	return client, nil
//}

var AllSvcs *v1.ServiceList
var AllPods *v1.PodList
var AllNodes *v1.NodeList

func GetAllResources() {
	cli := GetK8sClient()
	AllSvcs, _ = cli.CoreV1().Services("").List(metav1.ListOptions{})
	AllPods, _ = cli.CoreV1().Pods("").List(metav1.ListOptions{})
	AllNodes, _ = cli.CoreV1().Nodes().List(metav1.ListOptions{})
}

func GetK8sClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return client
}

func GetSelectorsFromSvc(svcname, namespace string) map[string]string {
	for _, svc := range AllSvcs.Items {
		if svc.GetName() == svcname && svc.GetNamespace() == namespace {
			return svc.Spec.Selector
		}
	}
	clog.Info("Service not found...")
	return make(map[string]string)
}

func GetPodsFromSvc(namespace string, selector map[string]string, k8sClient *kubernetes.Clientset) *v1.PodList {
	set := labels.Set(selector)
	pods, err := k8sClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: set.AsSelector().String()})
	if err != nil {
		clog.Fatalf("could not get pods: %s", err)
	}
	return pods
}

func NodeZone(nodename string) string {
	for _, node := range AllNodes.Items {
		if node.GetName() == nodename {
			return node.GetLabels()["failure-domain.beta.kubernetes.io/zone"]
		}
	}
	return "nil"
}

func PodsFromZones(namespace string, pods *v1.PodList, zoneName string) []string {
	IpsFromPods := make([]string, 0)
	for _, pod := range pods.Items {
		for _, node := range AllNodes.Items {
			if pod.Spec.NodeName == node.GetName() && node.GetLabels()["failures-domain.beta.kubernetes.io/zone"] == zoneName {
				clog.Info("Entrei no if")
				IpsFromPods = append(IpsFromPods, pod.Status.PodIP)
			}
		}
	}
	return IpsFromPods
}

func PodsFromZoness(namespace string, pods *v1.PodList, zoneName string, k8sClient *kubernetes.Clientset) []string {
	concurrency := len(pods.Items)
	podCh := make(chan *v1.Pod, concurrency)
	processedPod := make(chan *v1.Pod, 1)
	var processWg sync.WaitGroup
	IpsFromPods := make([]string, 0)

	for i := 0; i < concurrency; i++ {
		go func(namespace string, pod chan *v1.Pod, processed chan *v1.Pod, zoneName string, k8sClient *kubernetes.Clientset) {
			for pod := range podCh {
				n, _ := k8sClient.CoreV1().Nodes().Get(pod.Spec.NodeName, metav1.GetOptions{})
				if n.GetLabels()["failure-domain.beta.kubernetes.io/zone"] == zoneName {
					processedPod <- pod
				} else {
					processedPod <- nil
				}
			}
		}(namespace, podCh, processedPod, zoneName, k8sClient)
	}

	go func(processed chan *v1.Pod, wg *sync.WaitGroup) {
		for p := range processedPod {
			if p == nil {
				wg.Done()
			} else {
				IpsFromPods = append(IpsFromPods, p.Status.PodIP)
				wg.Done()
			}
		}
	}(processedPod, &processWg)

	for _, pod := range pods.Items {
		processWg.Add(1)
		podCh <- &pod
	}
	processWg.Wait()

	clog.Info("IPS RETORNADOS")
	clog.Info(IpsFromPods)
	return IpsFromPods
}

func MatchSelector(selector, labels map[string]string) bool {
	t := 0
	for k := range selector {
		for lk := range labels {
			if (k == lk) && (selector[k] == labels[lk]) {
				t++
			}
		}
	}
	if len(selector) == t {
		return true
	}
	return false
}
