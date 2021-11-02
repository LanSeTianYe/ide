package main

import (
	"context"
	"encoding/json"
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

	setTraceParams := SetTraceParams{Value: "verbose"}
	response := make(map[string]interface{})
	err := client.Call(ctx, "$/setTrace", &setTraceParams, &response, nil)
	if err != nil {
		log.Errorf("$/setTrace json rpc method failed. err: %s", err)
	}
	log.Infof("$/setTrace response: %s", pretty.Sprint(response))

	initializeParams := InitializeParams{}
	initializeParams.ClientInfo.Name = "test"
	initializeParams.ClientInfo.Version = "v1.0.0"
	initializeParams.WorkspaceFolders = []WorkspaceFolder{{Name: "test", URI: "file://test"}}
	result := InitializeResult{}
	log.Infof("initialize request: %s", pretty.Sprint(initializeParams))
	err = client.Call(ctx, "initialize", initializeParams, &result, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("initialize response: %s", pretty.Sprint(result))

	response = make(map[string]interface{})
	err = client.Call(ctx, "initialized", InitializedParams{}, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("initialized: %s", pretty.Sprint(response))

	didOpenParam := DidOpenTextDocumentParams{}
	didOpenParam.TextDocument.URI = "file://test//hello.go"
	didOpenParam.TextDocument.Version = 0
	didOpenParam.TextDocument.LanguageID = "go"
	didOpenParam.TextDocument.Text = "package main\n\nimport \"fmt\"\nimport \"github.com/sourcegraph/jsonrpc2\"\n\nfunc main() {\n\tresult := jsonrpc2.NewBufferedStream(nil, nil)\n\tfmt.Println(result)\n\tfmt.Println(\"Hello Ide!\")\n}\n"
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/didOpen", didOpenParam, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/didOpen response: %s", pretty.Sprint(response))

	didSaveTextDocumentParams := DidSaveTextDocumentParams{}
	didSaveTextDocumentParams.TextDocument.URI = "file://test//hello.go"
	data := "package main\n\nimport \"fmt\"\nimport \"github.com/sourcegraph/jsonrpc2\"\n\nfunc main() {\n\tresult := jsonrpc2.NewBufferedStream(nil, nil)\n\tfmt.Println(result)\n\tfmt.Println(\"Hello Ide!\")\n}\n"
	didSaveTextDocumentParams.Text = &data
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/didSave", &didSaveTextDocumentParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/didSave: %s", pretty.Sprint(response))

	didOpenParam = DidOpenTextDocumentParams{}
	didOpenParam.TextDocument.URI = "file://test//go.mod"
	didOpenParam.TextDocument.Version = 0
	didOpenParam.TextDocument.LanguageID = "go"
	didOpenParam.TextDocument.Text = "module hello\n\ngo 1.16\n"
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/didOpen", didOpenParam, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/didOpen response: %s", pretty.Sprint(response))

	didSaveTextDocumentParams = DidSaveTextDocumentParams{}
	didSaveTextDocumentParams.TextDocument.URI = "file://test//go.mod"
	data = "module hello\n\ngo 1.16\n"
	didSaveTextDocumentParams.Text = &data
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/didSave", &didSaveTextDocumentParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/didSave: %s", pretty.Sprint(response))

	completionParams := CompletionParams{}
	completionParams.WorkDoneToken = "333333333333333333333"
	completionParams.TextDocument.URI = "file://test//hello.go"
	completionParams.Position.Line = 1
	completionParams.Position.Character = 4

	log.Infof("textDocument/completion request: %s", pretty.Sprint(completionParams))
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/completion", &completionParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/completion: %s", pretty.Sprint(response))

	//在工作空间执行命令
	executeParams1 := ExecuteCommandParams{}
	executeParams1.Command = "gopls.tidy"
	executeParams1.Arguments = []json.RawMessage{[]byte(`{"URI":"file://test/go.mod"}`)}
	executeParams1.WorkDoneToken = "11111111111111111"
	response = make(map[string]interface{})
	//executeParams.Arguments = []json.RawMessage{[]byte("[version]")}
	err = client.Call(ctx, "workspace/executeCommand", &executeParams1, &response, nil)
	if err != nil {
		log.Errorf("gopls.tidy call json rpc method failed. err: %s", err)
	}
	log.Infof("workspace/executeCommand gopls.tidy: %s", pretty.Sprint(response))

	//在工作空间查询符号
	params := WorkspaceSymbolParams{}
	params.Query = "math"
	symbolResponse := make([]interface{}, 0)
	err = client.Call(ctx, "workspace/symbol", &params, &symbolResponse, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("workspace/symbol: %s", pretty.Sprint(symbolResponse))

	//在工作空间执行命令
	executeParams := ExecuteCommandParams{}
	executeParams.Command = "gopls.list_known_packages"
	executeParams.WorkDoneToken = "2222222222222222"
	executeParams.Arguments = []json.RawMessage{[]byte(`{"URI":"file://test/hello1.go"}`)}
	response = make(map[string]interface{})
	//executeParams.Arguments = []json.RawMessage{[]byte("[version]")}
	err = client.Call(ctx, "workspace/executeCommand", &executeParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("workspace/executeCommand: %s", pretty.Sprint(response))

	//在工作空间执行命令
	executeParams2 := ExecuteCommandParams{}
	executeParams2.Command = "gopls.vendor"
	executeParams2.Arguments = []json.RawMessage{[]byte(`{"URI":"file://test/go.mod"}`)}
	response = make(map[string]interface{})
	//executeParams.Arguments = []json.RawMessage{[]byte("[version]")}
	err = client.Call(ctx, "workspace/executeCommand", &executeParams2, &response, nil)
	if err != nil {
		log.Errorf("gopls.vendor call json rpc method failed. err: %s", err)
	}
	log.Infof("workspace/executeCommand gopls.vendor: %s", pretty.Sprint(response))

	//todo not implement
	//createFilesParams := CreateFilesParams{}
	//createFilesParams.Files = []FileCreate{{URI: "file://hello1.go"}}
	//err = client.Call(ctx, "workspace/didCreateFiles", &createFilesParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/didCreateFiles: %s", pretty.Sprint(response))

	didOpenParam = DidOpenTextDocumentParams{}
	didOpenParam.TextDocument.URI = "file://test//hello1.go"
	didOpenParam.TextDocument.Version = 0
	didOpenParam.TextDocument.LanguageID = "GO"
	didOpenParam.TextDocument.Text = "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/didOpen", didOpenParam, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/didOpen response: %s", pretty.Sprint(response))

	didChangeTextDocumentParams := DidChangeTextDocumentParams{}
	didChangeTextDocumentParams.TextDocument.URI = "file://test//hello1.go"
	didChangeTextDocumentParams.TextDocument.Version = 0
	didChangeTextDocumentParams.ContentChanges = make([]TextDocumentContentChangeEvent, 1)
	didChangeTextDocumentParams.ContentChanges[0].Text = "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/didChange", didChangeTextDocumentParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/didChange response: %s", pretty.Sprint(response))

	didSaveTextDocumentParams = DidSaveTextDocumentParams{}
	didSaveTextDocumentParams.TextDocument.URI = "file://test//hello1.go"
	data = "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	didSaveTextDocumentParams.Text = &data
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/didSave", didSaveTextDocumentParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/didSave: %s", pretty.Sprint(response))

	didCloseTextDocumentParams := DidCloseTextDocumentParams{}
	didCloseTextDocumentParams.TextDocument.URI = "file://test//hello1.go"
	response = make(map[string]interface{})
	err = client.Call(ctx, "textDocument/didClose", didCloseTextDocumentParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("textDocument/didClose: %s", pretty.Sprint(response))

	response = make(map[string]interface{})
	err = client.Call(ctx, "shutdown", nil, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	log.Infof("shutdown")

	response = make(map[string]interface{})
	err = client.Call(ctx, "exit", nil, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method failed. err: %s", err)
	}
	client.Close()
	log.Infof("exit")

	//referenceParams := ReferenceParams{}
	//referenceParams.TextDocument.URI = "file://test//test.go"
	//referenceParams.Position.Line = 6
	//referenceParams.Position.Character = 6
	//referenceParams.Context = ReferenceContext{IncludeDeclaration: true}
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/references", referenceParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("initialized: %s", pretty.Sprint(response))

}

type LSPHandler struct {
}

func (l *LSPHandler) Handle(context context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) {
	result := make(map[string]interface{})
	bytes := []byte(*request.Params)
	json.Unmarshal(bytes, &result)
	log.Infof("method:%s, message:%s", request.Method, pretty.Sprint(result["message"]))
}

type Logger struct {
}

func (l *Logger) Printf(format string, v ...interface{}) () {
	log.Infof(format, v...)
}
