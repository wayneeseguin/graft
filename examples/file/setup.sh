#!/bin/bash
# Setup script to create sample files for file operator examples

# Create directories
mkdir -p scripts configs/dev configs/prod certificates

# Create script files
cat > scripts/startup.sh << 'EOF'
#!/bin/bash
echo "Starting application..."
export APP_NAME="${APP_NAME:-myapp}"
export PORT="${PORT:-8080}"
exec java -jar app.jar
EOF

cat > scripts/healthcheck.sh << 'EOF'
#!/bin/bash
curl -f http://localhost:${PORT:-8080}/health || exit 1
EOF

# Create config files
cat > configs/dev/database.conf << 'EOF'
host=localhost
port=5432
name=myapp_dev
pool_size=5
EOF

cat > configs/prod/database.conf << 'EOF'
host=prod-db.example.com
port=5432
name=myapp_prod
pool_size=20
ssl_mode=require
EOF

# Create a certificate file
cat > certificates/server.crt << 'EOF'
-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKl3qOXr/VIFMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjBF
... (truncated for example) ...
-----END CERTIFICATE-----
EOF

# Create a JSON config file
cat > configs/features.json << 'EOF'
{
  "features": {
    "new_ui": true,
    "beta_api": false,
    "debug_mode": "${DEBUG:-false}"
  }
}
EOF

echo "Sample files created successfully!"
echo "Run 'spruce merge basic.yml' to see the examples in action."