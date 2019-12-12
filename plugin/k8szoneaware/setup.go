package k8szoneaware

import (
	//"fmt"
	//log "github.com/sirupsen/logrus"
	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() {
	GetAllResources()
	plugin.Register("k8szoneaware", setupK8sZoneAware)
}

func setupK8sZoneAware(c *caddy.Controller) error {
	c.Next()
	if c.NextArg() {
		return plugin.Error("k8szoneaware", c.ArgErr())
	}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return K8sZoneAware{k8scli: GetK8sClient()}
	})
	return nil
}
