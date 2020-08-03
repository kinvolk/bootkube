package asset

import (
	"crypto/rsa"
	"crypto/x509"
	"net"
	"net/url"

	"github.com/pborman/uuid"

	"github.com/kubernetes-sigs/bootkube/pkg/tlsutil"
)

// TLS organizations map to Kubernetes groups, and "system:masters"
// is a well-known Kubernetes group that gives a user admin power.
const orgSystemMasters = "system:masters"

func newTLSAssets(caCert *x509.Certificate, caPrivKey *rsa.PrivateKey, altNames tlsutil.AltNames) ([]Asset, error) {
	var (
		assets []Asset
		err    error
	)

	apiKey, apiCert, err := newAPIKeyAndCert(caCert, caPrivKey, altNames)
	if err != nil {
		return assets, err
	}

	aggregatorCAPrivKey, err := tlsutil.NewPrivateKey()
	if err != nil {
		return assets, err
	}

	cfg := tlsutil.CertConfig{
		CommonName: "front-proxy",
	}

	aggregatorCACert, err := tlsutil.NewSelfSignedCACertificate(cfg, aggregatorCAPrivKey)
	if err != nil {
		return assets, err
	}

	frontProxyPrivKey, err := tlsutil.NewPrivateKey()
	if err != nil {
		return assets, err
	}

	cfg = tlsutil.CertConfig{
		CommonName: "front-proxy-client",
	}

	frontProxyCert, err := tlsutil.NewSignedCertificate(cfg, frontProxyPrivKey, aggregatorCACert, aggregatorCAPrivKey)
	if err != nil {
		return assets, err
	}

	kubeletClientCertConfig := tlsutil.CertConfig{
		CommonName:   "apiserver-kubelet-client",
		Organization: []string{orgSystemMasters},
	}

	kubeletClientKey, kubeletClientCert, err := newAdminKeyAndCert(caCert, caPrivKey, kubeletClientCertConfig)
	if err != nil {
		return assets, err
	}

	saPrivKey, err := tlsutil.NewPrivateKey()
	if err != nil {
		return assets, err
	}

	saPubKey, err := tlsutil.EncodePublicKeyPEM(&saPrivKey.PublicKey)
	if err != nil {
		return assets, err
	}

	adminCertConfig := tlsutil.CertConfig{
		CommonName:   "admin",
		Organization: []string{orgSystemMasters},
	}
	adminKey, adminCert, err := newAdminKeyAndCert(caCert, caPrivKey, adminCertConfig)
	if err != nil {
		return assets, err
	}

	assets = append(assets, []Asset{
		{Name: AssetPathCAKey, Data: tlsutil.EncodePrivateKeyPEM(caPrivKey)},
		{Name: AssetPathCACert, Data: tlsutil.EncodeCertificatePEM(caCert)},
		{Name: AssetPathAPIServerKey, Data: tlsutil.EncodePrivateKeyPEM(apiKey)},
		{Name: AssetPathAPIServerCert, Data: tlsutil.EncodeCertificatePEM(apiCert)},
		{Name: AssetPathServiceAccountPrivKey, Data: tlsutil.EncodePrivateKeyPEM(saPrivKey)},
		{Name: AssetPathAggregatorCA, Data: tlsutil.EncodeCertificatePEM(aggregatorCACert)},
		{Name: AssetPathFrontProxyClientCert, Data: tlsutil.EncodeCertificatePEM(frontProxyCert)},
		{Name: AssetPathFrontProxyClientKey, Data: tlsutil.EncodePrivateKeyPEM(frontProxyPrivKey)},
		{Name: AssetPathKubeletClientKey, Data: tlsutil.EncodePrivateKeyPEM(kubeletClientKey)},
		{Name: AssetPathKubeletClientCert, Data: tlsutil.EncodeCertificatePEM(kubeletClientCert)},
		{Name: AssetPathServiceAccountPubKey, Data: saPubKey},
		{Name: AssetPathAdminKey, Data: tlsutil.EncodePrivateKeyPEM(adminKey)},
		{Name: AssetPathAdminCert, Data: tlsutil.EncodeCertificatePEM(adminCert)},
	}...)
	return assets, nil
}

func newCACert() (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := tlsutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	config := tlsutil.CertConfig{
		CommonName:         "kube-ca",
		Organization:       []string{uuid.New()},
		OrganizationalUnit: []string{"bootkube"},
	}

	cert, err := tlsutil.NewSelfSignedCACertificate(config, key)
	if err != nil {
		return nil, nil, err
	}

	return key, cert, err
}

func newAPIKeyAndCert(caCert *x509.Certificate, caPrivKey *rsa.PrivateKey, altNames tlsutil.AltNames) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := tlsutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	altNames.DNSNames = append(altNames.DNSNames, []string{
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		"kubernetes.default.svc.cluster.local",
	}...)

	config := tlsutil.CertConfig{
		CommonName:   "kube-apiserver",
		Organization: []string{"kube-master"},
		AltNames:     altNames,
	}
	cert, err := tlsutil.NewSignedCertificate(config, key, caCert, caPrivKey)
	if err != nil {
		return nil, nil, err
	}
	return key, cert, err
}

func newAdminKeyAndCert(caCert *x509.Certificate, caPrivKey *rsa.PrivateKey, config tlsutil.CertConfig) (*rsa.PrivateKey, *x509.Certificate, error) {

	key, err := tlsutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	cert, err := tlsutil.NewSignedCertificate(config, key, caCert, caPrivKey)
	if err != nil {
		return nil, nil, err
	}
	return key, cert, err
}

func newEtcdTLSAssets(etcdCACert, etcdClientCert *x509.Certificate, etcdClientKey *rsa.PrivateKey, caCert *x509.Certificate, caPrivKey *rsa.PrivateKey, etcdServers []*url.URL) ([]Asset, error) {
	var assets []Asset
	if etcdCACert == nil {
		// Use the master CA to generate etcd assets.
		etcdCACert = caCert

		// Create an etcd client cert.
		var err error
		etcdClientKey, etcdClientCert, err = newEtcdKeyAndCert(caCert, caPrivKey, "etcd-client", etcdServers)
		if err != nil {
			return nil, err
		}

		// Create an etcd peer cert (not consumed by self-hosted components).
		etcdPeerKey, etcdPeerCert, err := newEtcdKeyAndCert(caCert, caPrivKey, "etcd-peer", etcdServers)
		if err != nil {
			return nil, err
		}
		etcdServerKey, etcdServerCert, err := newEtcdKeyAndCert(caCert, caPrivKey, "etcd-server", etcdServers)
		if err != nil {
			return nil, err
		}

		assets = append(assets, []Asset{
			{Name: AssetPathEtcdPeerCA, Data: tlsutil.EncodeCertificatePEM(etcdCACert)},
			{Name: AssetPathEtcdPeerKey, Data: tlsutil.EncodePrivateKeyPEM(etcdPeerKey)},
			{Name: AssetPathEtcdPeerCert, Data: tlsutil.EncodeCertificatePEM(etcdPeerCert)},
			{Name: AssetPathEtcdServerCA, Data: tlsutil.EncodeCertificatePEM(etcdCACert)},
			{Name: AssetPathEtcdServerKey, Data: tlsutil.EncodePrivateKeyPEM(etcdServerKey)},
			{Name: AssetPathEtcdServerCert, Data: tlsutil.EncodeCertificatePEM(etcdServerCert)},
		}...)
	}

	assets = append(assets, []Asset{
		{Name: AssetPathEtcdClientCA, Data: tlsutil.EncodeCertificatePEM(etcdCACert)},
		{Name: AssetPathEtcdClientKey, Data: tlsutil.EncodePrivateKeyPEM(etcdClientKey)},
		{Name: AssetPathEtcdClientCert, Data: tlsutil.EncodeCertificatePEM(etcdClientCert)},
	}...)

	return assets, nil
}

func newEtcdKeyAndCert(caCert *x509.Certificate, caPrivKey *rsa.PrivateKey, commonName string, etcdServers []*url.URL) (*rsa.PrivateKey, *x509.Certificate, error) {
	addrs := make([]string, len(etcdServers))
	for i := range etcdServers {
		addrs[i] = etcdServers[i].Hostname()
	}
	return newKeyAndCert(caCert, caPrivKey, commonName, addrs)
}

func newKeyAndCert(caCert *x509.Certificate, caPrivKey *rsa.PrivateKey, commonName string, addrs []string) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := tlsutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	var altNames tlsutil.AltNames
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil {
			altNames.IPs = append(altNames.IPs, ip)
		} else {
			altNames.DNSNames = append(altNames.DNSNames, addr)
		}
	}
	config := tlsutil.CertConfig{
		CommonName:   commonName,
		Organization: []string{"etcd"},
		AltNames:     altNames,
	}
	cert, err := tlsutil.NewSignedCertificate(config, key, caCert, caPrivKey)
	if err != nil {
		return nil, nil, err
	}
	return key, cert, err
}
