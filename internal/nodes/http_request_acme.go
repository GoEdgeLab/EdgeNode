package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"path/filepath"
)

func (this *HTTPRequest) doACME() (shouldStop bool) {
	// TODO 对请求进行校验，防止恶意攻击

	var token = filepath.Base(this.RawReq.URL.Path)
	if token == "acme-challenge" || len(token) <= 32 {
		return false
	}

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		remotelogs.Error("RPC", "[ACME]rpc failed: "+err.Error())
		return false
	}

	keyResp, err := rpcClient.ACMEAuthenticationRPC.FindACMEAuthenticationKeyWithToken(rpcClient.Context(), &pb.FindACMEAuthenticationKeyWithTokenRequest{Token: token})
	if err != nil {
		remotelogs.Error("RPC", "[ACME]read key for token failed: "+err.Error())
		return false
	}
	if len(keyResp.Key) == 0 {
		return false
	}

	this.tags = append(this.tags, "ACME")

	this.writer.Header().Set("Content-Type", "text/plain")
	_, _ = this.writer.WriteString(keyResp.Key)

	return true
}
