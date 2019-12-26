package k8szoneaware

import (
	"fmt"

	clog "github.com/coredns/coredns/plugin/pkg/log"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
)

const (
	zoneLabel = "failure-domain.beta.kubernetes.io/zone"
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

var Nodes = make(map[string]v1.Node)
var Pods = make(map[string]v1.Pod)
var Services = make(map[string]v1.Service)

var AllSvcs *v1.ServiceList
var AllPods *v1.PodList
var AllNodes *v1.NodeList

func GetAllResources() {
	cli := GetK8sClient()
	AllSvcs, _ = cli.CoreV1().Services("").List(metav1.ListOptions{})
	AllPods, _ = cli.CoreV1().Pods("").List(metav1.ListOptions{})
	AllNodes, _ = cli.CoreV1().Nodes().List(metav1.ListOptions{})
}

func GetResources() {
	for _, service := range AllSvcs.Items {
		Services[fmt.Sprintf("%s/%s", service.Namespace, service.Name)] = service
	}
	for _, node := range AllNodes.Items {
		Nodes[node.Name] = node
	}
	for _, pod := range AllPods.Items {
		Pods[fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)] = pod
		Pods[pod.Status.PodIP] = pod
	}
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
	svc, ok := Services[fmt.Sprintf("%s/%s", namespace, svcname)]
	if !ok {
		clog.Infof("service %s/%s not found", namespace, svcname)
		return map[string]string{}
	}
	return svc.Spec.Selector
}
func GetPodsFromSvc(namespace string, selector map[string]string) []v1.Pod {
	var PodsFromSvc []v1.Pod
	for _, pod := range Pods {
		if MatchSelector(selector, pod.GetLabels()) {
			PodsFromSvc = append(PodsFromSvc, pod)
		}
	}
	return PodsFromSvc
}

func NodeZone(nodename string) string {
	node, ok := Nodes[nodename]
	if !ok {
		return ""
	}
	return node.GetLabels()[zoneLabel]
}

func PodsFromZones(namespace string, pods []v1.Pod, zoneName string) []string {
	IpsFromPods := make([]string, 0)
	for _, pod := range pods {
		n, ok := Nodes[pod.Spec.NodeName]
		if !ok {
			clog.Info("Pod's node not found")
			return IpsFromPods
		}
		if n.GetLabels()[zoneLabel] == zoneName {
			IpsFromPods = append(IpsFromPods, pod.Status.PodIP)
			return IpsFromPods
		}
	}
	return IpsFromPods
}

func MatchSelector(selector, labels map[string]string) bool {
	count := 0
	for k := range selector {
		for lk := range labels {
			if (k == lk) && (selector[k] == labels[lk]) {
				count++
			}
		}
	}
	return len(selector) == count
}
