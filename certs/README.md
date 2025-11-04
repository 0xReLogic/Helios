# TLS Certificates

## WARNING

**The certificates in this directory are for TESTING PURPOSES ONLY.**

These certificates are publicly available in the repository and should NEVER be used in production environments.

## Generating Your Own Certificates

### For Testing (Self-Signed)

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=localhost"
```

### For Production (Let's Encrypt)

```bash
# Install certbot
sudo apt-get install certbot

# Generate certificate for your domain
sudo certbot certonly --standalone -d yourdomain.com

# Copy certificates
sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem cert.pem
sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem key.pem
```

### For Production (Custom CA)

If you have certificates from a Certificate Authority:

1. Place your certificate in `cert.pem`
2. Place your private key in `key.pem`
3. Ensure proper file permissions:
   ```bash
   chmod 600 key.pem
   chmod 644 cert.pem
   ```

## Security Best Practices

- Never commit real production certificates to version control
- Use strong encryption (RSA 4096 or ECC)
- Rotate certificates before expiration
- Keep private keys secure with proper file permissions
- Consider using a secrets management system for production deployments
