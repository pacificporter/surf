.PHONY: init go_check go_tool_install go_test go_format go_init go_allcheck allcheck

NOVENDOR:=$(shell glide novendor)
NOVENDORX:=$(shell glide novendor -x)

init: go_init

go_check:
	go vet ${NOVENDOR}
	errcheck -ignore="Close|Run|Write" ${NOVENDOR}
	echo -n ${NOVENDOR} | xargs -d ' ' -L1 golint  | egrep -v 'Id.* should be .*ID|Url| should have comment | comment on exported ' | perl -e 'local $$/; $$o=<STDIN>; if ($$o eq "") {exit(0)}; print $$o; exit(1);'
	gocyclo -over 20 ${NOVENDORX}
	unconvert -v ${NOVENDORX} | perl -e 'local $$/; $$o=<STDIN>; if ($$o eq "") {exit(0)}; print $$o; exit(1);'
	gosimple ${NOVENDOR}
	ineffassign ${NOVENDORX}

go_tool_install:
	go get -u golang.org/x/tools/cmd/vet
	go get -u github.com/kisielk/errcheck
	go get -u github.com/golang/lint/golint
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/tcnksm/gotests
	go get -u github.com/Masterminds/glide
	go get -u github.com/mdempsky/unconvert
	go get -u github.com/sigma/gocyclo
	go get -u github.com/gordonklaus/ineffassign

go_test:
	go test ${NOVENDOR}

go_format:
	goimports -w=true .

go_init: go_tool_install

go_allcheck: go_check go_test

allcheck: go_allcheck
