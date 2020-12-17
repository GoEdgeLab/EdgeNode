package nodes

import (
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeNode/internal/remotelogs"
	"github.com/TeaOSLab/EdgeNode/internal/rpc"
	"net/http"
	"path/filepath"
)

func (this *HTTPRequest) doACME() {
	// TODO 对请求进行校验，防止恶意攻击

	token := filepath.Base(this.RawReq.URL.Path)

	rpcClient, err := rpc.SharedRPC()
	if err != nil {
		remotelogs.Error("RPC", "[ACME]rpc failed: "+err.Error())
		return
	}

	keyResp, err := rpcClient.ACMEAuthenticationRPC().FindACMEAuthenticationKeyWithToken(rpcClient.Context(), &pb.FindACMEAuthenticationKeyWithTokenRequest{Token: token})
	if err != nil {
		remotelogs.Error("RPC", "[ACME]read key for token failed: "+err.Error())
		return
	}
	if len(keyResp.Key) == 0 {
		this.writer.WriteHeader(http.StatusNotFound)
	} else {
		this.writer.Header().Set("Content-Type", "text/plain")
		_, _ = this.writer.WriteString(keyResp.Key)
	}
}
