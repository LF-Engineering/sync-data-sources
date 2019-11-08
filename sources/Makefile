GO_LIB_FILES=context.go error.go const.go log.go time.go exec.go threads.go
GO_BIN_FILES=cmd/syncdatasources/syncdatasources.go
GO_TEST_FILES=context_test.go time_test.go threads_test.go
GO_LIBTEST_FILES=test/time.go
GO_BIN_CMDS=github.com/LF-Engineering/sync-data-sources/sources/cmd/syncdatasources
GO_ENV=CGO_ENABLED=0
GO_BUILD=go build -ldflags '-s -w'
GO_INSTALL=go install -ldflags '-s'
GO_FMT=gofmt -s -w
GO_LINT=golint -set_exit_status
GO_VET=go vet
GO_CONST=goconst
GO_IMPORTS=goimports -w
GO_USEDEXPORTS=usedexports
GO_ERRCHECK=errcheck -asserts -ignore '[FS]?[Pp]rint*'
GO_TEST=go test
BINARIES=syncdatasources
STRIP=strip

all: check ${BINARIES}

syncdatasources: cmd/syncdatasources/syncdatasources.go ${GO_LIB_FILES}
	 ${GO_ENV} ${GO_BUILD} -o syncdatasources cmd/syncdatasources/syncdatasources.go

fmt: ${GO_BIN_FILES} ${GO_LIB_FILES} ${GO_TEST_FILES} ${GO_LIBTEST_FILES}
	./for_each_go_file.sh "${GO_FMT}"

lint: ${GO_BIN_FILES} ${GO_LIB_FILES} ${GO_TEST_FILES} ${GO_LIBTEST_FILES}
	./for_each_go_file.sh "${GO_LINT}"

vet: ${GO_BIN_FILES} ${GO_LIB_FILES} ${GO_TEST_FILES} ${GO_LIBTEST_FILES}
	./vet_files.sh "${GO_VET}"

imports: ${GO_BIN_FILES} ${GO_LIB_FILES} ${GO_TEST_FILES} ${GO_LIBTEST_FILES}
	./for_each_go_file.sh "${GO_IMPORTS}"

const: ${GO_BIN_FILES} ${GO_LIB_FILES} ${GO_TEST_FILES} ${GO_LIBTEST_FILES}
	${GO_CONST} ./...

usedexports: ${GO_BIN_FILES} ${GO_LIB_FILES} ${GO_TEST_FILES} ${GO_LIBTEST_FILES}
	${GO_USEDEXPORTS} ./...

errcheck: ${GO_BIN_FILES} ${GO_LIB_FILES} ${GO_TEST_FILES} ${GO_LIBTEST_FILES}
	${GO_ERRCHECK} ./...

test:
	${GO_TEST} ${GO_TEST_FILES}

check: fmt lint imports vet const usedexports errcheck

install: check ${BINARIES}
	${GO_INSTALL} ${GO_BIN_CMDS}

dockerinstall: ${BINARIES}
	${GO_INSTALL} ${GO_BIN_CMDS}

strip: ${BINARIES}
	${STRIP} ${BINARIES}

clean:
	rm -f ${BINARIES}

.PHONY: test
