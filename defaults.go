package micro

import (
	"c-z.dev/go-micro/client"
	"c-z.dev/go-micro/debug/trace"
	"c-z.dev/go-micro/server"
	"c-z.dev/go-micro/store"

	// set defaults
	gcli "c-z.dev/go-micro/client/grpc"
	memTrace "c-z.dev/go-micro/debug/trace/memory"
	gsrv "c-z.dev/go-micro/server/grpc"
	memoryStore "c-z.dev/go-micro/store/memory"
)

func init() {
	// default client
	client.DefaultClient = gcli.NewClient()
	// default server
	server.DefaultServer = gsrv.NewServer()
	// default store
	store.DefaultStore = memoryStore.NewStore()
	// set default trace
	trace.DefaultTracer = memTrace.NewTracer()
}
