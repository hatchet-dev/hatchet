# Directory to write generated code to (.js and .d.ts files)
OUT_DIR="./src/protoc"

# Generate code
./node_modules/.bin/grpc_tools_node_protoc \
  --plugin=protoc-gen-ts_proto=./node_modules/.bin/protoc-gen-ts_proto \
  --ts_proto_out=$OUT_DIR \
  --ts_proto_opt=outputServices=nice-grpc,outputServices=generic-definitions,useExactTypes=false \
  --proto_path=../api-contracts \
  ../api-contracts/**/*.proto