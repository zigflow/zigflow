# Zigflow

{{ template "chart.deprecationWarning" . }}

[![Version](https://img.shields.io/github/v/release/zigflow/zigflow?label=Version&color=007ec6)](https://github.com/zigflow/zigflow/tree/main/charts/zigflow)
![Type: Application](https://img.shields.io/badge/Type-Application-informational)

{{ template "chart.description" . }}

{{ template "chart.homepageLine" . }}

## TL;DR

Be sure to set `${ZIGFLOW_VERSION}` with [your desired version](https://github.com/zigflow/zigflow/pkgs/container/charts%2Fzigflow)

```sh
helm install myrelease oci://ghcr.io/zigflow/charts/zigflow@${ZIGFLOW_VERSION}
```

{{ template "chart.maintainersSection" . }}

{{ template "chart.sourcesSection" . }}

{{ template "chart.requirementsSection" . }}

{{ template "chart.valuesSectionHtml" . }}

{{ template "helm-docs.versionFooter" . }}
