package main

import (
	"github.com/kr/pretty"
	"lsp/logger"
	"net/rpc/jsonrpc"
)

const address = "http://127.0.0.1:9877"

var log = logger.Get()

func main() {
	//dial, err2 := net.Dial("tcp", "127.0.0.1:9877")
	//if err2 != nil {
	//	panic(err2)
	//}
	//log.Infof(pretty.Sprint(dial))

	log.Infof("connect to lsp server start")
	client := jsonrpc.NewClient(address)
	log.Infof("connect to lsp server success")
	initializeParams := InitializeParams{}
	initializeParams.ClientInfo.Name = "test"
	initializeParams.ClientInfo.Version = "v1.0.0"
	initializeParams.ProcessID = 2233
	initializeParams.WorkspaceFolders = []WorkspaceFolder{{Name: "test", URI: "file://test"}}
	call, err := client.Call("initialize", &initializeParams)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("initialize: %s", pretty.Sprint(call))

	client = jsonrpc.NewClient(address)
	call, err = client.Call("initialized", InitializedParams{})
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("call json rpc response. response:%v", pretty.Sprint(call))

	didOpenParam := DidOpenTextDocumentParams{}
	didOpenParam.TextDocument.URI = "file://test//test.go"
	didOpenParam.TextDocument.Version = 0
	didOpenParam.TextDocument.LanguageID = "GO"
	didOpenParam.TextDocument.Text = "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	call, err = client.Call("textDocument/didOpen", &didOpenParam)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("call json rpc response. response:%v", pretty.Sprint(call))

	param := ReferenceParams{}
	param.TextDocument.URI = "file://test//test.go"
	param.Position.Line = 6
	param.Position.Character = 6
	param.Context = ReferenceContext{IncludeDeclaration: true}
	call, err = client.Call("textDocument/references", &param)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("call json rpc response. response:%v", pretty.Sprint(call))
}
