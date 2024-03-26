package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/darmiel/go-xsenv"
)

func Must[T any](t T, err error) T {
	MustErr(err)
	return t
}

func MustErr(err error) {
	if err != nil {
		panic(err)
	}
}

// UAAConfig is some arbitrary configuration struct.
type UAAConfig struct {
	ClientID  string `json:"clientid"`
	XSAppName string `json:"xsappname"`
	URL       string `json:"url"`
	UAADomain string `json:"uaadomain"`
}

// UnmarshalService is an example how to unmarshal a service configuration.
func (u *UAAConfig) UnmarshalService(message *json.RawMessage) error {
	parsed := struct {
		Credentials UAAConfig `json:"credentials"`
	}{}
	if err := json.Unmarshal(*message, &parsed); err != nil {
		return err
	}

	// you can use MissingFieldError to indicate missing fields
	if parsed.Credentials.URL == "" {
		return xsenv.MissingFieldError("url")
	}

	// or use CheckAllFields to check all fields at once
	if err := xsenv.CheckAllFields(xsenv.Fields{
		"clientid":  parsed.Credentials.ClientID != "",
		"xsappname": parsed.Credentials.XSAppName != "",
		"uaadomain": parsed.Credentials.UAADomain != "",
	}); err != nil {
		return err
	}

	*u = parsed.Credentials
	return nil
}

func main() {
	env := Must(xsenv.LoadEnv())
	fmt.Println(env.ServicesByName)

	var uaa UAAConfig
	if err := env.LoadService(&uaa, "portal-uaa"); err != nil {
		if errors.Is(err, xsenv.ErrFieldMissing) {
			fmt.Println("Missing Field:", err)
		} else {
			panic(err)
		}
	}

	fmt.Printf("UAA: %+v\n", uaa)
}
