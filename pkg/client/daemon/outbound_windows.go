package daemon

import (
	"context"
	"net"
	"strings"

	"github.com/datawire/dlib/dgroup"

	"github.com/telepresenceio/telepresence/v2/pkg/client/daemon/dns"

	"github.com/datawire/dlib/dlog"
)

func (o *outbound) dnsServerWorker(c context.Context) error {
	o.setSearchPathFunc = func(c context.Context, paths []string) {
		namespaces := make(map[string]struct{})
		search := make([]string, 0)
		for _, path := range paths {
			if strings.ContainsRune(path, '.') {
				search = append(search, path)
			} else if path != "" {
				namespaces[path] = struct{}{}
			}
		}
		namespaces[tel2SubDomain] = struct{}{}
		o.domainsLock.Lock()
		o.namespaces = namespaces
		o.search = search
		o.domainsLock.Unlock()
		err := o.router.dev.SetDNS(c, o.dnsIP, search)
		if err != nil {
			dlog.Errorf(c, "failed to set DNS: %v", err)
		}
	}

	// Start local DNS server
	g := dgroup.NewGroup(c, dgroup.GroupConfig{})
	g.Go("Server", func(c context.Context) error {
		select {
		case <-c.Done():
			return nil
		case dnsIP := <-o.kubeDNS:
			o.dnsIP = dnsIP
			o.router.configureDNS(c, dnsIP, uint16(53), o.dnsListener.LocalAddr().(*net.UDPAddr))
		}
		defer o.dnsListener.Close()
		v := dns.NewServer(c, []net.PacketConn{o.dnsListener}, nil, o.resolveInCluster)
		return v.Run(c)
	})
	close(o.dnsConfigured)
	return g.Wait()
}
