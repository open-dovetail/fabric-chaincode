MAKEFILE_THIS := $(lastword $(MAKEFILE_LIST))
SRC_PATH      := $(patsubst %/,%,$(dir $(abspath $(MAKEFILE_THIS))))
APP_FILE      := sample.json
APP_NAME      := sample_cc
CONTRACT      := sample-contract.json

SCRIPT_PATH   ?= $(SRC_PATH)/../scripts
MARBLE_HOME   := $(SRC_PATH)/../samples/marble
FAB_PATH      ?= $(SRC_PATH)/../../hyperledger/fabric-samples
CC_DEPLOY     := $(FAB_PATH)/chaincode

.PHONY: all
all: build deploy

.PHONY: clean
clean:
	rm -Rf $(SRC_PATH)/$(APP_NAME)

.PHONY: build
build: $(CONTRACT) clean
	flogo contract2flow -c $(CONTRACT) -o $(APP_FILE)
	cp -R $(MARBLE_HOME)/META-INF $(SRC_PATH)
	$(SCRIPT_PATH)/build.sh $(APP_FILE) $(APP_NAME)

.PHONY: deploy
deploy: $(APP_NAME)_1.0.tar.gz
	cp $(APP_NAME)_1.0.tar.gz $(CC_DEPLOY)
	cp cc-init.sh $(FAB_PATH)/test-network/cc-init-sample.sh
	cp cc-test.sh $(FAB_PATH)/test-network/cc-test-sample.sh

.PHONY: start
start:
	cd $(FAB_PATH)/test-network && ./network.sh up createChannel -ca -s couchdb
	$(MARBLE_HOME)/gen-certs.sh

.PHONY: cc-init
cc-init: $(FAB_PATH)/test-network/cc-init-sample.sh
	docker exec cli bash -c './cc-init-sample.sh'

.PHONY: cc-test
cc-test: $(FAB_PATH)/test-network/cc-test-sample.sh
	docker exec cli bash -c './cc-test-sample.sh'

.PHONY: shutdown
shutdown:
	cd $(FAB_PATH)/test-network && ./network.sh down
