alias get_token='go run ./cmd/hatchet-admin token create --name local --tenant-id 707d0855-80ab-4e1f-a156-f1c4546cbf52'

cat > ./examples/simple/.env <<EOF
HATCHET_CLIENT_TENANT_ID=707d0855-80ab-4e1f-a156-f1c4546cbf52
HATCHET_CLIENT_TLS_ROOT_CA_FILE=$PWD/hack/dev/certs/ca.cert
HATCHET_CLIENT_TLS_SERVER_NAME=cluster
HATCHET_CLIENT_TOKEN="$(get_token)"
EOF
