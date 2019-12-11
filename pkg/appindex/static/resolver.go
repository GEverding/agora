package static

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/kinecosystem/agora-common/kin"
	"github.com/kinecosystem/agora-transaction-services/pkg/appindex"
	"github.com/kinecosystem/kin-api/genproto/common/v3"
)

var (
	mapping = map[uint16]string{
		0: "test.kin.org", // todo: maybe keep 0 index reserved,
		1: "api.kik.com",  // placeholder
	}
)

type resolver struct{}

// New returns an appindex.Resolver with static mappings.
func New() appindex.Resolver {
	return &resolver{}
}

// Resolve implements appindex.Resolver.Resolve.
func (r *resolver) Resolve(_ context.Context, m kin.Memo) (*common.AgoraDataUrl, error) {
	domain, ok := mapping[m.AppIndex()]
	if !ok {
		return nil, appindex.ErrNotFound
	}

	return &common.AgoraDataUrl{
		// todo: proper callback spec
		Value: fmt.Sprintf("https://%s/agora/resolve/%s", domain, base64.URLEncoding.EncodeToString(m[:])),
	}, nil
}