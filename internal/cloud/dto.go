package cloud

import (
	"encoding/json"
	"strconv"
)

type FlexibleID string

func (f *FlexibleID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexibleID(s)
		return nil
	}
	var i int64
	if err := json.Unmarshal(data, &i); err == nil {
		*f = FlexibleID(strconv.FormatInt(i, 10))
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(data, &num); err == nil {
		*f = FlexibleID(num.String())
		return nil
	}
	*f = FlexibleID(string(data))
	return nil
}

func (f FlexibleID) String() string {
	return string(f)
}

type ConnectRequest struct {
	Token string `json:"access_token"`
}

type ConnectResponse struct {
	LoginServer   string     `json:"login_server"`
	PreauthKey    string     `json:"preauth_key"`
	Hostname      string     `json:"hostname"`
	TailscaleIP   string     `json:"tailscale_ip"`
	TailscaleIPv6 string     `json:"tailscale_ipv6,omitempty"`
	ConnectionID  FlexibleID `json:"connection_id"`
	ExpiresAt     string     `json:"expires_at,omitempty"`
	ContainerName string     `json:"container_name,omitempty"`
	Name          string     `json:"name,omitempty"`
}

type ConnectApiResponse struct {
	Authorized bool             `json:"authorized"`
	Connection *ConnectResponse `json:"connection"`
	ConnectResponse
}

type ConfirmRequest struct {
	ConnectionID interface{} `json:"connection_id"`
}

type ConfirmResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}
