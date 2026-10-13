// Package api は docs/api/openapi.yaml から生成した型とサーバーインターフェースを置く（API-1）。
// api.gen.go は生成物なので編集しない。
package api

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../docs/api/openapi.yaml
