# smsutils-volc

Volcengine (VolcEngine) SMS driver for [smsutils](https://go.gh.ink/smsutils).

This module implements the `model.Driver` / `model.Client` interfaces of the core
`smsutils` library on top of the Volcengine `volc-sdk-golang` SMS service.

Driver name: `volc`

## Installation

```bash
go get go.gh.ink/smsutils/volc/v3
```

## Usage

Blank-import this package so it registers itself, then configure it via the core client.

```go
import (
	"go.gh.ink/smsutils/v3/client"
	"go.gh.ink/smsutils/v3/model"

	_ "go.gh.ink/smsutils/volc/v3"
)

clients, err := client.NewClient(model.Config{
	Credentials: model.C{
		"volc": {
			"accessKey":  "<your-access-key>",
			"secretKey":  "<your-secret-key>",
			"smsAccount": "<your-sms-account>",
		},
	},
})
if err != nil {
	panic(err)
}

// "sender" is the Sign; "ST_..." is the TemplateID.
err = clients["volc"].SendMessage("+8617601205205", "<sign>", "ST_12345", model.Vars{
	{Key: "code", Value: "1234"},
})
```

## Context-aware sending

The existing `client.NewClient` and `SendMessage` APIs remain available. With the
updated local core and this driver, use the same configuration with the new factory:

```go
// Additional imports: "context", "time".
clients, err := client.NewClientCtx(config, model.ClientOptions{Timeout: 5 * time.Second})
if err != nil {
    return err
}
ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
defer cancel()
err = clients["volc"].SendMessageCtx(ctx, dest, sender, template, vars)
```

`config` has the credential layout shown above. `ClientOptions.Transport` optionally
injects a concurrency-safe HTTP transport. Context cancellation reaches the HTTP request;
a timeout does not prove the provider did not accept the SMS, so avoid blind retries.
Nil variable entries and malformed provider responses are rejected.

These additions require the updated core source. Existing dependency versions are
intentionally unchanged until release. For local testing, create a Go workspace containing
this module and the sibling `smsutils/v3` core module; see the core repository's
`CONTEXT.md` (local path: `../smsutils/CONTEXT.md`) for the complete six-module
commands and API contracts.

## Credentials

| Key | Constant | Required | Description |
|-----|----------|----------|-------------|
| `accessKey` | `AccessKey` | Yes | Volcengine Access Key |
| `secretKey` | `SecretKey` | Yes | Volcengine Secret Key |
| `smsAccount` | `SmsAccount` | Yes (per send) | Volcengine SMS account ID |

Missing `accessKey` or `secretKey` yields `errors.ErrDriverCredentialInvalid`.

## Behavior

- The destination is normalized via `utils.ProcessNumberForChinese` before sending.
- `template` maps to the Volcengine `TemplateID`; `sender` maps to `Sign`.
- `vars` are passed **by key**: they are flattened to a `map[string]string`, JSON-marshalled
  (using the configured `Marshal`), and sent as `TemplateParam`. Each `Var.Key` must match a
  placeholder in your template.
- A non-`200` HTTP status code or a non-nil `ResponseMetadata.Error` produces
  `errors.ErrDriverSendFailed`, decorated with the status code, provider message, request ID
  and raw result.

Each constructed client owns an SDK instance and credentials; the SDK's package-level
`DefaultInstance` is not configured by this driver.

## Requirements

- Go 1.25.0+
- `go.gh.ink/smsutils/v3`
- Volcengine SDK: `github.com/volcengine/volc-sdk-golang`

## License

[Apache License 2.0](LICENSE)
