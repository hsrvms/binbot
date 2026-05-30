#!/bin/bash
echo "--- Compiling Protobuf Data Contracts ---"

export PATH="$PATH:/root/go/bin:/usr/local/go/bin:/root/.local/bin:/usr/local/bin"

pip install --break-system-packages protobuf mypy-protobuf types-protobuf > /dev/null 2>&1

mkdir -p go-oms/internal/pb python-engine/pb

protoc -I=contracts --go_out=go-oms/internal/pb --go_opt=paths=source_relative contracts/trading/events.proto
echo "✅ Go structs generated in go-oms/internal/pb/trading"

protoc -I=contracts --python_out=python-engine/pb --mypy_out=python-engine/pb contracts/trading/events.proto
echo "✅ Python classes & type stubs generated in python-engine/pb/trading"

if [ -f "go-oms/internal/pb/trading/events.pb.go" ] && [ -f "python-engine/pb/trading/events_pb2.pyi" ]; then
    echo "--- Compilation Successful ---"
else
    echo "--- Compilation Failed! ---"
    exit 1
fi
