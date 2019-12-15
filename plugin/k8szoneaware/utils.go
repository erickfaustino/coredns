package k8szoneaware

import (
	"fmt"

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

var Nodes map[string]*v1.Node
var Pods map[string]*v1.Pod
var Services map[string]*v1.Service

func GetAllResources() {
	cli := GetK8sClient()
	AllSvcs, _ = cli.CoreV1().Services("").List(metav1.ListOptions{})
	AllPods, _ = cli.CoreV1().Pods("").List(metav1.ListOptions{})
	AllNodes, _ = cli.CoreV1().Nodes().List(metav1.ListOptions{})
}

func GetResources() {
	for _, service := range AllSvcs.Items {
		Services[fmt.Sprintf("%s/%s", service.GetNamespace(), service.GetName())] = &service
	}
	for _, node := range AllNodes.Items {
		Nodes[node.GetName()] = &node
	}
	for _, pod := range AllPods.Items {
		Pods[fmt.Sprintf("%s/%s", pod.GetNamespace(), pod.GetName())] = &pod
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
func GetPodsFromSvc(namespace string, selector map[string]string, k8sClient *kubernetes.Clientset) *v1.PodList {
	set := labels.Set(selector)
	pods, err := k8sClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: set.AsSelector().String()})
	if err != nil {
		clog.Fatalf("could not get pods: %s", err)
	}
	return pods
}

func NodeZone(nodename string) string {
	node, ok := Nodes[nodename]
	if !ok {
		return ""
	}
	return node.GetLabels()["failure-domain.beta.kubernetes.io/zone"]
}

func PodsFromZones(namespace string, pods *v1.PodList, zoneName string) []string {
	IpsFromPods := make([]string, 0)
	for _, pod := range pods.Items {
		n, ok := Nodes[pod.Spec.NodeName]
		if !ok {
			clog.Info("Pod's node not found")
		}
		if n.GetLabels()["failure-domain.beta.kubernetes.io/zone"] == zoneName {
			IpsFromPods = append(IpsFromPods, pod.Status.PodIP)
		}
	}
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
