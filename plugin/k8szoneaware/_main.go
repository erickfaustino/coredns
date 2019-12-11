package k8szoneaware

import (
	"fmt"
	"time"
)

func main() {
	d := time.Now()
	fmt.Printf("Begin Main: %s \n", d.String())
	client, _ := GetK8sClient()
	ipdocaraio, err := client.CoreV1().Pods("").List(metav1.ListOptions{FieldSelector: "status.podIP=172.20.158.4"})
	fmt.Printf("IMPRIME O ERRO AQUI:   %s - %s\n", ipdocaraio.Items[0].GetName(), err)
	sel := GetSelectorsFromSvc("cluster-autoscaler", "kube-system", client)
	lp := GetPodsFromSvc("kube-system", sel, client)
	krl := PodsFromZones("kube-system", lp, "us-east-1b", client)
	fmt.Println(krl)
	f := time.Now()
	fmt.Printf("End main: %s \n", f.String())
}
