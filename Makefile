GOBIN?=$(shell go env GOBIN)
GOBIN:=$(if ${GOBIN}, ${GOBIN}, ${GOPATH}/bin)

.PHONY: init go_check go_tool_install go_test go_format go_init go_allcheck allcheck

init: go_init

go_check: go_static_check_tool_install
	go vet ./...
	errcheck -ignore="Close|Run|Write" ./...
	golint ./... | egrep -v 'Id.* should be .*ID|Url| should have comment | comment on exported ' | perl -e 'local $$/; $$o=<STDIN>; if ($$o eq "") {exit(0)}; print $$o; exit(1);'
	gocyclo -over 20 .
	unconvert -v . | perl -e 'local $$/; $$o=<STDIN>; if ($$o eq "") {exit(0)}; print $$o; exit(1);'
	staticcheck ./...
	ineffassign .
	nilerr ./...

STATIC_CHECK_TOOLS:=${GOBIN}/errcheck ${GOBIN}/golint ${GOBIN}/staticcheck ${GOBIN}/unconvert ${GOBIN}/ineffassign ${GOBIN}/gocyclo ${GOBIN}/nilerr
.PHONY: go_static_check_tool_install
go_static_check_tool_install: ${STATIC_CHECK_TOOLS}

${GOBIN}/errcheck:
	go build -o $@ github.com/kisielk/errcheck
${GOBIN}/golint:
	go build -o $@ golang.org/x/lint/golint
${GOBIN}/staticcheck:
	go build -o $@ honnef.co/go/tools/cmd/staticcheck
${GOBIN}/unconvert:
	go build -o $@ github.com/mdempsky/unconvert
${GOBIN}/ineffassign:
	go build -o $@ github.com/gordonklaus/ineffassign
${GOBIN}/gocyclo:
	go build -o $@ github.com/sigma/gocyclo
${GOBIN}/nilerr:
	go build -o $@ github.com/gostaticanalysis/nilerr/cmd/nilerr

go_test:
	go test ./...

go_format:
	goimports -w=true .

go_init: go_tool_install

go_allcheck: go_check go_test

allcheck: go_allcheck
