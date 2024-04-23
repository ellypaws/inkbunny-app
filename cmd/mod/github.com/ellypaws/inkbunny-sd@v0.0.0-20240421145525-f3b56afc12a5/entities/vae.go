// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    vAEs, err := UnmarshalVAEs(bytes)
//    bytes, err = vAEs.Marshal()

package entities

import "encoding/json"

type VAEs []VAE

func UnmarshalVAEs(data []byte) (VAEs, error) {
	var r VAEs
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *VAEs) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type VAE struct {
	ModelName string `json:"model_name"`
	Filename  string `json:"filename"`
}
