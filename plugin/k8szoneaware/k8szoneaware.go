package k8szoneaware

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var log = clog.NewWithPlugin("k8szoneaware")

type K8sZoneAware struct {
	Next plugin.Handler
}

func (kza K8sZoneAware) Name() string { return "k8szoneaware" }

func (kza K8sZoneAware) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	log.Info("We received this query!\n")
	state := request.Request{W: w, Req: r}
	a := &dns.Msg{}
	a.SetReply(r)
	a.Authoritative = true
	questionsplit := strings.Split(state.Name(), ".")
	ip := state.IP()
	log.Infof("requester: %s", ip)

	client, _ := GetK8sClient()
	fs := fmt.Sprintf("status.podIP==%s", ip)
	requestOrigin, err := client.CoreV1().Pods("").List(metav1.ListOptions{FieldSelector: fs})
	if err != nil {
		log.Fatal(err)
	}
	sel := GetSelectorsFromSvc(questionsplit[0], questionsplit[1], client)
	log.Infof("question: %s", questionsplit)
	lp := GetPodsFromSvc(questionsplit[1], sel, client)
	nodezone := NodeZone(requestOrigin.Items[0].GetName(), requestOrigin.Items[0].GetNamespace(), client)
	log.Infof("zona do node: %s", nodezone)
	podIps := PodsFromZoness(questionsplit[1], lp, nodezone, client)
	log.Infof("ips que retornaram: %s", podIps)

	rr := new(dns.A)
	rr.Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass(), Ttl: 10}
	rr.A = net.ParseIP(podIps[0])
	log.Infof("Reply: %s", podIps[0])
	a.Extra = []dns.RR{rr}
	r.Answer = a.Extra
	w.WriteMsg(r)
	return plugin.NextOrFailure(kza.Name(), kza.Next, ctx, w, r)

}
