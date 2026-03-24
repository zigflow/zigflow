# Zigflow

[![Version](https://img.shields.io/github/v/release/zigflow/zigflow?label=Version&color=007ec6)](https://github.com/zigflow/zigflow/tree/main/charts/zigflow)
![Type: Application](https://img.shields.io/badge/Type-Application-informational)

Define durable workflows in YAML, powered by Temporal

**Homepage:** <https://zigflow.dev>

## TL;DR

Be sure to set `${ZIGFLOW_VERSION}` with [your desired version](https://github.com/zigflow/zigflow/pkgs/container/charts%2Fzigflow)

```sh
helm install myrelease oci://ghcr.io/zigflow/charts/zigflow@${ZIGFLOW_VERSION}
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
			<td>config</td>
			<td>object</td>
			<td><pre lang="json">
{
  "log-level": "info"
}
</pre>
</td>
			<td>Accepts any of the command line arguments</td>
		</tr>
		<tr>
			<td>envvars</td>
			<td>string</td>
			<td><pre lang="json">
null
</pre>
</td>
			<td>Additional environment variables</td>
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
			<td>livenessProbe.httpGet.path</td>
			<td>string</td>
			<td><pre lang="json">
"/health"
</pre>
</td>
			<td>Path to demonstrate app liveness</td>
		</tr>
		<tr>
			<td>livenessProbe.httpGet.port</td>
			<td>string</td>
			<td><pre lang="json">
"health"
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
"/health"
</pre>
</td>
			<td>Path to demonstrate app readiness</td>
		</tr>
		<tr>
			<td>readinessProbe.httpGet.port</td>
			<td>string</td>
			<td><pre lang="json">
"health"
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
			<td>service.health.port</td>
			<td>int</td>
			<td><pre lang="json">
3000
</pre>
</td>
			<td>Health service port</td>
		</tr>
		<tr>
			<td>service.metrics.port</td>
			<td>int</td>
			<td><pre lang="json">
9090
</pre>
</td>
			<td>Metrics service port</td>
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
		<tr>
			<td>workflow.enabled</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Don't add a workflow to the deployment. Useful if you have built an image with the workflow embedded</td>
		</tr>
		<tr>
			<td>workflow.file</td>
			<td>string</td>
			<td><pre lang="json">
"/data/workflow.yaml"
</pre>
</td>
			<td>Location the workflow volumes is mapped</td>
		</tr>
		<tr>
			<td>workflow.inline</td>
			<td>object</td>
			<td><pre lang="json">
{
  "do": [
    {
      "set": {
        "output": {
          "as": {
            "data": "${ . }"
          }
        },
        "set": {
          "message": "Hello from Ziggy"
        }
      }
    }
  ],
  "document": {
    "dsl": "1.0.0",
    "name": "simple-workflow",
    "namespace": "zigflow",
    "version": "1.0.0"
  }
}
</pre>
</td>
			<td>Workflow YAML</td>
		</tr>
		<tr>
			<td>workflow.secret</td>
			<td>string</td>
			<td><pre lang="json">
"workflow"
</pre>
</td>
			<td>Name of the secret containing <code>workflow.yaml</code></td>
		</tr>
		<tr>
			<td>workflow.useInline</td>
			<td>bool</td>
			<td><pre lang="json">
true
</pre>
</td>
			<td>Use the inline workflow. If false, you must declare a secret with the workflow in <code>workflow.yaml</code></td>
		</tr>
	</tbody>
</table>

