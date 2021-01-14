#
# Copyright (c) 2020, TIBCO Software Inc.
# All rights reserved.
#
# SPDX-License-Identifier: BSD-3-Clause-Open-MPI
#
MAKEFILE_THIS := $(lastword $(MAKEFILE_LIST))
SRC_PATH      := $(patsubst %/,%,$(dir $(abspath $(MAKEFILE_THIS))))
APP_FILE      := marble.json
APP_NAME      := marble_cc

SCRIPT_PATH   ?= $(SRC_PATH)/../../scripts
FAB_PATH      ?= $(SRC_PATH)/../../../hyperledger/fabric-samples
CC_DEPLOY     := $(FAB_PATH)/chaincode

.PHONY: all
all: build deploy

.PHONY: clean
clean:
	rm -Rf $(SRC_PATH)/$(APP_NAME)

.PHONY: build
build: $(APP_FILE) clean
	$(SCRIPT_PATH)/build.sh $(APP_FILE) $(APP_NAME)

.PHONY: deploy
deploy: $(APP_NAME)_1.0.tar.gz
	cp $(APP_NAME)_1.0.tar.gz $(CC_DEPLOY)
	cp cc-init.sh $(FAB_PATH)/test-network/cc-init-marble.sh
	cp cc-test.sh $(FAB_PATH)/test-network/cc-test-marble.sh

.PHONY: start
start:
	cd $(FAB_PATH)/test-network && ./network.sh up createChannel -ca -s couchdb
	$(SRC_PATH)/gen-certs.sh

.PHONY: cc-init
cc-init: $(FAB_PATH)/test-network/cc-init-marble.sh
	docker exec cli bash -c './cc-init-marble.sh'

.PHONY: cc-test
cc-test: $(FAB_PATH)/test-network/cc-test-marble.sh
	docker exec cli bash -c './cc-test-marble.sh'

.PHONY: shutdown
shutdown:
	cd $(FAB_PATH)/test-network && ./network.sh down
