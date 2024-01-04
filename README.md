# uK8S 

This is a minimal project to parse a KUBECONFIG file and creating a set of token providers and 
generic 'clusters'.

The 'meshauth' project provides common auth and bootstraps mechanisms used in a mesh - the 'mk8s'
project is using the standard client libraries to provide authentication to K8S and TokenRequest
integration.

This project is intended for minimal clients that don't want a dependency on the much larger K8S
client library, as well as minimal GCP integration without deps on the full GCP client libraries.

It is not a K8S client library - just the bootstrap and auth.
