# Security Policy

## Supported Versions

Currently, only the latest version of gosearch is supported. Security updates will be provided for the current release.

## Reporting Security Vulnerabilities

If you discover a security vulnerability, please **do not open a public issue**.

### How to Report

Send an email to: **abuiliazeed@users.noreply.github.com**

Please include:
- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact assessment
- Any suggested fixes (if available)

### What to Expect

1. **Acknowledgment**: You will receive a response within 48 hours
2. **Investigation**: We will investigate the issue and determine severity
3. **Resolution**: We will work on a fix and coordinate release timing
4. **Disclosure**: Public disclosure will be coordinated after the fix is released

### Security Response Process

1. Verify and triage the reported vulnerability
2. Develop a fix in a private branch
3. Prepare a security advisory
4. Release fixed version
5. Publish security advisory

## Security Best Practices

### For Users

- **Keep updated**: Use the latest version of gosearch
- **Review logs**: Monitor crawl logs for suspicious activity
- **Network isolation**: Run crawls in isolated environments when possible
- **Resource limits**: Configure appropriate worker and depth limits

### For Developers

- **Input validation**: Always validate URLs and user input
- **Error handling**: Never expose sensitive information in error messages
- **Dependency updates**: Regularly update dependencies
- **Code review**: All code changes should be reviewed

## Known Security Considerations

### Web Crawling

- **Politeness**: gosearch respects robots.txt by default
- **Rate limiting**: Configurable delays prevent server overload
- **Resource limits**: Queue limits prevent unbounded crawling

### XSS Protection

- **Input sanitization**: All search result fields (title, URL, snippet) are sanitized using bluemonday to prevent XSS attacks
- **HTML stripping**: The default policy strips all HTML tags, leaving only plain text content
- **Safe defaults**: Sanitization is applied automatically to all API responses

### CORS Configuration

- **Configurable origins**: CORS allowed origins can be configured via `cors-allowed-origins` setting
- **Default wildcard**: By default, CORS is set to `*` (all origins) for development convenience
- **Production recommendation**: For production, set specific origins: `cors-allowed-origins: "https://example.com,https://app.example.com"`
- **Methods and headers**: Allowed methods and headers are also configurable

### Data Storage

- **No credentials**: The application does not store or transmit credentials
- **Local storage**: All data is stored locally by default
- **File permissions**: Users are responsible for setting appropriate file permissions

### Dependencies

We use Go modules with checksum verification. Dependencies are updated regularly.

## Security-Related Configuration

### Environment Variables

```bash
# Disable Redis caching if not needed
GOSEARCH_REDIS_HOST=""

# Limit crawl resources
GOSEARCH_MAX_WORKERS=5
GOSEARCH_MAX_DEPTH=2

# Configure CORS for production (restrict to specific origins)
GOSEARCH_CORS_ALLOWED_ORIGINS="https://example.com,https://app.example.com"
GOSEARCH_CORS_ALLOWED_METHODS="GET, POST"
GOSEARCH_CORS_ALLOWED_HEADERS="Content-Type, Authorization"
```

### Configuration File (.gosearch.yaml)

```yaml
# Secure CORS configuration for production
cors-allowed-origins: "https://example.com,https://app.example.com"
cors-allowed-methods: "GET, POST"
cors-allowed-headers: "Content-Type, Authorization"

# Development mode (wildcard - NOT recommended for production)
# cors-allowed-origins: "*"
```

### File Permissions

```bash
# Restrict data directory access
chmod 700 ./data

# Restrict config file access
chmod 600 .gosearch.yaml
```

## Vulnerability Disclosure

Past security advisories will be published in the [GitHub Security Advisories](https://github.com/abuiliazeed/gosearch/security/advisories) section.

## Receiving Security Updates

To receive security notifications:

1. **Watch the repository** on GitHub for releases
2. **Enable security alerts** in your GitHub notification settings
3. **Monitor the CHANGELOG.md** for security-related updates

## Security Contact

For security-related questions or concerns:
- Email: abuiliazeed@users.noreply.github.com
- GitHub Security: [Report a vulnerability](https://github.com/abuiliazeed/gosearch/security/advisories/new)

Thank you for helping keep gosearch secure!
