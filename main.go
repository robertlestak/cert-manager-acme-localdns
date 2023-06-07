package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	"github.com/cert-manager/webhook-example/internal/store"
)

var GroupName = os.Getenv("GROUP_NAME")

func init() {
	ll, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		ll = log.InfoLevel
	}
	log.SetLevel(ll)
}

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}
	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	solver := &localDNSProviderSolver{}
	if err := solver.Init(); err != nil {
		panic(err)
	}
	cmd.RunWebhookServer(GroupName,
		solver,
	)
}

func (s *localDNSProviderSolver) Init() error {
	l := log.WithFields(log.Fields{
		"storeType":   s.StoreType,
		"storeConfig": s.StoreConfig,
		"domainName":  s.DomainName,
		"nameserver":  s.Nameserver,
		"dnsPort":     s.DNSPort,
	})
	l.Debug("initializing")
	if os.Getenv("DNS_PORT") != "" {
		pv, err := strconv.Atoi(os.Getenv("DNS_PORT"))
		if err != nil {
			panic(err)
		}
		s.DNSPort = pv
	} else {
		s.DNSPort = 53
	}
	if s.Nameserver == "" && os.Getenv("NAMESERVER") != "" {
		s.Nameserver = os.Getenv("NAMESERVER")
	}
	if s.DomainName == "" && os.Getenv("DOMAIN_NAME") != "" {
		s.DomainName = os.Getenv("DOMAIN_NAME")
	}
	if s.PublicIP == "" && os.Getenv("PUBLIC_IP") != "" {
		s.PublicIP = os.Getenv("PUBLIC_IP")
	}
	if s.Nameserver == "" && s.DomainName != "" {
		s.Nameserver = s.DomainName
	}
	if s.DomainName == "" && s.Nameserver != "" {
		s.DomainName = s.Nameserver
	}
	if s.RName == "" && os.Getenv("RNAME") != "" {
		s.RName = os.Getenv("RNAME")
	}
	if s.RName == "" {
		s.RName = "hostmaster." + s.DomainName
	}
	if !strings.HasSuffix(s.RName, ".") {
		s.RName = s.RName + "."
	}
	if !strings.HasSuffix(s.DomainName, ".") {
		s.DomainName = s.DomainName + "."
	}
	if !strings.HasSuffix(s.Nameserver, ".") {
		s.Nameserver = s.Nameserver + "."
	}
	if s.StoreType == "" && os.Getenv("STORE_TYPE") != "" {
		s.StoreType = store.StoreType(os.Getenv("STORE_TYPE"))
	}
	if len(s.StoreConfig) == 0 && os.Getenv("STORE_CONFIG") != "" {
		err := json.Unmarshal([]byte(os.Getenv("STORE_CONFIG")), &s.StoreConfig)
		if err != nil {
			l.WithError(err).Error("failed to unmarshal store config")
			return err
		}
	}
	// envsubst the store config
	for k, v := range s.StoreConfig {
		if sv, ok := v.(string); ok {
			s.StoreConfig[k] = os.ExpandEnv(sv)
		}
	}
	var err error
	s.store, err = store.Init(s.StoreType, s.StoreConfig)
	if err != nil {
		l.WithError(err).Error("failed to initialize store")
		return err
	}
	l.Debug("initialized")
	return nil
}

// localDNSProviderSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/cert-manager/cert-manager/pkg/acme/webhook.Solver`
// interface.
type localDNSProviderSolver struct {
	// If a Kubernetes 'clientset' is needed, you must:
	// 1. uncomment the additional `client` field in this structure below
	// 2. uncomment the "k8s.io/client-go/kubernetes" import at the top of the file
	// 3. uncomment the relevant code in the Initialize method below
	// 4. ensure your webhook's service account has the required RBAC role
	//    assigned to it for interacting with the Kubernetes APIs you need.
	//client kubernetes.Clientset
	// StoreType   store.StoreType `json:"storeType"`
	// StoreConfig map[string]any  `json:"storeConfig"`
	// DomainName  string          `json:"domainName"`
	DNSPort     int             `json:"dnsPort"`
	Nameserver  string          `json:"nameserver"`
	RName       string          `json:"rname"`
	StoreType   store.StoreType `json:"storeType"`
	StoreConfig map[string]any  `json:"storeConfig"`
	DomainName  string          `json:"domainName"`
	PublicIP    string          `json:"publicIP"`

	server *dns.Server `json:"-"`
	store  store.Store `json:"-"`
}

// localDNSProviderConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
// This typically includes references to Secret resources containing DNS
// provider credentials, in cases where a 'multi-tenant' DNS solver is being
// created.
// If you do *not* require per-issuer or per-certificate configuration to be
// provided to your webhook, you can skip decoding altogether in favour of
// using CLI flags or similar to provide configuration.
// You should not include sensitive information here. If credentials need to
// be used by your provider here, you should reference a Kubernetes Secret
// resource and fetch these credentials using a Kubernetes clientset.
type localDNSProviderConfig struct {
	// Change the two fields below according to the format of the configuration
	// to be decoded.
	// These fields will be set by users in the
	// `issuer.spec.acme.dns01.providers.webhook.config` field.

	//Email           string `json:"email"`
	//APIKeySecretRef v1alpha1.SecretKeySelector `json:"apiKeySecretRef"`
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (c *localDNSProviderSolver) Name() string {
	return "localdns"
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (c *localDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	l := log.WithFields(log.Fields{
		"fqdn": ch.ResolvedFQDN,
		"type": ch.ResolvedZone,
		"key":  ch.Key,
	})
	l.Debug("Presenting record")
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}
	l = l.WithField("config", cfg)
	l.Debug("Loaded configuration")
	if err := c.store.Set(ch.ResolvedFQDN, []byte(ch.Key)); err != nil {
		return err
	}
	dn := c.DomainName
	if !strings.HasSuffix(dn, ".") {
		dn = dn + "."
	}
	fqdn := strings.ToLower(ch.ResolvedFQDN + dn)
	if err := c.store.Set(fqdn, []byte(ch.Key)); err != nil {
		return err
	}
	return nil
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (c *localDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	l := log.WithFields(log.Fields{
		"fqdn": ch.ResolvedFQDN,
		"type": ch.ResolvedZone,
		"key":  ch.Key,
	})
	l.Debug("Presenting record")
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}
	l = l.WithField("config", cfg)
	l.Debug("Loaded configuration")
	if err := c.store.Delete(ch.ResolvedFQDN); err != nil {
		return err
	}
	dn := c.DomainName
	if !strings.HasSuffix(dn, ".") {
		dn = dn + "."
	}
	fqdn := strings.ToLower(ch.ResolvedFQDN + dn)
	if err := c.store.Delete(fqdn); err != nil {
		return err
	}
	return nil
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (c *localDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	c.server = &dns.Server{
		Addr:    fmt.Sprintf(":%d", c.DNSPort),
		Net:     "udp",
		Handler: dns.HandlerFunc(c.handleDNSRequest),
	}
	go func(done <-chan struct{}) {
		<-done
		if err := c.server.Shutdown(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		}
	}(stopCh)
	go func() {
		if err := c.server.ListenAndServe(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(1)
		}
	}()
	///// UNCOMMENT THE BELOW CODE TO MAKE A KUBERNETES CLIENTSET AVAILABLE TO
	///// YOUR CUSTOM DNS PROVIDER

	//cl, err := kubernetes.NewForConfig(kubeClientConfig)
	//if err != nil {
	//	return err
	//}
	//
	//c.client = cl

	///// END OF CODE TO MAKE KUBERNETES CLIENTSET AVAILABLE
	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (localDNSProviderConfig, error) {
	cfg := localDNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}
