# Directory to write generated code to (.js and .d.ts files)
OUT_DIR="./protoc"

pnpm proto-loader-gen-types \
    --grpcLib=@grpc/grpc-js \
    --outDir=${OUT_DIR} \
    ../api-contracts/dispatcher/*.proto \
    ../api-contracts/events/*.proto \
    ../api-contracts/workflows/*.proto \