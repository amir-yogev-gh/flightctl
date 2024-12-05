// Package v1alpha1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.3.0 DO NOT EDIT.
package v1alpha1

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
