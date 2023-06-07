<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="256" width="256" alt="cert-manager project logo" />
</p>

# ACME Local DNS Webhook

This is a [cert-manager](https://github.com/cert-manager/cert-manager) ACME DNS01 webhook solver that uses a local DNS server to solve DNS challenges. Inspired by the existing [acme-dns](https://github.com/joohoi/acme-dns) project, this webhook aims to provide a slimmer and less complex alternative.

## Background

If you are using cloud-hosted DNS, you can use one of the many existing DNS-01 solvers to automatically solve DNS challenges. If you are hosting your own PKI / ACME instance, you can use the EAB (External Account Binding) feature to solve DNS challenges. However, if you are hosting your own DNS but _not_ your own PKI / ACME instance, you need a way to solve dynamic DNS challenges for your certificates without updating your existing DNS tooling. This service aims to solve that problem.

## How it works

- Delegate a DNS zone to the webhook server. Note this must be publicly accessible.
  - eg. `acme.example.com`
- Before creating a certificate, create a DNS `CNAME` record pointing to the webhook server
  - eg. to generate a certificate for `mydomain.com`, create a `CNAME` record for `_acme-challenge.mydomain.com` pointing to `_acme-challenge.mydomain.com.acme.example.com`. Note you simply append the delegated zone to the challenge domain.
- When cert-manager creates a DNS challenge, the webhook server will receive a request to create a TXT record for `_acme-challenge.mydomain.com.acme.example.com`. Once this is validated, the webhook server will delete the TXT record, but the CNAME record will remain in place for future challenges.

## Installation

The webhook can be configured and deployed with Helm. Check `deploy/values.yaml` for the available configuration options.

If you stick with the defaults, you just need to edit the following values in the first few lines:

```yaml
# The GroupName here is used to identify your company or business unit that
# created this webhook.
# For example, this may be "acme.mycompany.com".
# This name will need to be referenced in each Issuer's `webhook` stanza to
# inform cert-manager of where to send ChallengePayload resources in order to
# solve the DNS01 challenge.
# This group name should be **unique**, hence using your own company's domain
# here is recommended.
groupName: acme.example.com
# The nameserver is the authoritative nameserver that will be returned
# in queries for the domain name. This is usually the same as the domain
nameserver: "acme.example.com."
# The domain name is the domain name / zone that will be managed by this server.
domainName: "acme.example.com."
# The rname is the email address that will be used in the SOA record, where the
# @ symbol is replaced with a dot.
rname: "admin.acme.example.com."
# The publicIP is the IP address (or CNAME) that will point to this server.
# If you don't yet have this, such as in instances where your load balancer IP
# will be dynamically assigned, you can leave this blank and update it later.
publicIP: "1.2.3.4"
```

By default, the `dnsService.Type` is `ClusterIP`. In practice, the DNS service will need to be publicly available (or at least, the records it presents need to be publicly resolvable). This means that if you do not have a larger UDP-capable ingress gateway, you will need to change this to `LoadBalancer`, and then update the `publicIP` value to the IP address (or CNAME) of the load balancer.

Once you have configured the values, you can install the webhook with:

```bash
helm upgrade -i -n cert-manager \
  cert-manager-webhook-acme-localdns \
  ./deploy/localdns \
  -f /path/to/values.yaml
```

## Usage

Once the webhook is installed, you can create an Issuer or ClusterIssuer with the following configuration:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: localdns-staging
  namespace: cert-manager
spec:
  acme:
    email: admin@example.com
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      name: cm-issuer-localdns-staging
    solvers:
    - dns01:
        cnameStrategy: Follow
        webhook:
          groupName: acme.example.com
          solverName: localdns
```

Note the `cnameStrategy` is set to `Follow`. This is required to enable the webhook to follow the CNAME record to the correct zone.

Also, note that the `groupName` and `solverName` must match the values you configured in the webhook deployment.

Now that your issuer is installed, and assuming your DNS zone is delegated to the webhook server, you can create a certificate:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example.com
  namespace: istio-system
spec:
  secretName: example.com
  issuerRef:
    name: localdns-prod
    kind: ClusterIssuer
  dnsNames:
  - 'example.com'
```

You should now be able to do a DNS lookup for `_acme-challenge.example.com` and see the CNAME record pointing to `_acme-challenge.example.com.acme.example.com`, which in turn should return the TXT record for the challenge. Once the challenge is complete, the TXT record will be deleted, but the CNAME record will remain in place for future challenges.

## Security

Whereas `acme-dns` exposes a REST interface and requires token-based auth, `localdns` exposes a Kubernetes cluster-internal webhook and uses TLS client certificates for authentication. This means that the webhook is only accessible from within the cluster, and only cert-manager is able to authenticate to the webhook.

The UDP/53 DNS service which is intended to be exposed to the public internet is a separate service from the cluster-internal webhook service, enabling further service mesh RBAC if desired. The DNS service is also configured to only respond to queries for the domain name that is managed by the webhook, and only for the TXT record that is being validated, ensuring it remains authoritative for only the domain name that is delegated to it.