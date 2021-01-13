package private_test

import (
	"context"
	"os"
	"testing"

	"github.com/luthersystems/substratecommon"
	"github.com/luthersystems/substratecommon/private"
	"github.com/luthersystems/substratecommon/substratewrapper"
)

func TestPrivate(t *testing.T) {
	var tests = []struct {
		Name string
		Func func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon)
	}{
		{
			Name: "export missing",
			Func: func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon) {
				var exportedData map[string]interface{}
				err := private.Export(context.Background(), client, "DSID-missing", exportedData)
				if err == nil {
					t.Fatal("expected error")
				}
			},
		},
		{
			Name: "purge missing",
			Func: func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon) {
				err := private.Purge(context.Background(), client, "DSID-missing")
				if err == nil {
					t.Fatal("expected error")
				}
			},
		},
		{
			Name: "profile to missing DSID",
			Func: func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon) {
				_, err := private.ProfileToDSID(context.Background(), client, []string{"profile-missing"})
				if err == nil {
					t.Fatal("expected error")
				}
			},
		},
		{
			Name: "encode zero transforms",
			Func: func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon) {
				message := struct {
					Hello string
					Fnord string
				}{
					"world",
					"fnord",
				}
				var transforms []*private.Transform
				_, err := private.Encode(context.Background(), client, message, transforms)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			},
		},
		{
			Name: "encode and decode (zero transforms)",
			Func: func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon) {
				message := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{
					"world",
					"fnord",
				}
				var transforms []*private.Transform
				resp, err := private.Encode(context.Background(), client, message, transforms)
				if err != nil {
					t.Fatalf("encode: %s", err)
				}
				decodedMessage := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{}
				err = private.Decode(context.Background(), client, resp, &decodedMessage)
				if err != nil {
					t.Fatalf("decode: %s", err)
				}
				if message != decodedMessage {
					t.Fatalf("message mismatch, expected: %v != got: %v", message, decodedMessage)
				}
			},
		},
		{
			Name: "encode and decode (1 transform)",
			Func: func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon) {
				message := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{
					"world",
					"fnord",
				}
				var transforms []*private.Transform
				transforms = append(transforms, &private.Transform{
					ContextPath: ".",
					Header: &private.TransformHeader{
						ProfilePaths: []string{".fnord"},
						PrivatePaths: []string{"."},
						Encryptor:    private.EncryptorAES256,
						Compressor:   private.CompressorZlib,
					},
				})
				resp, err := private.Encode(context.Background(), client, message, transforms)
				if err != nil {
					t.Fatalf("encode: %s", err)
				}
				decodedMessage := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{}
				err = private.Decode(context.Background(), client, resp, &decodedMessage)
				if err != nil {
					t.Fatalf("decode: %s", err)
				}
				if message != decodedMessage {
					t.Fatalf("message mismatch")
				}
			},
		},
		{
			Name: "wrap",
			Func: func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon) {
				message := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{
					"world",
					"fnord",
				}
				var transforms []*private.Transform
				transforms = append(transforms, &private.Transform{
					ContextPath: ".",
					Header: &private.TransformHeader{
						ProfilePaths: []string{".fnord"},
						PrivatePaths: []string{"."},
						Encryptor:    private.EncryptorAES256,
						Compressor:   private.CompressorZlib,
					},
				})
				wrap := private.WrapCall(context.Background(), client, "wrap_all", transforms...)
				decodedMessage := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{}
				config, err := private.WithSeed()
				if err != nil {
					t.Fatalf("iv: %s", err)
				}
				err = wrap(message, &decodedMessage, config)
				if err != nil {
					t.Fatalf("wrap: %s", err)
				}
				if message != decodedMessage {
					t.Fatalf("message mismatch: expected: %v != got: %v", message, decodedMessage)
				}
			},
		},
		{
			Name: "no wrap (encode/decode passthrough)",
			Func: func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon) {
				message := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{
					"world",
					"fnord",
				}
				var transforms []*private.Transform
				wrap := private.WrapCall(context.Background(), client, "wrap_none", transforms...)
				decodedMessage := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{}
				err := wrap(message, &decodedMessage)
				if err != nil {
					t.Fatalf("wrap: %s", err)
				}
				if message != decodedMessage {
					t.Fatalf("message mismatch: expected: %v != got: %v", message, decodedMessage)
				}
			},
		},
		{
			// IMPORTANT: this test must run after `wrap`!
			Name: "partial wrap (no encode, yes decode)",
			Func: func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon) {
				message := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{
					"world",
					"fnord",
				}
				var transforms []*private.Transform
				wrap := private.WrapCall(context.Background(), client, "wrap_output", transforms...)
				decodedMessage := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{}
				err := wrap(message, &decodedMessage)
				if err != nil {
					t.Fatalf("wrap: %s", err)
				}
				if message != decodedMessage {
					t.Fatalf("message mismatch: expected: %v != got: %v", message, decodedMessage)
				}
			},
		},
		{
			Name: "partial wrap (yes encode, no decode)",
			Func: func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon) {
				message := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{
					"world",
					"fnord",
				}
				var transforms []*private.Transform
				transforms = append(transforms, &private.Transform{
					ContextPath: ".",
					Header: &private.TransformHeader{
						ProfilePaths: []string{".fnord"},
						PrivatePaths: []string{"."},
						Encryptor:    private.EncryptorAES256,
						Compressor:   private.CompressorZlib,
					},
				})
				wrap := private.WrapCall(context.Background(), client, "wrap_input", transforms...)
				decodedMessage := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{}
				err := wrap(message, &decodedMessage)
				if err != nil {
					t.Fatalf("wrap: %s", err)
				}
				if message != decodedMessage {
					t.Fatalf("message mismatch: expected: %v != got: %v", message, decodedMessage)
				}
			},
		},
		{
			Name: "wrap error (no IV)",
			Func: func(t *testing.T, client *substratewrapper.SubstrateInstanceWrapperCommon) {
				message := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{
					"world",
					"fnord",
				}
				var transforms []*private.Transform
				transforms = append(transforms, &private.Transform{
					ContextPath: ".",
					Header: &private.TransformHeader{
						ProfilePaths: []string{".fnord"},
						PrivatePaths: []string{"."},
						Encryptor:    private.EncryptorAES256,
						Compressor:   private.CompressorZlib,
					},
				})
				wrap := private.WrapCall(context.Background(), client, "wrap_all", transforms...)
				decodedMessage := struct {
					Hello string `json:"hello"`
					Fnord string `json:"fnord"`
				}{}
				err := wrap(message, &decodedMessage)
				if err == nil {
					t.Fatalf("expected IV error")
				}
			},
		},
	}
	err := substratecommon.Connect(
		func(substrate substratecommon.Substrate) error {
			sw := substratewrapper.NewSubstrateWrapper(substrate)

			siwm, err := sw.NewMockFrom("test", "test", nil)
			if err != nil {
				return err
			}

			phylumString := `
(in-package 'sample)
(use-package 'router)

(defendpoint "init" ()
             (route-success ()))

(defendpoint "wrap_all" (msg)
             (handler-bind ((csprng-uninitialized (lambda (c &rest _)
                                                    (route-failure "missing CSPRNG seed"))))
               (let* ([dec (private:mxf-decode msg)]
                      [dec-msg (first dec)]
                      [dec-mxf (second dec)]
                      [new-enc (private:put-mxf "test-key" dec-msg dec-mxf)])
                 (route-success new-enc))))

(defendpoint "wrap_none" (msg)
             (route-success msg))

(defendpoint "wrap_output" (msg)
             (route-success (statedb:get "test-key")))

(defendpoint "wrap_input" (msg)
             (let* ([dec (private:mxf-decode msg)]
                    [dec-msg (first dec)])
               (route-success dec-msg)))
`

			err = siwm.Init(substratecommon.EncodePhylumBytes(phylumString))
			if err != nil {
				return err
			}

			for _, tc := range tests {
				t.Run(tc.Name, func(t *testing.T) {
					t.Logf("running: %s\n", tc.Name)
					tc.Func(t, siwm.Upcast())
				})
			}

			return siwm.CloseMock()
		},
		substratecommon.ConnectWithCommand(os.Getenv("SUBSTRATEHCP_FILE")))
	if err != nil {
		t.Fatal(err)
	}
}
