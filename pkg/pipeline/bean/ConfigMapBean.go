package bean

import (
	"encoding/base64"
	"encoding/json"
)

type ConfigDataRequest struct {
	Id            int           `json:"id"`
	AppId         int           `json:"appId"`
	EnvironmentId int           `json:"environmentId,omitempty"`
	ConfigData    []*ConfigData `json:"configData"`
	UserId        int32         `json:"-"`
}

type ESOSecretData struct {
	SecretStore     json.RawMessage `json:"secretStore,omitempty"`
	SecretStoreRef  json.RawMessage `json:"secretStoreRef,omitempty"`
	EsoData         []ESOData       `json:"esoData,omitempty"`
	RefreshInterval string          `json:"refreshInterval,omitempty"`
}

type ESOData struct {
	SecretKey string `json:"secretKey"`
	Key       string `json:"key"`
	Property  string `json:"property,omitempty"`
}

type ConfigData struct {
	Name                  string           `json:"name"`
	Type                  string           `json:"type"`
	External              bool             `json:"external"`
	MountPath             string           `json:"mountPath,omitempty"`
	Data                  json.RawMessage  `json:"data"`
	DefaultData           json.RawMessage  `json:"defaultData,omitempty"`
	DefaultMountPath      string           `json:"defaultMountPath,omitempty"`
	Global                bool             `json:"global"`
	ExternalSecretType    string           `json:"externalType"`
	ESOSecretData         ESOSecretData    `json:"esoSecretData"`
	DefaultESOSecretData  ESOSecretData    `json:"defaultESOSecretData,omitempty"`
	ExternalSecret        []ExternalSecret `json:"secretData"`
	DefaultExternalSecret []ExternalSecret `json:"defaultSecretData,omitempty"`
	RoleARN               string           `json:"roleARN"`
	SubPath               bool             `json:"subPath"`
	FilePermission        string           `json:"filePermission"`
	Overridden            bool             `json:"overridden"`
}
type ExternalSecret struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Property string `json:"property,omitempty"`
	IsBinary bool   `json:"isBinary"`
}

type BulkPatchRequest struct {
	Payload     []*BulkPatchPayload `json:"payload"`
	Filter      *BulkPatchFilter    `json:"filter,omitempty"`
	ProjectId   int                 `json:"projectId"`
	Global      bool                `json:"global"`
	Type        string              `json:"type"`
	Name        string              `json:"name"`
	Key         string              `json:"key"`
	Value       string              `json:"value"`
	PatchAction int                 `json:"patchAction"` // 1=add, 2=update, 0=delete
	UserId      int32               `json:"-"`
}

type BulkPatchPayload struct {
	AppId int `json:"appId"`
	EnvId int `json:"envId"`
}

type BulkPatchFilter struct {
	AppNameIncludes string `json:"appNameIncludes,omitempty"`
	AppNameExcludes string `json:"appNameExcludes,omitempty"`
	EnvId           int    `json:"envId,omitempty"`
}

type JobEnvOverrideResponse struct {
	Id              int    `json:"id"`
	AppId           int    `json:"appId"`
	EnvironmentId   int    `json:"environmentId,omitempty"`
	EnvironmentName string `json:"environmentName,omitempty"`
}

type CreateJobEnvOverridePayload struct {
	AppId  int   `json:"appId"`
	EnvId  int   `json:"envId"`
	UserId int32 `json:"-"`
}

type SecretsList struct {
	ConfigData []*ConfigData `json:"secrets"`
}
type SecretTransformMode int

const (
	EncodeSecret SecretTransformMode = 1
	DecodeSecret SecretTransformMode = 2
)

func GetTransformedDataForSecret(data string, mode SecretTransformMode) (string, error) {
	secretsList := SecretsList{}
	err := json.Unmarshal([]byte(data), &secretsList)
	if err != nil {
		return "", err
	}

	for _, configData := range secretsList.ConfigData {
		configData.Data = GetDecodedAndEncodedData(configData.Data, mode)
	}

	marshal, err := json.Marshal(secretsList)
	if err != nil {
		return "", err
	}
	return string(marshal), nil
}

func GetDecodedAndEncodedData(data json.RawMessage, transformer SecretTransformMode) []byte {
	dataMap := make(map[string]string)
	err := json.Unmarshal(data, &dataMap)
	if err != nil {
		return nil
	}
	var transformedData []byte
	for key, value := range dataMap {
		switch transformer {
		case EncodeSecret:
			transformedData = []byte(base64.StdEncoding.EncodeToString([]byte(value)))
		case DecodeSecret:
			transformedData, err = base64.StdEncoding.DecodeString(value)
		}

		dataMap[key] = string(transformedData)
	}
	marshal, err := json.Marshal(dataMap)
	if err != nil {
		return nil
	}
	return marshal
}
