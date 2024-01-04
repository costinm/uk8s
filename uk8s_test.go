package uk8s

import (
	"context"
	"log"
	"testing"

	"github.com/costinm/meshauth"
)

// Test for the uk8s minimal bootstraping and auth

// Test using uK8S as token source, exchange with GCP tokens.
func TestGCP2(t *testing.T) {
	ctx := context.Background()

	ma := meshauth.NewMeshAuth(&meshauth.MeshAuthCfg{})

	// Load all clusters from kube config
	def, _ := KubeFromEnv(ma)
	if def == nil {
		t.Skip("Can't find a kube config file")
	}

	def.Namespace = "default"

	gsa := "k8s-" + def.Namespace + "@" + ma.ProjectID + ".iam.gserviceaccount.com"

	t.Run("K8S istio-ca tokens", func(t *testing.T) {
		// Tokens using istio-ca audience for Istio - this is what Citadel and Istiod expect
		catokenS := &meshauth.AudienceOverrideTokenSource{TokenSource: def, Audience: "istio-ca"}

		istiocaTok, err := catokenS.GetToken(ctx, "Foo")
		if err != nil {
			t.Fatal(err)
		}

		jwt := meshauth.DecodeJWT(istiocaTok)

		if jwt.Audience() != "istio-ca" {
			t.Fatal(t)
		}
	})

	t.Run("K8S audience tokens", func(t *testing.T) {
		// Without audience overide - K8STokenSource is a TokenSource as well
		tok, err := def.GetToken(ctx, "http://example.com")
		if err != nil {
			t.Error("Getting tokens with audience from k8s", err)
		}

		jwt := meshauth.DecodeJWT(tok)

		if jwt.Audience() != "http://example.com" {
			t.Fatal(t)
		}
		t.Log(jwt.String())
	})

	t.Run("K8S GCP federated tokens", func(t *testing.T) {
		sts1 := meshauth.NewFederatedTokenSource(&meshauth.STSAuthConfig{
			TokenSource:    def,
			AudienceSource: ma.ProjectID + ".svc.id.goog",
			// no GSA set - returns the original federated access token
		})
		tok, err := sts1.GetToken(ctx, "http://example.com")
		if err != nil {
			t.Error(err)
		}
		t.Log("Federated access token", tok[0:10])

	})

	// Use K8S as a JWT token source with AudienceSource, get a fed token and impersonate a GSA.
	gsaSrc := meshauth.NewFederatedTokenSource(&meshauth.STSAuthConfig{
		TokenSource:    def,
		GSA:            gsa,
		AudienceSource: ma.ProjectID + ".svc.id.goog",
	})

	t.Run("K8S GCP tokens", func(t *testing.T) {
		tok, err := gsaSrc.GetToken(ctx, "http://example.com")
		if err != nil {
			t.Error(err)
		}

		tokT := meshauth.DecodeJWT(tok)
		t.Log(tokT)

		tok, err = gsaSrc.GetToken(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Delegated user access token", tok[0:9])
	})

	t.Run("ADC-user", func(t *testing.T) {
		oa := FindDefaultCredentials()
		//log.Println(oa)
		tok, err := oa.Token(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		log.Println(tok[0:10])

		tok, err = oa.Token(ctx, "32555940559.apps.googleusercontent.com")
		if err != nil {
			t.Fatal(err)
		}

		log.Println(meshauth.DecodeJWT(tok))
	})

	t.Run("gke", func(t *testing.T) {
		tok, err := gsaSrc.GetToken(ctx, "")
		if err != nil {
			t.Error(err)
		}

		g := GCPAuth{}
		cd, err := g.GKEClusters(ctx, tok, ma.ProjectID)
		if err != nil {
			t.Fatal(err)
		}
		log.Println(cd)
	})

	t.Run("hub", func(t *testing.T) {
		tok, err := gsaSrc.GetToken(ctx, "")
		if err != nil {
			t.Error(err)
		}

		g := GCPAuth{}
		cd, err := g.HubClusters(ctx, tok, ma.ProjectID)
		if err != nil {
			t.Fatal(err)
		}
		log.Println(cd)
	})

	t.Run("secret", func(t *testing.T) {
		tok, err := gsaSrc.GetToken(ctx, "")
		if err != nil {
			t.Error(err)
		}

		cd, err := GetSecret(ctx, tok, ma.ProjectID, "ca", "1")
		if err != nil {
			t.Fatal(err)
		}
		log.Println(string(cd))
	})

}
