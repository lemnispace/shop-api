# Shop API Terraform Infrastructure

This directory contains Terraform configurations for managing the shop-api infrastructure across multiple environments.

## Directory Structure

```
terraform/
├── README.md                    # This file
├── main.tf                      # Production infrastructure (legacy)
├── variables.tf                 # Production variables (legacy)
├── outputs.tf                   # Production outputs (legacy)
├── modules/                     # Reusable Terraform modules
│   ├── dynamodb/               # DynamoDB table module
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── outputs.tf
│   └── routes/                 # API Gateway routes module
│       └── main.tf
└── environments/               # Environment-specific configurations
    ├── local/                  # LocalStack development environment
    │   ├── main.tf
    │   ├── variables.tf
    │   └── outputs.tf
    └── dev/                    # AWS dev environment
        ├── main.tf
        ├── variables.tf
        └── outputs.tf
```

## Environments

### 1. Local Development (LocalStack)

**Purpose**: Local development and testing using LocalStack to emulate AWS services.

**Location**: `terraform/environments/local/`

**Features**:
- DynamoDB table with all GSIs (including ProductsByStatus)
- S3 bucket for file storage
- No remote state (local state file)
- Provisioned billing mode (required by LocalStack)
- Disabled PITR and encryption for faster local development

**Usage**:

```bash
# Start LocalStack
cd ../..
docker compose up -d localstack

# Initialize Terraform
cd terraform/environments/local
terraform init

# Plan and apply
terraform plan
terraform apply

# Test connectivity
aws dynamodb describe-table \
  --table-name ShopAPI \
  --endpoint-url http://localhost:4566 \
  --region us-east-1
```

**Environment Variables**:
```bash
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1
```

### 2. Development Environment (AWS)

**Purpose**: AWS-based development environment for integration testing.

**Location**: `terraform/environments/dev/`

**Features**:
- DynamoDB table with PAY_PER_REQUEST billing
- Point-in-time recovery enabled
- Server-side encryption enabled
- Remote state in S3
- Integration with shared infrastructure

**Usage**:

```bash
# Build Lambda function first
cd ../../..
make build

# Initialize Terraform
cd terraform/environments/dev
terraform init

# Plan and apply
terraform plan
terraform apply
```

**Prerequisites**:
- AWS credentials configured
- Shared infrastructure deployed (`infra` repository)
- S3 bucket for remote state: `lemnispace-terraform-state`
- DynamoDB table for state locking: `terraform-state-lock`

### 3. Production Environment (Legacy)

**Purpose**: Production infrastructure (to be migrated to environments pattern).

**Location**: `terraform/main.tf` (root level)

**Note**: This will eventually be migrated to `terraform/environments/prod/`.

## DynamoDB Module

The `modules/dynamodb` module provides a reusable DynamoDB table configuration with:

### Primary Keys
- **PK** (String): Partition key
- **SK** (String): Sort key

### Global Secondary Indexes (GSIs)

#### GSI1: Status and Customer Lookups
- **Hash Key**: GSI1PK
- **Sort Key**: GSI1SK
- **Purpose**: Status-based queries, customer-specific lookups

#### GSI2: SKU and Additional Indexes
- **Hash Key**: GSI2PK
- **Sort Key**: GSI2SK
- **Purpose**: SKU lookups, additional access patterns

#### ProductsByStatus: Efficient Product Listing
- **Hash Key**: EntityType (always "PRODUCT")
- **Sort Key**: CreatedAt (ISO 8601 timestamp)
- **Purpose**: Efficient product listing with proper pagination
- **Benefits**:
  - Replaces table-wide Scan with targeted Query
  - Native sorting by creation date
  - Proper cursor-based pagination
  - Avoids 1MB scan limit issues

### Module Variables

```hcl
module "dynamodb" {
  source = "../../modules/dynamodb"

  table_name            = "shop-api-dev"
  billing_mode          = "PAY_PER_REQUEST"  # or "PROVISIONED"

  # Only used if billing_mode = "PROVISIONED"
  table_read_capacity   = 5
  table_write_capacity  = 5
  gsi_read_capacity     = 5
  gsi_write_capacity    = 5

  enable_pitr           = true
  enable_encryption     = true
  ttl_attribute         = ""  # Optional TTL attribute name

  environment           = "dev"
  tags                  = {
    Service = "shop-api"
  }
}
```

## Migration from Scan to Query

The ProductsByStatus GSI enables a significant performance improvement for product listing:

**Before (Scan)**:
- Table-wide scan with filter expression
- Limited by 1MB scan cap
- Required scanning 100x more items to account for filtering
- Pagination breaks with filters

**After (Query)**:
```go
// Query products efficiently using GSI
queryInput := &dynamodb.QueryInput{
    TableName:              aws.String(tableName),
    IndexName:              aws.String("ProductsByStatus"),
    KeyConditionExpression: aws.String("EntityType = :entityType"),
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":entityType": &types.AttributeValueMemberS{Value: "PRODUCT"},
    },
    ScanIndexForward: aws.Bool(false), // Newest first
}
```

**Benefits**:
- Query instead of Scan (faster, more efficient)
- Proper pagination with DynamoDB's native cursors
- Native sorting by CreatedAt
- No 1MB scan limit issues
- Lower read costs

## Best Practices

### 1. Local Development Workflow

```bash
# 1. Start local services
docker compose up -d dynamodb-local localstack

# 2. Use LocalStack for Terraform testing
cd terraform/environments/local
terraform apply

# 3. Run shop-api locally
cd ../../..
make dev-up
```

### 2. Development Environment Workflow

```bash
# 1. Build Lambda function
make build

# 2. Deploy infrastructure
cd terraform/environments/dev
terraform apply

# 3. Test deployed API
curl https://dev-api.lemnispace.com/v1/shop/products
```

### 3. Infrastructure as Code Guidelines

- **Never** create AWS resources manually via console or CLI
- Always use Terraform for infrastructure changes
- Test infrastructure changes in `local` or `dev` before production
- Use modules for reusable components
- Keep environment-specific configurations in `environments/`

### 4. DynamoDB Access Patterns

When adding new features, ensure you:
1. Define the access pattern
2. Determine if existing GSIs support it
3. Add new GSI if needed (via module)
4. Update application code to use the GSI
5. Test with local DynamoDB first

## Troubleshooting

### LocalStack Connection Issues

```bash
# Check LocalStack is running
docker ps | grep localstack

# Test LocalStack endpoint
curl http://localhost:4566/_localstack/health

# List tables
aws dynamodb list-tables \
  --endpoint-url http://localhost:4566 \
  --region us-east-1
```

### DynamoDB Local Issues

```bash
# Check DynamoDB local is running
docker ps | grep dynamodb-local

# Describe table
aws dynamodb describe-table \
  --table-name ShopAPI \
  --endpoint-url http://localhost:8000 \
  --region us-east-1
```

### Terraform State Issues

```bash
# Local environment (no remote state)
cd terraform/environments/local
rm -rf .terraform terraform.tfstate*
terraform init

# Dev/Prod (with remote state)
cd terraform/environments/dev
terraform init -reconfigure
```

## Testing the GSI

To verify the ProductsByStatus GSI is working:

```bash
# 1. Create a test product
curl -X POST http://localhost:8080/v1/products \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Product",
    "status": "active",
    "price": 29.99
  }'

# 2. List products (should use GSI Query)
curl http://localhost:8080/v1/products?limit=10&sort=created_at&order=desc

# 3. Check CloudWatch logs or debug output for:
#    "Executing DynamoDB query on ProductsByStatus GSI"
```

## Next Steps

1. **Migrate Production**: Move `terraform/main.tf` to `terraform/environments/prod/`
2. **Add Staging**: Create `terraform/environments/staging/`
3. **CI/CD Integration**: Automate Terraform apply in GitHub Actions
4. **Monitoring**: Add CloudWatch alarms for DynamoDB metrics
5. **Cost Optimization**: Review and optimize GSI usage patterns
