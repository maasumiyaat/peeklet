# SAM invokes `build-ApiFunction` during `sam build`, passing ARTIFACTS_DIR.
# The compiled binary MUST be named `bootstrap` (provided.al2023 contract).
build-ApiFunction:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc \
		-o $(ARTIFACTS_DIR)/bootstrap ./cmd/lambda

.PHONY: tidy build deploy keys hashpw local-health

tidy:
	go mod tidy

build: tidy
	sam build

# First deploy is interactive and writes samconfig.toml; later just `make deploy`.
deploy: build
	sam deploy --guided

# Generate the CloudFront RSA keypair (prints the two PEM parameter values).
keys:
	./scripts/gen-keys.sh

# Print a bcrypt hash for the owner password: make hashpw PW=yourpassword
hashpw:
	go run ./cmd/hashpw "$(PW)"

# Smoke-test the built function locally (needs Docker for `sam local`).
local-health:
	sam local invoke ApiFunction --event events/health.json