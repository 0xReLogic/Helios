# Admin API Security

The Admin API provides powerful runtime control over Helios, including the ability to add/remove backends and change load balancing strategies. To secure this API, Helios supports multiple layers of protection.

## Authentication

### JWT Token Authentication

All Admin API endpoints (except `/v1/health`) require a JWT token passed via the `Authorization: Bearer <token>` header.

**Configuration:**
```yaml
admin_api:
  enabled: true
  port: 9091
  auth_token: "your-secret-token-here"
```

**Usage:**
```bash
curl -H "Authorization: Bearer your-secret-token-here" \
  http://localhost:9091/v1/backends
```

⚠️ **Security Best Practices:**
- Never use the default token `change-me` in production
- Use a strong, randomly generated token (minimum 32 characters)
- Rotate tokens regularly
- Store tokens securely (environment variables, secrets management)

## IP-Based Access Control

### Overview

IP filtering provides an additional layer of security by restricting Admin API access based on client IP addresses. This is particularly useful for:

- Restricting access to internal networks only
- Blocking known malicious IPs
- Implementing defense-in-depth security

### Configuration

IP filtering supports two types of lists:

1. **Allow List (Whitelist)**: Only specified IPs can access the API
2. **Deny List (Blacklist)**: Specified IPs are blocked from accessing the API

**Example Configuration:**
```yaml
admin_api:
  enabled: true
  port: 9091
  auth_token: "your-secret-token"
  ip_allow_list:
    - "127.0.0.1"           # Allow localhost
    - "192.168.1.0/24"      # Allow local network
    - "10.0.0.0/8"          # Allow private network
  ip_deny_list:
    - "203.0.113.0/24"      # Block specific subnet
    - "198.51.100.50"       # Block specific IP
```

### IP Filter Behavior

1. **Deny List Priority**: Deny list is checked first. If an IP matches the deny list, it's blocked immediately.

2. **Allow List Logic**:
   - If `ip_allow_list` is empty: All IPs are allowed (except those in deny list)
   - If `ip_allow_list` is specified: Only IPs in the list are allowed

3. **CIDR Notation Support**:
   - Single IP: `192.168.1.100`
   - IPv4 CIDR: `192.168.1.0/24`
   - IPv6 CIDR: `2001:db8::/32`
   - Large subnets: `10.0.0.0/8`

4. **Blocked Requests**: Return HTTP 403 (Forbidden) with message "Forbidden: IP address not allowed"

### Common Use Cases

#### 1. Localhost Only (Development)

```yaml
admin_api:
  ip_allow_list:
    - "127.0.0.1"
    - "::1"  # IPv6 localhost
```

#### 2. Internal Network Only (Production)

```yaml
admin_api:
  ip_allow_list:
    - "10.0.0.0/8"        # Private network
    - "172.16.0.0/12"     # Private network
    - "192.168.0.0/16"    # Private network
```

#### 3. Specific Admin IPs

```yaml
admin_api:
  ip_allow_list:
    - "203.0.113.10"      # Admin workstation 1
    - "203.0.113.11"      # Admin workstation 2
    - "198.51.100.0/24"   # Admin subnet
```

#### 4. Block Malicious IPs

```yaml
admin_api:
  ip_deny_list:
    - "203.0.113.0/24"    # Known attack source
    - "198.51.100.50"     # Specific malicious IP
```

#### 5. Allow All Except Specific IPs

```yaml
admin_api:
  # Empty allow list = allow all
  ip_deny_list:
    - "203.0.113.0/24"    # Block this subnet
```

### Testing IP Filtering

You can test IP filtering using curl with different source IPs:

```bash
# Test from allowed IP (should succeed)
curl -H "Authorization: Bearer token" \
  --interface 192.168.1.100 \
  http://localhost:9091/v1/backends

# Test from blocked IP (should return 403)
curl -H "Authorization: Bearer token" \
  --interface 10.0.0.1 \
  http://localhost:9091/v1/backends
```

### Logging

When an IP is blocked, Helios logs a warning message:

```
WRN IP blocked by filter client_ip=10.0.0.1 path=/v1/backends
```

This helps with:
- Security monitoring
- Debugging access issues
- Identifying potential attacks

### Performance Considerations

- IP filtering uses efficient CIDR matching with `net.IPNet`
- Minimal overhead (microseconds per request)
- No impact on main proxy traffic (only affects Admin API)
- Scales well with large allow/deny lists

### Security Recommendations

1. **Use Both Authentication and IP Filtering**: Combine JWT tokens with IP filtering for defense-in-depth

2. **Principle of Least Privilege**: Only allow IPs that need access

3. **Regular Audits**: Review and update IP lists regularly

4. **Monitor Logs**: Watch for blocked access attempts

5. **Use HTTPS**: Always use TLS for Admin API in production

6. **Network Segmentation**: Place Admin API on a separate network if possible

### Troubleshooting

#### Issue: Can't access Admin API

**Check:**
1. Is your IP in the allow list?
2. Is your IP in the deny list?
3. Are you using the correct token?
4. Check logs for "IP blocked by filter" messages

**Solution:**
```bash
# Check your current IP
curl ifconfig.me

# Add it to the allow list
admin_api:
  ip_allow_list:
    - "YOUR_IP_HERE"
```

#### Issue: Deny list not working

**Remember:** Deny list only works if the IP would otherwise be allowed. If you have an allow list, the IP must be in the allow list first before the deny list is checked.

### Example: Complete Secure Configuration

```yaml
admin_api:
  enabled: true
  port: 9091
  auth_token: "use-a-strong-random-token-here-min-32-chars"
  
  # Allow only internal network
  ip_allow_list:
    - "127.0.0.1"           # Localhost
    - "192.168.1.0/24"      # Internal network
    - "10.0.0.0/8"          # VPN network
  
  # Block known bad actors
  ip_deny_list:
    - "203.0.113.0/24"      # Known attack subnet
    - "198.51.100.50"       # Specific malicious IP
```

### References

- [Go net package documentation](https://pkg.go.dev/net)
- [CIDR notation explained](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing)
- [RFC 1918 - Private Address Space](https://tools.ietf.org/html/rfc1918)
