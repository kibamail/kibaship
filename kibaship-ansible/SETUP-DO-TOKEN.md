# DigitalOcean API Token Setup

To run E2E tests, you need a DigitalOcean API token.

## Step 1: Create DigitalOcean API Token

1. **Log in** to your DigitalOcean account: https://cloud.digitalocean.com/
2. **Navigate** to API section: https://cloud.digitalocean.com/account/api/tokens
3. **Click** "Generate New Token"
4. **Configure** the token:
   - **Name**: `kibaship-e2e-testing`
   - **Expiration**: Choose appropriate duration (90 days recommended for development)
   - **Scopes**: Select "Full Access" (Read + Write)
5. **Copy** the generated token (you won't see it again!)

## Step 2: Configure the E2E Environment

```bash
# Navigate to the project root
cd /path/to/kibaship-ansible

# Copy the configuration template
cp e2e/terraform/terraform.tfvars.example e2e/terraform/terraform.tfvars

# Edit the configuration file
nano e2e/terraform/terraform.tfvars
```

## Step 3: Add Your Token

Edit `e2e/terraform/terraform.tfvars`:

```hcl
# Replace 'your_digitalocean_api_token_here' with your actual token
do_token = "dop_v1_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

# Optional: Customize other settings
droplet_region = "nyc3"        # Choose your preferred region
droplet_size = "s-2vcpu-4gb"   # Sufficient for testing
project_name = "kibaship-e2e"  # Your project identifier
```

## Available Regions

Common DigitalOcean regions:
- `nyc1`, `nyc3` - New York
- `lon1` - London  
- `fra1` - Frankfurt
- `sgp1` - Singapore
- `tor1` - Toronto
- `sfo3` - San Francisco
- `ams3` - Amsterdam
- `blr1` - Bangalore

## Available Sizes

Recommended droplet sizes for testing:
- `s-1vcpu-2gb` - Basic ($0.018/hour) - Minimal for light testing
- `s-2vcpu-4gb` - Recommended ($0.036/hour) - Good for full testing
- `s-4vcpu-8gb` - Extended ($0.071/hour) - For performance testing

## Test Your Setup

```bash
# Run a quick validation
./e2e/run-e2e-test.sh provision

# If successful, clean up
./e2e/run-e2e-test.sh destroy
```

## Security Best Practices

1. **Token Security**:
   - Never commit `terraform.tfvars` to git (it's already in .gitignore)
   - Use separate tokens for different environments
   - Regularly rotate your API tokens

2. **Cost Management**:
   - Always destroy test infrastructure when done
   - Monitor your DigitalOcean billing dashboard
   - Set up billing alerts

3. **Resource Cleanup**:
   - The script automatically destroys resources after tests
   - Use `--keep` flag only for debugging
   - Manually verify cleanup in the DO dashboard if needed

## Troubleshooting

### Invalid Token Error
```
Error: GET https://api.digitalocean.com/v2/account: 401 (request "...")
```
- Double-check your token is correct
- Verify the token has "Full Access" permissions
- Check if the token has expired

### Rate Limiting
```
Error: GET https://api.digitalocean.com/v2/droplets: 429 Too Many Requests
```
- Wait a few minutes and retry
- DigitalOcean has API rate limits

### Insufficient Resources
```
Error: POST https://api.digitalocean.com/v2/droplets: 422 (request "...")
```
- Check your account has sufficient credit/balance
- Verify the selected region supports your chosen droplet size
- Try a different region or smaller droplet size

## Alternative Setup Methods

### Environment Variable (Advanced)

Instead of `terraform.tfvars`, you can use an environment variable:

```bash
export DO_TOKEN="your_token_here"
export TF_VAR_do_token="$DO_TOKEN"

# Then run tests without terraform.tfvars file
./e2e/run-e2e-test.sh up
```

### CI/CD Integration

For GitHub Actions or other CI systems:

```yaml
# In your CI/CD configuration
env:
  DO_TOKEN: ${{ secrets.DIGITALOCEAN_TOKEN }}
  
# In the script
run: |
  echo "do_token = \"$DO_TOKEN\"" > e2e/terraform/terraform.tfvars
  ./e2e/run-e2e-test.sh up
```

## Next Steps

Once your token is configured:

1. **Test basic provisioning**: `./e2e/run-e2e-test.sh provision`
2. **Run full E2E test**: `./e2e/run-e2e-test.sh up`
3. **Develop with confidence**: All Ansible roles tested on real infrastructure!

Remember to run `./e2e/run-e2e-test.sh destroy` to clean up any leftover resources.