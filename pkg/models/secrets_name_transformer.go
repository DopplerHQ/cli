/*
Copyright © 2021 Doppler <support@doppler.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package models

type SecretsNameTransformer struct {
	Name      string
	Type      string
	EnvCompat bool
}

var UpperCamelTransformer = &SecretsNameTransformer{
	Name:      "Upper Camel",
	Type:      "upper-camel",
	EnvCompat: true,
}
var CamelTransformer = &SecretsNameTransformer{
	Name:      "Camel",
	Type:      "camel",
	EnvCompat: true,
}
var LowerKebabTransformer = &SecretsNameTransformer{
	Name:      "Lower Kebab",
	Type:      "lower-kebab",
	EnvCompat: true,
}
var LowerSnakeTransformer = &SecretsNameTransformer{
	Name:      "Lower Snake",
	Type:      "lower-snake",
	EnvCompat: true,
}
var TFVarTransformer = &SecretsNameTransformer{
	Name:      "TF Var",
	Type:      "tf-var",
	EnvCompat: true,
}
var DotNETTransformer = &SecretsNameTransformer{
	Name:      ".NET",
	Type:      "dotnet",
	EnvCompat: false,
}
var DotNETEnvTransformer = &SecretsNameTransformer{
	Name:      ".NET (ENV)",
	Type:      "dotnet-env",
	EnvCompat: true,
}

var SecretsNameTransformersList = []*SecretsNameTransformer{
	UpperCamelTransformer,
	CamelTransformer,
	LowerKebabTransformer,
	LowerSnakeTransformer,
	TFVarTransformer,
	DotNETTransformer,
	DotNETEnvTransformer,
}

var SecretsNameTransformerTypes []string
var SecretsEnvCompatNameTransformerTypes []string
var SecretsNameTransformerMap map[string]*SecretsNameTransformer

func init() {
	SecretsNameTransformerMap = map[string]*SecretsNameTransformer{}
	for _, transformer := range SecretsNameTransformersList {
		SecretsNameTransformerTypes = append(SecretsNameTransformerTypes, transformer.Type)
		SecretsNameTransformerMap[transformer.Type] = transformer
		if transformer.EnvCompat {
			SecretsEnvCompatNameTransformerTypes = append(SecretsEnvCompatNameTransformerTypes, transformer.Type)
		}
	}
}
