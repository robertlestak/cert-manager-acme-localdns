package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/cert-manager/cert-manager/test/acme/dns"
	"github.com/cert-manager/webhook-example/internal/store"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
)

func TestRunsSuite(t *testing.T) {
	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.
	//

	// Uncomment the below fixture when implementing your custom DNS provider
	//fixture := dns.NewFixture(&customDNSProviderSolver{},
	//	dns.SetResolvedZone(zone),
	//	dns.SetAllowAmbientCredentials(false),
	//	dns.SetManifestPath("testdata/my-custom-solver"),
	//	dns.SetBinariesPath("_test/kubebuilder/bin"),
	//)
	//solver := localdns.New("59351")
	//fixture := dns.NewFixture(solver,
	solver := &localDNSProviderSolver{
		DNSPort:    59351,
		Nameserver: "acme.com.",
		DomainName: "acme.com.",
		PublicIP:   "127.0.0.1",
		StoreType:  store.StoreTypeSqlite,
		StoreConfig: map[string]any{
			"path": "/tmp/test.db",
		},
	}
	if err := solver.Init(); err != nil {
		t.Fatal(err)
	}
	fixture := dns.NewFixture(solver,
		dns.SetResolvedZone("example.com."),
		dns.SetManifestPath("testdata/localdns"),
		dns.SetDNSServer(fmt.Sprintf("127.0.0.1:%d", solver.DNSPort)),
		dns.SetUseAuthoritative(false),
	)
	//need to uncomment and  RunConformance delete runBasic and runExtended once https://github.com/cert-manager/cert-manager/pull/4835 is merged
	//fixture.RunConformance(t)
	fixture.RunBasic(t)
	fixture.RunExtended(t)

}
