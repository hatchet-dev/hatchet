package encryption

// import (
// 	"github.com/tink-crypto/tink-go/aead"
// 	"github.com/tink-crypto/tink-go/proto/tink_go_proto"
// 	"github.com/tink-crypto/tink-go/tink"
// )

// func getEnvelopeKeyWithRemote(keyUri string)

// func getHandle(template *tink_go_proto.KeyTemplate, remote tink.AEAD) {
// 	envelope := aead.NewKMSEnvelopeAEAD2(aead.AES128GCMKeyTemplate(), a)

// 	if envelope == nil {
// 		return nil, fmt.Errorf("failed to create envelope")
// 	}

// 	aead.AES128GCMKeyTemplate()

// 	dek := aead.AES128CTRHMACSHA256KeyTemplate()
// 	template, err := aead.CreateKMSEnvelopeAEADKeyTemplate(keyUri, dek)

// 	if err != nil {
// 		return nil, err
// 	}

// 	handle, err := keyset.NewHandle(template)

// 	if err != nil {
// 		return nil, err
// 	}

// 	a, err := aead.New(handle)

// 	if err != nil {
// 		return nil, err
// 	}
// }
