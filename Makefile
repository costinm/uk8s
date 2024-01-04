include tools/common.mk

deps-gen:
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.13
	go install k8s.io/code-generator/cmd/client-gen@v0.28.3
	go install k8s.io/code-generator/cmd/register-gen@v0.28.3
	go install k8s.io/code-generator/cmd/lister-gen@v0.28.3
	go install k8s.io/code-generator/cmd/informer-gen@v0.28.3
	go install k8s.io/code-generator/cmd/openapi-gen@v0.28.3

gen: gen-controller gen-client gen-register

# This is an easy one: relative paths - must be run in a proper go mod.
# Will generate deepcopy (object) and CRDs
# -www option shows help for the tags
gen-controller:
	(cd echo && controller-gen \
	   object \
	   output:object:dir=./apis/v1 \
	   rbac:roleName=echo \
	   output:rbac:dir=../manifests/charts/echo \
	   crd \
	   output:crd:dir=../manifests/charts/echo/crds \
       paths=./apis/... )

PKG_BASE=github.com/costinm/uk8s

# input-dirs can be relative path, inside the modules. Also works as package name, with /...
# input-base is the module package - should match go.mod
# input is the module-relative package for the API
#
# output package can be anything - should be separate dir
# clientset=name seems to be usually named versioned.
gen-client:
	(cd echo && client-gen --input-dirs ./apis/v1/... \
         --input-base ${PKG_BASE}/echo  \
	     --go-header-file ${BASE}/tools/boilerplate/boilerplate.generatego.txt \
         --output-package  ${PKG_BASE}/client/echo \
	     --fake-clientset=false \
 	    --clientset-name versioned \
	     --input apis/v1 )

# Lister and informers - depend on client. Lister depends on informer indirectly, via store interface
gen-cachedclient:
	(cd echo && lister-gen \
          --input-dirs "${PKG_BASE}/echo/apis/v1" \
      	  --go-header-file ${BASE}/tools/boilerplate/boilerplate.generatego.txt \
          --output-package "${PKG_BASE}/cachedclient/echo/lister" )
	(cd echo && informer-gen \
          --input-dirs "${PKG_BASE}/echo/apis/v1" \
		  --versioned-clientset-package "${PKG_BASE}/client/echo/versioned" \
  		  --listers-package "${PKG_BASE}/cachedclient/echo/lister" \
      	  --go-header-file ${BASE}/tools/boilerplate/boilerplate.generatego.txt \
          --output-package "${PKG_BASE}/cachedclient/echo/informer" )

# zz_generated.register.go in the package dir
gen-register:
	  (cd echo && register-gen  \
          --input-dirs "${PKG_BASE}/echo/apis/v1" \
	     --go-header-file ${BASE}/tools/boilerplate/boilerplate.generatego.txt \
            --output-package "${PKG_BASE}/echo/apis/v1" \
            )

gen-clean:
	rm -rf client/echo
	rm manifests/charts/echo/crds/*
	rm -rf cachedclient/echo
	rm echo/apis/v1/zz_generated*
