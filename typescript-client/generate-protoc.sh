# Directory to write generated code to (.js and .d.ts files)
OUT_DIR="./protoc"

# Generate code
pnpm grpc_tools_node_protoc \
    --js_out=import_style=commonjs,binary:$OUT_DIR \
    --grpc_out=grpc_js:$OUT_DIR \
    --plugin=protoc-gen-grpc=`which grpc_tools_node_protoc_plugin` \
    -I ../api-contracts \
    ../api-contracts/**/*.proto

# Generate types
protoc \
    --plugin=protoc-gen-ts=./node_modules/.bin/protoc-gen-ts \
    --ts_out=grpc_js:$OUT_DIR \
    -I ../api-contracts \
    ../api-contracts/**/*.proto