# Security Policy

Zigflow is an open-source project licensed under the Apache License 2.0 and is
provided on an "AS IS" basis, without warranties or conditions of any kind.

Security issues are taken seriously and will be addressed on a best-effort basis.

---

## Supported Versions

Security updates are provided for:

- The `main` branch
- The most recent stable release

Older releases may not receive security updates. Users are strongly encouraged
to upgrade to the latest version.

There are no guaranteed response times or service level agreements.

---

## Reporting a Vulnerability

If you discover a security vulnerability, please report it privately.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, use one of the following:

- GitHub Security Advisories via the repository's Security tab
- Email: <hello@zigflow.dev>

Please include:

- A clear description of the vulnerability
- The affected version(s) or commit(s)
- Steps to reproduce
- A proof of concept, exploit details or logs where possible
- Any suggested mitigation

You will receive an acknowledgement within 5 working days.
As Zigflow is maintained in spare time, response times may occasionally be longer.

If you do not receive a response within that time, please follow up.

---

## What to Expect

After a report is received:

1. The issue will be reviewed and severity assessed.
2. Additional information may be requested.
3. If confirmed, a fix will be developed and released.
4. The vulnerability may be disclosed publicly once a fix is available.

Fix timelines depend on severity, complexity and maintainer availability.
There are no guaranteed remediation timelines.

---

## CVEs

Where appropriate, a CVE identifier may be requested and published for
confirmed vulnerabilities.

If a CVE is assigned:

- The CVE ID will be referenced in the security advisory
- Release notes will document affected versions
- Remediation guidance will be provided

---

## Scope

This policy applies to:

- The Zigflow source code
- Official releases and distributed artefacts
- Build and packaging configuration

Out of scope:

- Vulnerabilities in third-party dependencies unless directly introduced by Zigflow
- Theoretical issues without a reproducible attack path
- Denial of service under unrealistic load conditions
- Social engineering or phishing attempts

---

## Responsible Disclosure

Please:

- Avoid accessing data that does not belong to you
- Avoid modifying or deleting data
- Avoid actions that could degrade availability for other users

If you act in good faith and follow responsible disclosure practices, no legal
action will be taken against you for reporting vulnerabilities.

---

## Security Best Practices for Users

Users of Zigflow should:

- Run the latest supported version
- Restrict network exposure where applicable
- Follow the principle of least privilege
- Monitor logs and system behaviour for anomalies
