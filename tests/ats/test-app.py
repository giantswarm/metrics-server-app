import logging

import pykube
from pykube.objects import APIObject
from pykube.objects import NamespacedAPIObject

import pytest
from pytest_helm_charts.clusters import Cluster
from pytest_helm_charts.giantswarm_app_platform.app import AppCR
from pytest_helm_charts.k8s.deployment import wait_for_deployments_to_run


logger = logging.getLogger(__name__)

timeout: int = 360

app_catalog_url = "https://giantswarm.github.io/giantswarm-catalog/"

app_namespace = "kube-system"
app_name = "metrics-server-app"

class NodeMetrics(APIObject):
    """
    Kubernetes API object for Node metrics.

    See https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/resource-metrics-api.md
    """

    version = "metrics.k8s.io/v1beta1"
    endpoint = "nodes"
    kind = "NodeMetrics"

class PodMetrics(NamespacedAPIObject):
    """
    Kubernetes API object for Pod metrics.

    See https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/resource-metrics-api.md
    """

    version = "metrics.k8s.io/v1beta1"
    endpoint = "pods"
    kind = "PodMetrics"

@pytest.mark.smoke
@pytest.mark.upgrade
def test_api_working(kube_cluster: Cluster) -> None:
    """
    Test if the kubernetes api works
    """
    assert kube_cluster.kube_client is not None
    assert len(pykube.Node.objects(kube_cluster.kube_client)) >= 1

    kube_cluster.kubectl("get ns")

@pytest.mark.smoke
@pytest.mark.upgrade
def test_app_deployed(kube_cluster: Cluster):
    app = (
        AppCR.objects(kube_cluster.kube_client)
        .filter(namespace=app_namespace)
        .get_by_name(app_name)
    )
    app_version = app.obj["status"]["version"]
    wait_for_deployments_to_run(
        kube_cluster.kube_client,
        # this is the name of the deployments
        ["metrics-server"],
        app_namespace,
        timeout,
    )
    logger.info(f"metrics-server installed in appVersion {app_version}")

    logger.info("waiting for the metrics become available")

    retries = 5
    while retries < timeout:
        try:
          for node_metrics in NodeMetrics.objects(kube_cluster.client):
            logger.info(f"successfully retrieved node metrics from metrics server: {node_metrics.name}")
          for pod_metrics in PodMetrics.objects(kube_cluster.client, namespace=pykube.all):
            logger.info(f"successfully retrieved pod metrics from metrics server: {pod_metrics.namespace}/{pod_metrics.name}")
        except Exception:
          logger.exception(f"failed to get node usage metrics for the {retries} time")
          retries += 1
