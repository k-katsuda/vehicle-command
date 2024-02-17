package authentication

import (
	"time"

	"github.com/k-katsuda/vehicle-command/pkg/protocol/protobuf/signatures"
	universal "github.com/k-katsuda/vehicle-command/pkg/protocol/protobuf/universalmessage"
)

// sessionInfo provides an interface for extracting metadata from both HMAC- and GCM-authenticated messages.
//
// Autogenerated code implements this interface for the respective message types.
type sessionInfo interface {
	GetCounter() uint32
	GetEpoch() []byte
	GetExpiresAt() uint32
}

// A Peer is the parent type for Signer and Verifier.
type Peer struct {
	domain       universal.Domain
	verifierName []byte
	counter      uint32
	epoch        [epochIdLength]byte
	timeZero     time.Time
	session      Session
}

func (p *Peer) timestamp() uint32 {
	return uint32(time.Since(p.timeZero) / time.Second)
}

// extractMetadata populates metadata.
func (p *Peer) extractMetadata(meta *metadata, message *universal.RoutableMessage, info sessionInfo, method signatures.SignatureType) error {
	meta.Add(signatures.Tag_TAG_SIGNATURE_TYPE, []byte{byte(method)})

	// Authenticate domain. Use domain from message because sender might be using BROADCAST.
	if x, ok := message.ToDestination.GetSubDestination().(*universal.Destination_Domain); ok {
		if 0 > x.Domain || x.Domain > 255 {
			return newError(errCodeInvalidDomain, "domain out of range")
		}
		meta.Add(signatures.Tag_TAG_DOMAIN, []byte{byte(x.Domain)})
	} else {
		return newError(errCodeInvalidDomain, "domain missing")
	}

	if err := meta.Add(signatures.Tag_TAG_PERSONALIZATION, p.verifierName); err != nil {
		return newError(errCodeWrongPerso, "recipient name too long")
	}

	expires := time.Duration(info.GetExpiresAt()) * time.Second

	// Bounds check ensures: (1) we can encode in a 4-byte buffer and (2)
	// will not overflow time.Duration.
	if expires > epochLength || expires < 0 {
		return newError(errCodeBadParameter, "out of bounds expiration time")
	}

	meta.Add(signatures.Tag_TAG_EPOCH, p.epoch[:])
	meta.AddUint32(signatures.Tag_TAG_EXPIRES_AT, info.GetExpiresAt())
	meta.AddUint32(signatures.Tag_TAG_COUNTER, info.GetCounter())

	// For backwards compatibility, message flags are only explicitly added to
	// the metadata hash if at least one of them is set. (If a MITM
	// clears these bits, the hashes will not match, as desired).
	if message.Flags > 0 {
		meta.AddUint32(signatures.Tag_TAG_FLAGS, message.Flags)
	}

	return nil
}

// hmacTag computes the HMAC-SHA256 tag for a message.
func (p *Peer) hmacTag(message *universal.RoutableMessage, hmacData *signatures.HMAC_Personalized_Signature_Data) ([]byte, error) {
	meta := newMetadataHash(p.session.NewHMAC(labelMessageAuth))
	if err := p.extractMetadata(meta, message, hmacData, signatures.SignatureType_SIGNATURE_TYPE_HMAC_PERSONALIZED); err != nil {
		return nil, err
	}
	return meta.Checksum(message.GetProtobufMessageAsBytes()), nil
}
