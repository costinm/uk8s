package uk8s

import (
	"encoding/base64"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/costinm/meshauth"
	"sigs.k8s.io/yaml"
)

// Kubeconfig support, for bootstraping from .kubeconfig or in-cluser k8s.

// Auth storage is loosely based on K8S kube.config:
// - no yaml - all files must be converted to JSON ( to avoid dep on yaml lib )
//   Caller can read yaml and convert before calling this.
// - default config is kube.json
// - expects that the first user is the 'default', has client cert
// - 'context' is the used to locate the URL, cert, user.

// "Clusters" are known nodes.
// WorkloadID can be the public-key based WorkloadID, or a hostname or "default"
//
// "User" has credentials - "default" is the workload WorkloadID.
//
// "Context" pairs a MeshCluster with a User.
// "default" is the upstream / primary server
//

func LoadKubeconfig() (*KubeConfig, error) {
	kc := os.Getenv("KUBECONFIG")
	if kc == "" {
		kc = os.Getenv("HOME") + "/.kube/config"
	}
	kconf := &KubeConfig{}

	var kcd []byte
	if kc != "" {
		if _, err := os.Stat(kc); err == nil {
			// Explicit kube config, using it.
			// 	"sigs.k8s.io/yaml"
			kcd, err = ioutil.ReadFile(kc)
			if err != nil {
				return nil, err
			}
			err := yaml.Unmarshal(kcd, kconf)
			if err != nil {
				return nil, err
			}

			return kconf, nil
		}
	}
	return nil, nil
}

// InitK8S will detect k8s env, and if present will load the mesh defaults and init
// authenticators.
func InitK8S(ma *meshauth.MeshAuth, kc *KubeConfig) (*meshauth.K8STokenSource, error) {
	var err error
	var def *meshauth.K8STokenSource
	if kc != nil {
		def, err = addKubeConfigClusters(ma, kc)
		if err != nil {
			return nil, err
		}
	}

	return def, nil
}

// AddKubeConfigClusters extracts supported RestClusters from the kube config, returns the default and the list
// of clusters by location.
// GKE naming conventions are assumed for extracting the location.
//
// URest is used to configure TokenProvider and as factory for the http client.
// Returns the default client and the list of non-default clients.
func addKubeConfigClusters(ma *meshauth.MeshAuth, kc *KubeConfig) (*meshauth.K8STokenSource, error) {
	var cluster *KubeCluster
	var user *KubeUser

	if kc == nil || len(kc.Clusters) == 0 || len(kc.Users) == 0 || kc.CurrentContext == "" {
		return nil, nil
	}

	for _, cc := range kc.Contexts {
		cc := cc
		// Find the cluster and user for the context
		for _, c := range kc.Clusters {
			c := c
			if c.Name == cc.Context.Cluster {
				cluster = &c.Cluster
			}
		}
		for _, c := range kc.Users {
			c := c
			if c.Name == cc.Context.User {
				user = &c.User
			}
		}
		rc, err := kubeconfig2Rest(cc.Context.Cluster, cluster, user, cc.Context.Namespace)
		if err != nil {
			log.Println("Skipping incompatible cluster ", cc.Context.Cluster, err)
		} else {
			ma.Dst[cc.Name] = rc.Dest
			ma.AuthProviders[cc.Name] = rc
		}
	}

	defc := ma.AuthProviders[kc.CurrentContext].(*meshauth.K8STokenSource)

	parts := strings.Split(kc.CurrentContext, "_")
	if parts[0] == "gke" {
		ma.Location = parts[2]
		ma.ProjectID = parts[1]
		ma.Namespace = defc.Namespace
	}

	if ma.Namespace == "" {
		ma.Namespace = "default"
	}

	return defc, nil
}

func kubeconfig2Rest(name string, cluster *KubeCluster, user *KubeUser, ns string) (*meshauth.K8STokenSource, error) {
	if ns == "" {
		ns = "default"
	}
	rc := &meshauth.K8STokenSource{
		Dest: &meshauth.Dest{
			Addr:                  cluster.Server,
			InsecureSkipTLSVerify: cluster.InsecureSkipTLSVerify,
		},
	}
	if user.Token != "" {
		rc.TokenProvider = &meshauth.StaticTokenSource{Token: user.Token}
	}
	if user.TokenFile != "" {
		rc.TokenProvider = &meshauth.FileTokenSource{TokenFile: user.TokenFile}
	}

	// May be useful to AddService: strings.HasPrefix(name, "gke_") ||
	//if user.AuthProvider.Name != "" {
	//	rc.TokenProvider = uk.AuthProviders[user.AuthProvider.Name]
	//	if rc.TokenProvider == nil {
	//		return nil, errors.New("Missing provider " + user.AuthProvider.Name)
	//	}
	//}

	// TODO: support client cert, token file (with reload)
	if cluster.CertificateAuthority != "" {
		caCert, err := os.ReadFile(cluster.CertificateAuthority)
		if err != nil {
			return nil, err
		}
		rc.CACertPEM = caCert
	}

	caCert, err := base64.StdEncoding.DecodeString(string(cluster.CertificateAuthorityData))
	if err != nil {
		return nil, err
	}
	rc.CACertPEM = caCert

	return rc, nil
}

func KubeFromEnv(ma *meshauth.MeshAuth) (*meshauth.K8STokenSource, error) {
	kc, err := LoadKubeconfig()
	if err != nil {
		return nil, err
	}
	return InitK8S(ma, kc)
}

// KubeConfig is the JSON representation of the kube config.
// The format supports most of the things we need and also allows connection to real k8s clusters.
// UGate implements a very light subset - should be sufficient to connect to K8S, but without any
// generated stubs. Based in part on https://github.com/ericchiang/k8s (abandoned), which is a light
// client.
type KubeConfig struct {
	// Must be v1
	ApiVersion string `json:"apiVersion"`
	// Must be Config
	Kind string `json:"kind"`

	// Clusters is a map of referencable names to cluster configs
	Clusters []KubeNamedCluster `json:"clusters"`

	// AuthInfos is a map of referencable names to user configs
	Users []KubeNamedUser `json:"users"`

	// Contexts is a map of referencable names to context configs
	Contexts []KubeNamedContext `json:"contexts"`

	// CurrentContext is the name of the context that you would like to use by default
	CurrentContext string `json:"current-context" yaml:"current-context"`
}

type KubeNamedCluster struct {
	Name    string      `json:"name"`
	Cluster KubeCluster `json:"cluster"`
}
type KubeNamedUser struct {
	Name string   `json:"name"`
	User KubeUser `json:"user"`
}
type KubeNamedContext struct {
	Name    string  `json:"name"`
	Context Context `json:"context"`
}

type KubeCluster struct {
	// LocationOfOrigin indicates where this object came from.  It is used for round tripping config post-merge, but never serialized.
	// +k8s:conversion-gen=false
	//LocationOfOrigin string
	// Server is the address of the kubernetes cluster (https://hostname:port).
	Server string `json:"server"`
	// InsecureSkipTLSVerify skips the validity check for the server's certificate. This will make your HTTPS connections insecure.
	// +optional
	InsecureSkipTLSVerify bool `json:"insecure-skip-tls-verify,omitempty"`
	// CertificateAuthority is the path to a cert file for the certificate authority.
	// +optional
	CertificateAuthority string `json:"certificate-authority,omitempty" yaml:"certificate-authority"`
	// CertificateAuthorityData contains PEM-encoded certificate authority certificates. Overrides CertificateAuthority
	// +optional
	CertificateAuthorityData string `json:"certificate-authority-data,omitempty"  yaml:"certificate-authority-data"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	// +optional
	//Extensions map[string]runtime.Object `json:"extensions,omitempty"`
}

// KubeUser contains information that describes identity information.  This is use to tell the kubernetes cluster who you are.
type KubeUser struct {
	// LocationOfOrigin indicates where this object came from.  It is used for round tripping config post-merge, but never serialized.
	// +k8s:conversion-gen=false
	//LocationOfOrigin string
	// ClientCertificate is the path to a client cert file for TLS.
	// +optional
	ClientCertificate string `json:"client-certificate,omitempty"`
	// ClientCertificateData contains PEM-encoded data from a client cert file for TLS. Overrides ClientCertificate
	// +optional
	ClientCertificateData []byte `json:"client-certificate-data,omitempty"`
	// ClientKey is the path to a client key file for TLS.
	// +optional
	ClientKey string `json:"client-key,omitempty"`
	// ClientKeyData contains PEM-encoded data from a client key file for TLS. Overrides ClientKey
	// +optional
	ClientKeyData []byte `json:"client-key-data,omitempty"`
	// Token is the bearer token for authentication to the kubernetes cluster.
	// +optional
	Token string `json:"token,omitempty"`
	// TokenFile is a pointer to a file that contains a bearer token (as described above).  If both Token and TokenFile are present, Token takes precedence.
	// +optional
	TokenFile string `json:"tokenFile,omitempty"`
	// Impersonate is the username to act-as.
	// +optional
	//Impersonate string `json:"act-as,omitempty"`
	// ImpersonateGroups is the groups to imperonate.
	// +optional
	//ImpersonateGroups []string `json:"act-as-groups,omitempty"`
	// ImpersonateUserExtra contains additional information for impersonated user.
	// +optional
	//ImpersonateUserExtra map[string][]string `json:"act-as-user-extra,omitempty"`
	// Username is the username for basic authentication to the kubernetes cluster.
	// +optional
	Username string `json:"username,omitempty"`
	// Password is the password for basic authentication to the kubernetes cluster.
	// +optional
	Password string `json:"password,omitempty"`
	// AuthProvider specifies a custom authentication plugin for the kubernetes cluster.
	// +optional
	AuthProvider UserAuthProvider `json:"auth-provider,omitempty" yaml:"auth-provider,omitempty"`
	// Exec specifies a custom exec-based authentication plugin for the kubernetes cluster.
	// +optional
	//Exec *ExecConfig `json:"exec,omitempty"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	// +optional
	//Extensions map[string]runtime.Object `json:"extensions,omitempty"`
}

type UserAuthProvider struct {
	Name string `json:"name,omitempty"`
}

// Context is a tuple of references to a cluster (how do I communicate with a kubernetes cluster), a user (how do I identify myself), and a namespace (what subset of resources do I want to work with)
type Context struct {
	// Cluster is the name of the cluster for this context
	Cluster string `json:"cluster"`
	// AuthInfo is the name of the authInfo for this context
	User string `json:"user"`
	// Namespace is the default namespace to use on unspecified requests
	// +optional
	Namespace string `json:"namespace,omitempty"`
}
