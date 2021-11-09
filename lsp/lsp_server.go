package main

import (
	"context"
	"encoding/json"
	"github.com/kr/pretty"
	"github.com/sourcegraph/jsonrpc2"
	"lsp/logger"
	"lsp/protocol"
	"net"
	"sync"
)

var log = logger.Get()

type ServerConfig struct {
	NetWork string
	Address string
}

type LanguageServer struct {
	initialized bool
	started     bool

	mutex        sync.Mutex
	ctx          context.Context
	conn         net.Conn
	rpcConn      *jsonrpc2.Conn
	serverConfig ServerConfig
}

func InitLanguageServer(ctx context.Context, config ServerConfig) *LanguageServer {
	server := LanguageServer{}
	server.ctx = ctx
	server.serverConfig.NetWork = config.NetWork
	server.serverConfig.Address = config.Address
	server.initialized = true
	return &server
}

func (lsp *LanguageServer) Start() {
	lsp.mutex.Lock()
	defer lsp.mutex.Unlock()
	if lsp.started {
		log.Infof("LanguageServer start server, server have started")
		return
	}

	lsp.fatalfIfNotInit()

	log.Infof("LanguageServer start server")
	conn, err := net.Dial(lsp.serverConfig.NetWork, lsp.serverConfig.Address)
	if err != nil {
		log.Fatalf("LanguageServer net dial failed. network:%s, address:%s", lsp.serverConfig.NetWork, lsp.serverConfig.Address)
	}
	lsp.conn = conn

	stream := jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{})
	client := jsonrpc2.NewConn(lsp.ctx, stream, &LSPHandler{}, jsonrpc2.LogMessages(&Logger{}))
	lsp.rpcConn = client

	go lsp.serverListenerLoop()

	lsp.started = true

	log.Infof("LanguageServer start success")
}

func (lsp *LanguageServer) serverListenerLoop() {
	for {
		select {
		case <-lsp.ctx.Done():
			log.Infof("lsp context done")
			lsp.Shutdown()
		}
	}
}

func (lsp *LanguageServer) Shutdown() {
	err := lsp.rpcConn.Close()
	if err != nil {
		log.Warnf("LanguageServer Shutdown close rpc connection failed. err:%s", err)
	}
}

func (lsp *LanguageServer) InitWorkSpace(name, uri string) {
	log.Infof("LanguageServer InitWorkSpace start. name:%s, uri:%s", name, uri)

	initializeParams := protocol.InitializeParams{}
	initializeParams.ClientInfo.Name = "test"
	initializeParams.ClientInfo.Version = "v1.0.0"
	initializeParams.WorkspaceFolders = []protocol.WorkspaceFolder{{Name: name, URI: uri}}

	initializeResult := protocol.InitializeResult{}
	err := lsp.rpcConn.Call(lsp.ctx, "initialize", initializeParams, &initializeResult, nil)
	if err != nil {
		log.Errorf("InitWorkSpace call json rpc method `initialize` failed. err: %s", err)
	}

	log.Infof("InitWorkSpace initialize response: %s", pretty.Sprint(initializeResult))

	err = lsp.rpcConn.Call(lsp.ctx, "initialized", protocol.InitializedParams{}, nil, nil)
	if err != nil {
		log.Errorf("InitWorkSpace call json rpc method `initialized` failed. err: %s", err)
	}
	log.Infof("LanguageServer InitWorkSpace success")
}

func (lsp *LanguageServer) DidOpenTextDocument() {
	log.Infof("DidOpenTextDocument start")
	didOpenParam := protocol.DidOpenTextDocumentParams{}
	didOpenParam.TextDocument.URI = "file://test//hello.go"
	didOpenParam.TextDocument.Version = 0
	didOpenParam.TextDocument.LanguageID = "go"
	didOpenParam.TextDocument.Text = "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello Ide!\")\n}\n"
	err := lsp.rpcConn.Call(lsp.ctx, "textDocument/didOpen", didOpenParam, nil, nil)
	if err != nil {
		log.Errorf("DidOpenTextDocument call json rpc method [textDocument/didOpen] failed. err: %s", err)
	}

	didOpenParam = protocol.DidOpenTextDocumentParams{}
	didOpenParam.TextDocument.URI = "file://test//go.mod"
	didOpenParam.TextDocument.Version = 0
	didOpenParam.TextDocument.LanguageID = "go"
	didOpenParam.TextDocument.Text = "module hello\\n\\ngo 1.16\\n"
	err = lsp.rpcConn.Call(lsp.ctx, "textDocument/didOpen", didOpenParam, nil, nil)
	if err != nil {
		log.Errorf("DidOpenTextDocument call json rpc method [textDocument/didOpen] failed. err: %s", err)
	}
	log.Infof("DidOpenTextDocument success")
}

func (lsp *LanguageServer) DidSaveTextDocument() {
	log.Infof("DidSaveTextDocument start")
	didSaveParam := protocol.DidSaveTextDocumentParams{}
	didSaveParam.TextDocument.URI = "file://test//hello.go"
	data := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello Ide!\")\n}\n"
	didSaveParam.Text = &data
	err := lsp.rpcConn.Call(lsp.ctx, "textDocument/didSave", didSaveParam, nil, nil)
	if err != nil {
		log.Errorf("DidSaveTextDocument call json rpc method [textDocument/didOpen] failed. err: %s", err)
	}

	didSaveParam = protocol.DidSaveTextDocumentParams{}
	didSaveParam.TextDocument.URI = "file://test//hello.go"
	data = "module hello\\n\\ngo 1.16\\n"
	didSaveParam.Text = &data
	err = lsp.rpcConn.Call(lsp.ctx, "textDocument/didSave", didSaveParam, nil, nil)
	if err != nil {
		log.Errorf("DidSaveTextDocument call json rpc method [textDocument/didOpen] failed. err: %s", err)
	}
	log.Infof("DidSaveTextDocument success")
}

func (lsp *LanguageServer) ExecuteGoModTidy() {
	log.Infof("ExecuteGoModTidy start")

	executeParams := protocol.ExecuteCommandParams{}
	executeParams.Command = "gopls.tidy"
	executeParams.Arguments = []json.RawMessage{[]byte(`{"URI":"file://test"}`)}
	executeParams.WorkDoneToken = "11111111111111111"
	response := make(map[string]interface{})
	//executeParams.Arguments = []json.RawMessage{[]byte("[version]")}
	err := lsp.rpcConn.Call(lsp.ctx, "workspace/executeCommand", &executeParams, &response, nil)
	if err != nil {
		log.Errorf("call json rpc method [workspace/executeCommand gopls.tidy] failed. err: %s", err)
	}

	log.Infof("ExecuteGoModTidy success: %s", pretty.Sprint(response))
}

func (lsp *LanguageServer) fatalfIfNotInit() {
	if !lsp.initialized {
		log.Fatalf("start language server failed. please init first")
	}
}

func main() {
	ctx := context.Background()
	languageServer := InitLanguageServer(ctx, ServerConfig{NetWork: "tcp", Address: "127.0.0.1:9877"})
	languageServer.Start()
	languageServer.InitWorkSpace("test", "file:///test")
	languageServer.DidOpenTextDocument()
	languageServer.ExecuteGoModTidy()
	languageServer.Shutdown()

	////在工作空间执行命令
	//executeParams1 := ExecuteCommandParams{}
	//executeParams1.Command = "gopls.tidy"
	//executeParams1.Arguments = []json.RawMessage{[]byte(`{"URI":"file://test"}`)}
	//executeParams1.WorkDoneToken = "11111111111111111"
	//response = make(map[string]interface{})
	////executeParams.Arguments = []json.RawMessage{[]byte("[version]")}
	//err = client.Call(ctx, "workspace/executeCommand", &executeParams1, &response, nil)
	//if err != nil {
	//	log.Errorf("gopls.tidy call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/executeCommand gopls.tidy: %s", pretty.Sprint(response))
	//
	////在工作空间执行命令
	//executeParams2 := ExecuteCommandParams{}
	//executeParams2.Command = "gopls.vendor"
	//executeParams2.Arguments = []json.RawMessage{[]byte(`{"URI":"file://test/"}`)}
	//executeParams2.WorkDoneToken = "xxxxxxxxxxxxxxxx"
	//response = make(map[string]interface{})
	////executeParams.Arguments = []json.RawMessage{[]byte("[version]")}
	//err = client.Call(ctx, "workspace/executeCommand", &executeParams2, &response, nil)
	//if err != nil {
	//	log.Errorf("gopls.vendor call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/executeCommand gopls.vendor: %s", pretty.Sprint(response))
	//
	//filesParams := DidChangeWatchedFilesParams{}
	//filesParams.Changes = []FileEvent{{URI: "file://test//go.mod", Type: 6}, {URI: "file://test//hello.go", Type: 6}}
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "workspace/didChangeWatchedFiles", filesParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/didChangeWatchedFiles response: %s", pretty.Sprint(response))
	//
	//didOpenParam := DidOpenTextDocumentParams{}
	//didOpenParam.TextDocument.URI = "file://test//hello.go"
	//didOpenParam.TextDocument.Version = 0
	//didOpenParam.TextDocument.LanguageID = "go"
	//didOpenParam.TextDocument.Text = "package main\n\nimport \"fmt\"\nimport \"github.com/sourcegraph/jsonrpc2\"\n\nfunc main() {\n\tresult := jsonrpc2.NewBufferedStream(nil, nil)\n\tfmt.Println(result)\n\tfmt.Println()\n}\n"
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didOpen", didOpenParam, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didOpen response: %s", pretty.Sprint(response))
	//
	//didSaveTextDocumentParams := DidSaveTextDocumentParams{}
	//didSaveTextDocumentParams.TextDocument.URI = "file://test//hello.go"
	//data := "package main\n\nimport \"fmt\"\nimport \"github.com/sourcegraph/jsonrpc2\"\n\nfunc main() {\n\tresult := jsonrpc2.NewBufferedStream(nil, nil)\n\tfmt.Println(result)\n\tfmt.Println()\n}\n"
	//didSaveTextDocumentParams.Text = &data
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didSave", &didSaveTextDocumentParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didSave: %s", pretty.Sprint(response))
	//
	//didOpenParam = DidOpenTextDocumentParams{}
	//didOpenParam.TextDocument.URI = "file://test//go.mod"
	//didOpenParam.TextDocument.Version = 0
	//didOpenParam.TextDocument.LanguageID = "go"
	//didOpenParam.TextDocument.Text = "module hello\n\ngo 1.16\n"
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didOpen", didOpenParam, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didOpen response: %s", pretty.Sprint(response))
	//
	//didSaveTextDocumentParams = DidSaveTextDocumentParams{}
	//didSaveTextDocumentParams.TextDocument.URI = "file://test//go.mod"
	//data = "module hello\n\ngo 1.16\n"
	//didSaveTextDocumentParams.Text = &data
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didSave", &didSaveTextDocumentParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didSave: %s", pretty.Sprint(response))
	//
	//completionParams := CompletionParams{}
	//completionParams.WorkDoneToken = "333333333333333333333"
	//completionParams.TextDocument.URI = "file://test//hello.go"
	//completionParams.Position.Line = 1
	//completionParams.Position.Character = 4
	//
	////自动补全
	//log.Infof("textDocument/completion request: %s", pretty.Sprint(completionParams))
	//completionList := CompletionList{}
	//err = client.Call(ctx, "textDocument/completion", &completionParams, &completionList, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/completion: %s", pretty.Sprint(response))
	//
	////悬停信息
	//hoverParams := HoverParams{}
	//hoverParams.TextDocument.URI = "file://test//hello.go"
	//hoverParams.Position.Line = 8
	//hoverParams.Position.Character = 10
	//hoverParams.WorkDoneToken = "44444444444444"
	//response = make(map[string]interface{})
	//log.Infof("textDocument/hover request: %s", pretty.Sprint(hoverParams))
	//err = client.Call(ctx, "textDocument/hover", &hoverParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/hover: %s", pretty.Sprint(response))
	//
	////签名信息
	//signatureHelpParams := SignatureHelpParams{}
	//signatureHelpParams.TextDocument.URI = "file://test//hello.go"
	//signatureHelpParams.Position.Line = 9
	//signatureHelpParams.Position.Character = 17
	//signatureHelpParams.WorkDoneToken = "5555555555555"
	//response = make(map[string]interface{})
	//log.Infof("textDocument/signatureHelp request: %s", pretty.Sprint(signatureHelpParams))
	//err = client.Call(ctx, "textDocument/signatureHelp", &signatureHelpParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/signatureHelp: %s", pretty.Sprint(response))
	//
	//////声明位置
	////definitionParams := DefinitionParams{}
	////definitionParams.TextDocument.URI = "file://test//hello.go"
	////definitionParams.Position.Line = 9
	////definitionParams.Position.Character = 12
	////definitionParams.WorkDoneToken = "6666666666666666666"
	////response = make(map[string]interface{})
	////log.Infof("textDocument/definition request: %s", pretty.Sprint(definitionParams))
	////err = client.Call(ctx, "textDocument/definition", &definitionParams, &response, nil)
	////if err != nil {
	////	log.Errorf("call json rpc method failed. err: %s", err)
	////}
	////log.Infof("textDocument/definition: %s", pretty.Sprint(response))
	//
	////查找引用
	//referenceParams := ReferenceParams{}
	//referenceParams.TextDocument.URI = "file://test//hello.go"
	//referenceParams.Position.Line = 7
	//referenceParams.Position.Character = 5
	//referenceParams.WorkDoneToken = "77777777777777777777"
	//responseLocation := make([]Location, 0)
	//log.Info("textDocument/references request: %s", pretty.Sprint(referenceParams))
	//err = client.Call(ctx, "textDocument/references", &referenceParams, &responseLocation, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/references: %s", pretty.Sprint(responseLocation))
	//
	////代码高亮
	//documentHighlightParams := DocumentHighlightParams{}
	//documentHighlightParams.TextDocument.URI = "file://test//hello.go"
	//documentHighlightParams.Position.Line = 7
	//documentHighlightParams.Position.Character = 5
	//documentHighlightParams.WorkDoneToken = "888888888888888"
	//documentHighlightResponse := make([]DocumentHighlight, 0)
	//log.Info("textDocument/documentHighlight request: %s", pretty.Sprint(documentHighlightParams))
	//err = client.Call(ctx, "textDocument/documentHighlight", &referenceParams, &documentHighlightResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/documentHighlight: %s", pretty.Sprint(documentHighlightResponse))
	//
	////文档符号
	//documentSymbolParams := DocumentSymbolParams{}
	//documentSymbolParams.TextDocument.URI = "file://test//hello.go"
	//documentSymbolParams.WorkDoneToken = "99999999999999999999"
	//documentSymbolResponse := make([]DocumentSymbol, 0)
	//log.Info("textDocument/documentSymbol request: %s", pretty.Sprint(documentSymbolParams))
	//err = client.Call(ctx, "textDocument/documentSymbol", &documentSymbolParams, &documentSymbolResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/documentSymbol: %s", pretty.Sprint(documentSymbolResponse))
	//
	////可以执行的优化命令
	//codeActionParams := CodeActionParams{}
	//codeActionParams.TextDocument.URI = "file://test//hello.go"
	//codeActionParams.WorkDoneToken = "1010101010101010"
	//codeActionParams.Range.Start.Line = 7
	//codeActionParams.Range.Start.Character = 0
	//codeActionParams.Range.End.Line = 8
	//codeActionParams.Range.End.Line = 0
	//codeActionResponse := make([]CodeAction, 0)
	//log.Info("textDocument/codeAction request: %s", pretty.Sprint(codeActionParams))
	//err = client.Call(ctx, "textDocument/codeAction", &codeActionParams, &codeActionResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/codeAction: %s", pretty.Sprint(codeActionResponse))
	//
	////代码信息显示
	//codeLensParams := CodeLensParams{}
	//codeLensParams.TextDocument.URI = "file://test//hello.go"
	//codeLensParams.WorkDoneToken = "1212121212121212"
	//codeLensResponse := CodeLens{}
	//log.Info("textDocument/codeLens: %s", pretty.Sprint(codeLensParams))
	//err = client.Call(ctx, "textDocument/codeLens", &codeLensParams, &codeLensResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/codeLens: %s", pretty.Sprint(codeLensResponse))
	//
	////文档颜色
	//documentColorParams := DocumentColorParams{}
	//documentColorParams.TextDocument.URI = "file://test//hello.go"
	//documentColorParams.WorkDoneToken = "1313131313131313131313"
	//colorInformation := ColorInformation{}
	//log.Info("textDocument/documentColor: %s", pretty.Sprint(documentColorParams))
	//err = client.Call(ctx, "textDocument/documentColor", &documentColorParams, &colorInformation, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/documentColor: %s", pretty.Sprint(colorInformation))
	//
	////重命名
	//renameParams := RenameParams{}
	//renameParams.TextDocument.URI = "file://test//hello.go"
	//renameParams.Position.Line = 6
	//renameParams.Position.Character = 3
	//renameParams.NewName = "aa"
	//renameParams.WorkDoneToken = "1414141414141441411441"
	//renameResponse := WorkspaceEdit{}
	//log.Info("textDocument/rename: %s", pretty.Sprint(renameParams))
	//err = client.Call(ctx, "textDocument/rename", &renameParams, &renameResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/rename: %s", pretty.Sprint(renameResponse))
	//
	////在工作空间执行命令
	//executeParams := ExecuteCommandParams{}
	//executeParams.Command = "gopls.list_known_packages"
	//executeParams.WorkDoneToken = "2222222222222222"
	//executeParams.Arguments = []json.RawMessage{[]byte(`{"URI":"file://test/hello1.go"}`)}
	//response = make(map[string]interface{})
	////executeParams.Arguments = []json.RawMessage{[]byte("[version]")}
	//err = client.Call(ctx, "workspace/executeCommand", &executeParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/executeCommand: %s", pretty.Sprint(response))
	//
	////在工作空间查询符号
	//params := WorkspaceSymbolParams{}
	//params.Query = "math"
	//symbolResponse := make([]interface{}, 0)
	//err = client.Call(ctx, "workspace/symbol", &params, &symbolResponse, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("workspace/symbol: %s", pretty.Sprint(symbolResponse))
	//
	////todo not implement
	////createFilesParams := CreateFilesParams{}
	////createFilesParams.Files = []FileCreate{{URI: "file://hello1.go"}}
	////err = client.Call(ctx, "workspace/didCreateFiles", &createFilesParams, &response, nil)
	////if err != nil {
	////	log.Errorf("call json rpc method failed. err: %s", err)
	////}
	////log.Infof("workspace/didCreateFiles: %s", pretty.Sprint(response))
	//
	//didOpenParam = DidOpenTextDocumentParams{}
	//didOpenParam.TextDocument.URI = "file://test//hello1.go"
	//didOpenParam.TextDocument.Version = 0
	//didOpenParam.TextDocument.LanguageID = "GO"
	//didOpenParam.TextDocument.Text = "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didOpen", didOpenParam, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didOpen response: %s", pretty.Sprint(response))
	//
	//didChangeTextDocumentParams := DidChangeTextDocumentParams{}
	//didChangeTextDocumentParams.TextDocument.URI = "file://test//hello1.go"
	//didChangeTextDocumentParams.TextDocument.Version = 0
	//didChangeTextDocumentParams.ContentChanges = make([]TextDocumentContentChangeEvent, 1)
	//didChangeTextDocumentParams.ContentChanges[0].Text = "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didChange", didChangeTextDocumentParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didChange response: %s", pretty.Sprint(response))
	//
	//didSaveTextDocumentParams = DidSaveTextDocumentParams{}
	//didSaveTextDocumentParams.TextDocument.URI = "file://test//hello1.go"
	//data = "package main\n\nimport \"fmt\"\n\nfunc main(){\n    fmt.Println(\"Hello\")\n}"
	//didSaveTextDocumentParams.Text = &data
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didSave", didSaveTextDocumentParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didSave: %s", pretty.Sprint(response))
	//
	//didCloseTextDocumentParams := DidCloseTextDocumentParams{}
	//didCloseTextDocumentParams.TextDocument.URI = "file://test//hello1.go"
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "textDocument/didClose", didCloseTextDocumentParams, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("textDocument/didClose: %s", pretty.Sprint(response))
	//
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "shutdown", nil, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//log.Infof("shutdown")
	//
	//response = make(map[string]interface{})
	//err = client.Call(ctx, "exit", nil, &response, nil)
	//if err != nil {
	//	log.Errorf("call json rpc method failed. err: %s", err)
	//}
	//client.Close()
	//log.Infof("exit")
	//
	////referenceParams := ReferenceParams{}
	////referenceParams.TextDocument.URI = "file://test//test.go"
	////referenceParams.Position.Line = 6
	////referenceParams.Position.Character = 6
	////referenceParams.Context = ReferenceContext{IncludeDeclaration: true}
	////response = make(map[string]interface{})
	////err = client.Call(ctx, "textDocument/references", referenceParams, &response, nil)
	////if err != nil {
	////	log.Errorf("call json rpc method failed. err: %s", err)
	////}
	////log.Infof("initialized: %s", pretty.Sprint(response))

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
