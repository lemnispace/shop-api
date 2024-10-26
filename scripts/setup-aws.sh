#!/bin/bash

# Retrieve variables from environment
PROFILE=${AWS_PROFILE}
SSO_START_URL=${AWS_SSO_START_URL}
SSO_ACCOUNT_ID=${AWS_SSO_ACCOUNT_ID}
SSO_ROLE_NAME=${AWS_SSO_ROLE_NAME}
AWS_REGION=${AWS_REGION:-us-east-1}
AWS_OUTPUT=${AWS_OUTPUT:-json}

# Create the AWS CLI config directory if it doesn't exist
mkdir -p ~/.aws

# Write the AWS CLI config file
cat > ~/.aws/config <<EOL
[profile ${PROFILE}]
sso_session = ${PROFILE}
sso_account_id = ${SSO_ACCOUNT_ID}
sso_role_name = ${SSO_ROLE_NAME}
output = ${AWS_OUTPUT}
[sso-session ${PROFILE}]
sso_start_url = ${SSO_START_URL}
sso_region = ${AWS_REGION}
sso_registration_scopes = sso:account:access
EOL

echo "AWS CLI configured with SSO profile '${PROFILE}'."

# Attempt to perform AWS SSO login
echo "Attempting AWS SSO login..."
aws sso login --profile ${PROFILE}

if [ $? -eq 0 ]; then
  echo "AWS SSO login successful."
else
  echo "AWS SSO login failed or requires interactive login."
  echo "Please run 'aws sso login --profile ${PROFILE}' in the terminal to authenticate."
fi

echo "Environment setup complete. You can now use AWS CLI and Terraform."
