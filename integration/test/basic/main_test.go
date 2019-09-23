// +build k8srequired

package basic

import (
	"context"
	"fmt"
	"os"
	"testing"

	e2esetup "github.com/giantswarm/e2esetup/chart"
	"github.com/giantswarm/e2esetup/chart/env"
	"github.com/giantswarm/e2esetup/k8s"
	"github.com/giantswarm/e2etests/basicapp"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/kubernetes-metrics-server/integration/templates"
)

const (
	envVarTarballURL  = "TARBALL_URL"
	metricsServerName = "metrics-server"
	chartName         = "kubernetes-metrics-server"
)

var (
	tarballURL string
	helmClient *helmclient.Client
	k8sSetup   *k8s.Setup
	l          micrologger.Logger
	ba         *basicapp.BasicApp
)

func init() {
	var err error

	{
		tarballURL = os.Getenv(envVarTarballURL)
		if tarballURL == "" {
			panic(fmt.Sprintf("env var '%s' must not be empty", envVarTarballURL))
		}
	}

	{
		c := micrologger.Config{}
		l, err = micrologger.New(c)
		if err != nil {
			panic(err.Error())
		}
	}

	var k8sClients *k8sclient.Clients
	{
		c := k8sclient.ClientsConfig{
			Logger: l,

			KubeConfigPath: env.KubeConfigPath(),
		}
		k8sClients, err = k8sclient.NewClients(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := k8s.SetupConfig{
			Logger: l,

			Clients: k8sClients,
		}
		k8sSetup, err = k8s.NewSetup(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := helmclient.Config{
			Logger:          l,
			K8sClient:       k8sClients.K8sClient(),
			RestConfig:      k8sClients.RestConfig(),
			TillerNamespace: "giantswarm",
		}
		helmClient, err = helmclient.New(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := basicapp.Config{
			Clients:    k8sClients,
			HelmClient: helmClient,
			Logger:     l,

			App: basicapp.Chart{
				ChartValues: templates.MetricsServerValues,
				Name:        chartName,
				Namespace:   metav1.NamespaceSystem,
				URL:         tarballURL,
			},
			ChartResources: basicapp.ChartResources{
				Deployments: []basicapp.Deployment{
					{
						Name:      metricsServerName,
						Namespace: metav1.NamespaceSystem,
						DeploymentLabels: map[string]string{
							"giantswarm.io/service-type": "managed",
							"app":                        metricsServerName,
						},
						MatchLabels: map[string]string{
							"app": metricsServerName,
						},
						PodLabels: map[string]string{
							"giantswarm.io/service-type": "managed",
							"app":                        metricsServerName,
						},
					},
				},
			},
		}
		ba, err = basicapp.New(c)
		if err != nil {
			panic(err.Error())
		}
	}
}

// TestMain allows us to have common setup and teardown steps that are run
// once for all the tests https://golang.org/pkg/testing/#hdr-Main.
func TestMain(m *testing.M) {
	ctx := context.Background()

	{
		c := e2esetup.Config{
			HelmClient: helmClient,
			Setup:      k8sSetup,
		}

		v, err := e2esetup.Setup(ctx, m, c)
		if err != nil {
			l.LogCtx(ctx, "level", "error", "message", "e2e test failed", "stack", fmt.Sprintf("%#v\n", err))
		}

		os.Exit(v)
	}
}
