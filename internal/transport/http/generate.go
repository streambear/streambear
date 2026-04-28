package http

//go:generate go run -modfile=../../../../go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -o commonapi/generated.go  ../../../api/common.yaml

//go_:generate go run -modfile=../../../../go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen  ../../../api/authorizer.yaml
