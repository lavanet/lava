package rest

/* legacy, removed next version */

import (
	"log"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	govrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
)

func PlansAddProposalRESTHandler(clientCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "plans_add",
		Handler:  postPlansAddProposalHandlerFn(clientCtx),
	}
}

func postPlansAddProposalHandlerFn(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("postPlansAddProposalHandlerFn")
	}
}

func PlansDelProposalRESTHandler(clientCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "plans_del",
		Handler:  postPlansDelProposalHandlerFn(clientCtx),
	}
}

func postPlansDelProposalHandlerFn(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("postPlansDelProposalHandlerFn")
	}
}
