package group

import (
	"chat-go/apps/social/api/internal/logic/group"
	"chat-go/apps/social/api/internal/svc"
	"chat-go/apps/social/api/internal/types"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GroupUserListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GroupUserListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.Error(w, err)
			return
		}

		l := group.NewGroupUserListLogic(r.Context(), svcCtx)
		resp, err := l.GroupUserList(&req)
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}
