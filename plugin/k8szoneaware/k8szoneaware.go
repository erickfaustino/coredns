package k8szoneaware

import (
	"context"
	"net"
	"strings"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"k8s.io/client-go/kubernetes"
)

var log = clog.NewWithPlugin("k8szoneaware")

type K8sZoneAware struct {
	Next   plugin.Handler
	k8scli *kubernetes.Clientset
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

	requestOrigin, ok := Pods[ip]
	if !ok {
		log.Fatalf("pod with IP %s not found", ip)
	}
	sel := GetSelectorsFromSvc(questionsplit[0], questionsplit[1])
	log.Infof("question: %s", questionsplit)
	log.Infof("selector: %s", sel)
	lp := GetPodsFromSvc(questionsplit[1], sel)
	log.Info(lp)
	nodezone := NodeZone(requestOrigin.Spec.NodeName)
	log.Infof("zona do node: %s", nodezone)
	podIps := PodsFromZones(questionsplit[1], lp, nodezone)
	log.Infof("ips que retornaram: %s", podIps)

	rr := new(dns.A)
	rr.Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass(), Ttl: 10}
	rr.A = net.ParseIP(podIps[0])
	clog.Infof("Reply: %s", podIps[0])
	a.Extra = []dns.RR{rr}
	r.Answer = a.Extra
	w.WriteMsg(r)
	return plugin.NextOrFailure(kza.Name(), kza.Next, ctx, w, r)

}
