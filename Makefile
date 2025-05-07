BINARY_NAME	:=	synctags
SOURCE_NAME	:=	"synctags.go qualys_client.go tags.go crowdstrike_client.go"
SOURCE_FILES 	:=	$(shell echo ${SOURCE_NAME} | tr -d '"')

deploy: scan lint build

test:
	@echo $(SOURCE_FILES)
scan:
	staticcheck ${SOURCE_FILES}

lint:
	golangci-lint run ${SOURCE_FILES}

build:
	go build -o ${BINARY_NAME} ${SOURCE_FILES}
	cp ${BINARY_NAME} ${HOME}/bin
 
run:
	go run -o ${BINARY_NAME} ${SOURCE_FILES}
 
clean:
	go clean
	rm ${BINARY_NAME}
	rm ~/bin/${BINARY_NAME}
