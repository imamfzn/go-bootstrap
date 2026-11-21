// Development tools, kept out of go.mod so their dependencies do not become
// dependencies of the service. Run one with:
//
//	go tool -modfile=tools.mod goose
//
// Add one with:
//
//	go get -tool -modfile=tools.mod example.com/cmd/tool@latest
module github.com/imamfzn/go-bootstrap

go 1.27
