# shop-api

API for the e-commerce aspect of lemnispace

## Development

### AWS SSO Configuration Script

#### Purpose

The `setup-aws.sh` script automates the setup of AWS CLI configuration for AWS Single Sign-On (SSO) authentication.
It creates the necessary AWS configuration files and initiates the SSO login process.

#### Prerequisites

- AWS CLI v2 installed
- The following environment variables must be set:
  - `AWS_PROFILE`: The name of the AWS profile to create
  - `AWS_SSO_START_URL`: Your organization's SSO start URL
  - `AWS_SSO_ACCOUNT_ID`: Your AWS account ID
  - `AWS_SSO_ROLE_NAME`: The IAM role name to assume
  - `AWS_REGION`: (Optional) AWS region (defaults to us-east-1)
  - `AWS_OUTPUT`: (Optional) Output format (defaults to json)

#### Usage

1. Set the required environment variables
2. Ensure the script is executable:
   ```chmod +x setup-aws.sh```
3. Run the script:
   ```./setup-aws.sh```

#### What it does

1. Creates `~/.aws` directory if it doesn't exist
2. Generates AWS CLI configuration file with SSO settings
3. Attempts automatic SSO login
4. Provides feedback on the login status

#### When to use

- Initial development environment setup
- When setting up new AWS SSO access
- After refreshing AWS SSO credentials
- When configuring CI/CD pipelines that need AWS authentication

#### Notes

- If automatic login fails, you'll need to run the SSO login command manually
- The script is non-destructive and can be run multiple times
- Existing AWS configurations will be overwritten for the specified profile
