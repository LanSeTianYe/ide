package main

import (
	"context"
	"github.com/kr/pretty"
	"github.com/sourcegraph/jsonrpc2"
	"lsp/logger"
	"net"
)

const address = "http://127.0.0.1:9877"

var log = logger.Get()

func main() {
	conn, err2 := net.Dial("tcp", "127.0.0.1:9877")
	if err2 != nil {
		panic(err2)
	}
	log.Infof(pretty.Sprint(conn))
	log.Infof("connect to lsp server start")
	stream := jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{})
	ctx := context.Background()
	handler := &LSPHandler{}
	client := jsonrpc2.NewConn(ctx, stream, handler, jsonrpc2.LogMessages(&Logger{}))
	log.Infof("connect to lsp server success")
	initializeParams := InitializeParams{}
	initializeParams.ClientInfo.Name = "test"
	initializeParams.ClientInfo.Version = "v1.0.0"
	initializeParams.ProcessID = 2233
	initializeParams.WorkspaceFolders = []WorkspaceFolder{{Name: "test", URI: "file://test"}}
	response := make(map[string]interface{})
	err := client.Call(ctx, "initialize", initializeParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("initialize: %s", pretty.Sprint(response))

	response = make(map[string]interface{})
	err = client.Call(ctx, "initialized", InitializedParams{}, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("initialized: %s", pretty.Sprint(response))

	//createFilesParams := CreateFilesParams{}
	//createFilesParams.Files = []FileCreate{FileCreate{URI: "file://test//test.go"}}
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "workspace/willCreateFiles", createFilesParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/willCreateFiles: %s", pretty.Sprint(response))

	didOpenParam := DidOpenTextDocumentParams{}
	didOpenParam.TextDocument.URI = "file://test//test.go"
	didOpenParam.TextDocument.Version = 0
	didOpenParam.TextDocument.LanguageID = "GO"
	didOpenParam.TextDocument.Text = "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/didOpen", didOpenParam, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/didOpen: %s", pretty.Sprint(response))

	didSaveTextDocumentParams := DidSaveTextDocumentParams{}
	didSaveTextDocumentParams.TextDocument.URI = "file://test//test.go"
	data := "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	didSaveTextDocumentParams.Text = &data
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/didSave", didSaveTextDocumentParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/didSave: %s", pretty.Sprint(response))

	didCloseTextDocumentParams := DidCloseTextDocumentParams{}
	didCloseTextDocumentParams.TextDocument.URI = "file://test//test.go"
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/didClose", didCloseTextDocumentParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/didClose: %s", pretty.Sprint(response))

	referenceParams := ReferenceParams{}
	referenceParams.TextDocument.URI = "file://test//test.go"
	referenceParams.Position.Line = 6
	referenceParams.Position.Character = 6
	referenceParams.Context = ReferenceContext{IncludeDeclaration: true}
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/references", referenceParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("initialized: %s", pretty.Sprint(response))

}

type LSPHandler struct {
}

func (l *LSPHandler) Handle(context context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) {
	log.Infof(pretty.Sprint(request))
}

type Logger struct {
}

func (l *Logger) Printf(format string, v ...interface{}) () {
	log.Infof(format, v...)
}
