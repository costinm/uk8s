package uk8s

//// Get a mesh env setting. May be replaced by an env variable.
//// Used to configure PROJECT_NUMBER and other internal settings.
//func (ms *K8STokenSource) GetEnv(k, def string) string {
//	v := os.Getenv(k)
//	if v != "" {
//		return v
//	}
//	v = ms.Env[k]
//	if v != "" {
//		return v
//	}
//
//	return def
//}
//

// Old: LoadMeshEnv will load the 'mesh-env' config map in istio-system, and save the
// settings. It had broad RBAC permissions. and included GCP settings.
//
// This is required for:
//   - getting the PROJECT_NUMBER, for GCP access/id token - used for stackdriver or MCP (not
//     required for federated tokens and certs)
//   - getting the 'mesh connector' address - used to connect to Istiod from 'outside'
//   - getting cluster info - if the K8S cluster is not initiated using a kubeconfig (we can
//     extract it from names).
//
// Replacement: control plane or user should set them, using automation on the initial config.
// or distribute along with system roots and bootstrap info, or use extended MDS.
//
// TODO: get cluster info by getting an initial token.
//func (def *K8STokenSource) LoadMeshEnv(ctx context.Context) error {
//	// Found a K8S cluster, try to locate configs in K8S by getting a config map containing Istio properties
//	cm, err := def.GetConfigMap(ctx, "istio-system", "mesh-env")
//	if def.Env == nil {
//		def.Env = map[string]string{}
//	}
//	if err == nil {
//		// Tokens using istio-ca audience for Istio
//		// If certificates exist, namespace/sa are initialized from the cert SAN
//		for k, v := range cm {
//			def.Env[k] = v
//		}
//	} else {
//		log.Println("Invalid mesh-env config map", err)
//		return err
//	}
//	if def.ProjectID == "" {
//		def.ProjectID = def.GetEnv("PROJECT_ID", "")
//	}
//	if def.Location == "" {
//		def.Location = def.GetEnv("CLUSTER_LOCATION", "")
//	}
//	if def.ClusterName == "" {
//		def.ClusterName = def.GetEnv("CLUSTER_NAME", "")
//	}
//	return nil
//}

// Equivalent config using shell:
//
//```shell
//CMD="gcloud container clusters describe ${CLUSTER} --zone=${ZONE} --project=${PROJECT}"
//
//K8SURL=$($CMD --format='value(endpoint)')
//K8SCA=$($CMD --format='value(masterAuth.clusterCaCertificate)' )
//```
//
//```yaml
//apiVersion: v1
//kind: Config
//current-context: my-cluster
//contexts: [{name: my-cluster, context: {cluster: cluster-1, user: user-1}}]
//users: [{name: user-1, user: {auth-provider: {name: gcp}}}]
//clusters:
//- name: cluster-1
//  cluster:
//    server: "https://${K8SURL}"
//    certificate-authority-data: "${K8SCA}"
//
//```
