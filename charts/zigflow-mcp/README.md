# Zigflow MCP

[![Version](https://img.shields.io/github/v/release/zigflow/zigflow?label=Version&color=007ec6)](https://github.com/zigflow/zigflow/tree/main/charts/zigflow-mcp)
![Type: Application](https://img.shields.io/badge/Type-Application-informational)

Zigflow's MCP server

**Homepage:** <https://zigflow.dev>

## TL;DR

Be sure to set `${ZIGFLOW_VERSION}` with [your desired version](https://github.com/zigflow/zigflow/pkgs/container/charts%2Fzigflow-mcp)

```sh
helm install myrelease oci://ghcr.io/zigflow/charts/zigflow-mcp@${ZIGFLOW_VERSION}
```

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Simon Emms | <simon@simonemms.com> | <https://simonemms.com> |

## Source Code

* <https://github.com/zigflow/zigflow>

## Values

<table>
	<thead>
		<th>Key</th>
		<th>Type</th>
		<th>Default</th>
		<th>Description</th>
	</thead>
	<tbody>
		<tr>
			<td>affinity</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Node affinity</td>
		</tr>
		<tr>
			<td>autoscaling.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td>Autoscaling enabled</td>
		</tr>
		<tr>
			<td>autoscaling.maxReplicas</td>
			<td>int</td>
			<td><pre lang="json">
100
</pre>
</td>
			<td>Maximum replicas</td>
		</tr>
		<tr>
			<td>autoscaling.minReplicas</td>
			<td>int</td>
			<td><pre lang="json">
1
</pre>
</td>
			<td>Minimum replicas</td>
		</tr>
		<tr>
			<td>autoscaling.targetCPUUtilizationPercentage</td>
			<td>int</td>
			<td><pre lang="json">
80
</pre>
</td>
			<td>When to trigger a new replica</td>
		</tr>
		<tr>
			<td>deployment.args[0]</td>
			<td>string</td>
			<td><pre lang="json">
"mcp"
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>deployment.args[1]</td>
			<td>string</td>
			<td><pre lang="json">
"--transport=http"
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>deployment.args[2]</td>
			<td>string</td>
			<td><pre lang="json">
"--address=0.0.0.0:{{ .Values.service.port }}"
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>deployment.envvars[0].name</td>
			<td>string</td>
			<td><pre lang="json">
"LOG_LEVEL"
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>deployment.envvars[0].value</td>
			<td>string</td>
			<td><pre lang="json">
"info"
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>fullnameOverride</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>String to fully override names</td>
		</tr>
		<tr>
			<td>httpRoute</td>
			<td>object</td>
			<td><pre lang="json">
{
  "annotations": {},
  "enabled": false,
  "hostnames": [
    "chart-example.local"
  ],
  "parentRefs": [
    {
      "name": "gateway",
      "sectionName": "http"
    }
  ],
  "rules": [
    {
      "matches": [
        {
          "path": {
            "type": "PathPrefix",
            "value": "/"
          }
        }
      ]
    }
  ]
}
</pre>
</td>
			<td>Expose the service via gateway-api HTTPRoute Requires Gateway API resources and suitable controller installed within the cluster (see: https://gateway-api.sigs.k8s.io/guides/)</td>
		</tr>
		<tr>
			<td>image.pullPolicy</td>
			<td>string</td>
			<td><pre lang="json">
"IfNotPresent"
</pre>
</td>
			<td>Image pull policy</td>
		</tr>
		<tr>
			<td>image.repository</td>
			<td>string</td>
			<td><pre lang="json">
"ghcr.io/zigflow/zigflow"
</pre>
</td>
			<td>Image repositiory</td>
		</tr>
		<tr>
			<td>image.tag</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>Image tag - defaults to the chart's <code>Version</code> if not set</td>
		</tr>
		<tr>
			<td>imagePullSecrets</td>
			<td>list</td>
			<td><pre lang="json">
[]
</pre>
</td>
			<td>Docker registry secret names</td>
		</tr>
		<tr>
			<td>ingress.annotations</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>ingress.className</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>ingress.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
false
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>ingress.hosts[0].host</td>
			<td>string</td>
			<td><pre lang="json">
"chart-example.local"
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>ingress.hosts[0].paths[0].path</td>
			<td>string</td>
			<td><pre lang="json">
"/"
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>ingress.hosts[0].paths[0].pathType</td>
			<td>string</td>
			<td><pre lang="json">
"ImplementationSpecific"
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>ingress.tls</td>
			<td>list</td>
			<td><pre lang="json">
[]
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>livenessProbe.httpGet.path</td>
			<td>string</td>
			<td><pre lang="json">
"/healthz"
</pre>
</td>
			<td>Path to demonstrate app liveness</td>
		</tr>
		<tr>
			<td>livenessProbe.httpGet.port</td>
			<td>string</td>
			<td><pre lang="json">
"http"
</pre>
</td>
			<td>Port to demonstrate app liveness</td>
		</tr>
		<tr>
			<td>nameOverride</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>String to partially override name</td>
		</tr>
		<tr>
			<td>nodeSelector</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Node selector</td>
		</tr>
		<tr>
			<td>podAnnotations</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Pod <a href="https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations" target="_blank">annotations</a></td>
		</tr>
		<tr>
			<td>podLabels</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Pod <a href="https://kubernetes.io/docs/concepts/overview/working-with-objects/labels" target="_blank">labels</a></td>
		</tr>
		<tr>
			<td>podSecurityContext</td>
			<td>object</td>
			<td><pre lang="json">
{
  "fsGroup": 1000,
  "runAsNonRoot": true,
  "seccompProfile": {
    "type": "RuntimeDefault"
  }
}
</pre>
</td>
			<td>Pod's <a href="https://kubernetes.io/docs/tasks/configure-pod-container/security-context" target="_blank">security context</a></td>
		</tr>
		<tr>
			<td>readinessProbe.httpGet.path</td>
			<td>string</td>
			<td><pre lang="json">
"/healthz"
</pre>
</td>
			<td>Path to demonstrate app readiness</td>
		</tr>
		<tr>
			<td>readinessProbe.httpGet.port</td>
			<td>string</td>
			<td><pre lang="json">
"http"
</pre>
</td>
			<td>Port to demonstrate app readiness</td>
		</tr>
		<tr>
			<td>replicaCount</td>
			<td>int</td>
			<td><pre lang="json">
1
</pre>
</td>
			<td>Number of replicas</td>
		</tr>
		<tr>
			<td>resources</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Configure resources available</td>
		</tr>
		<tr>
			<td>securityContext</td>
			<td>object</td>
			<td><pre lang="json">
{
  "allowPrivilegeEscalation": false,
  "capabilities": {
    "drop": [
      "ALL"
    ]
  },
  "readOnlyRootFilesystem": true,
  "runAsNonRoot": true,
  "seccompProfile": {
    "type": "RuntimeDefault"
  }
}
</pre>
</td>
			<td>Container's security context</td>
		</tr>
		<tr>
			<td>service.port</td>
			<td>int</td>
			<td><pre lang="json">
8080
</pre>
</td>
			<td></td>
		</tr>
		<tr>
			<td>service.type</td>
			<td>string</td>
			<td><pre lang="json">
"ClusterIP"
</pre>
</td>
			<td>Service type</td>
		</tr>
		<tr>
			<td>serviceAccount.annotations</td>
			<td>object</td>
			<td><pre lang="json">
{}
</pre>
</td>
			<td>Annotations to add to the service account</td>
		</tr>
		<tr>
			<td>serviceAccount.automount</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Automatically mount a ServiceAccount's API credentials?</td>
		</tr>
		<tr>
			<td>serviceAccount.create</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Specifies whether a service account should be created</td>
		</tr>
		<tr>
			<td>serviceAccount.name</td>
			<td>string</td>
			<td><pre lang="json">
""
</pre>
</td>
			<td>The name of the service account to use. If not set and create is true, a name is generated using the fullname template</td>
		</tr>
		<tr>
			<td>tolerations</td>
			<td>list</td>
			<td><pre lang="json">
[]
</pre>
</td>
			<td>Node toleration</td>
		</tr>
		<tr>
			<td>volumeMounts</td>
			<td>list</td>
			<td><pre lang="json">
[]
</pre>
</td>
			<td>Additional volumeMounts on the output Deployment definition.</td>
		</tr>
		<tr>
			<td>volumes</td>
			<td>list</td>
			<td><pre lang="json">
[]
</pre>
</td>
			<td>Additional volumes on the output Deployment definition.</td>
		</tr>
	</tbody>
</table>

