# Security Policy

## Supported Versions

The `ubuntu-insights` client, shared library, and development headers are currently released as a package included in the [Ubuntu archive](https://launchpad.net/ubuntu/+source/ubuntu-insights). Its API is also available as a [Go package](https://pkg.go.dev/github.com/ubuntu/ubuntu-insights/insights).
We provide security updates for these releases of `ubuntu-insights`. Please ensure you are using a supported version to receive updates and patches.

## Reporting a Vulnerability

If you discover a security vulnerability within this repository, we encourage responsible disclosure. Please report any security issues to help us keep `ubuntu-insights` secure for everyone.

The server services of `ubuntu-insights` are packaged into a [Juju Charm](https://github.com/canonical/ubuntu-insights-k8s-operator). For vulnerabilities specific to the Charm, please report them there.

### Private Vulnerability Reporting

The most straightforward way to report a security vulnerability is via [GitHub](https://github.com/ubuntu/ubuntu-insights/security/advisories/new).
For detailed instructions, please review the [Privately reporting a security vulnerability](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability) documentation.
This method enables you to communicate vulnerabilities directly and confidentially with the `ubuntu-insights` maintainers.

The project's admins will be notified of the issue and will work with you to determine whether the issue qualifies as a security issue and, if so, in which component.
We will then handle figuring out a fix, getting a CVE assigned and coordinating the release of the fix to the various Linux distributions.

The [Ubuntu Security disclosure and embargo policy](https://ubuntu.com/security/disclosure-policy) contains more information about what you can expect when you contact us, and what we expect from you.

#### Steps to Report a Vulnerability

1. Go to the [Security Advisories Page](https://github.com/ubuntu/ubuntu-insights/security/advisories) of the `ubuntu-insights` repository.
2. Click "Report a Vulnerability."
3. Provide detailed information about the vulnerability, including steps to reproduce, affected versions, and potential impact.

## Security Resources

- [Canonical's Security Site](https://ubuntu.com/security)
- [Ubuntu Security disclosure and embargo policy](https://ubuntu.com/security/disclosure-policy)
- [Ubuntu Security Notices](https://ubuntu.com/security/notices)

If you have any questions regarding security vulnerabilities, please reach out to the maintainers via the aforementioned channels.
