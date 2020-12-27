#!/bin/bash
# generate user certificates after starting test-network with CA, i.e.,
# ./network.sh up createChannel -ca -s couchdb
# ./gen-certs.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"; echo "$(pwd)")"
WORK=/tmp/ca

if [ -z "${FAB_PATH}" ]; then
  FAB_PATH=${SCRIPT_DIR}/../../../hyperledger/fabric-samples
fi


# rename user cert.pem with org prefix for fabric-sdk-go to work
function renameCerts {
  local certs=$(find organizations/peerOrganizations -name cert.pem | grep users)
  for f in ${certs}; do
    echo "rename ${f}"
    local token=${f##*/users/}
    token=${token%%/*}
    local newfile=${f/cert.pem/${token}-cert.pem}
    echo "new file ${newfile}"
    mv ${f} ${newfile}
  done
}

# genUser <user> <role> <org>
function genUser {
  local org_name=org1
  local port=7054
  local user=$1
  local role=$2
  if [ "$3" == "org2" ]; then
    org_name=org2
    port=8054
  fi
  local org=${org_name}.example.com
  local caname=ca-${org_name}
  local tlsca=${FAB_PATH}/test-network/organizations/fabric-ca/${org_name}/tls-cert.pem

  echo "generate key and cert for user ${user}@${org} with role ${role}"
  if [ -d "${WORK}" ]; then
    echo "cleanup ${WORK}"
    rm -R "${WORK}"
  fi

  # enroll CA admin
  export FABRIC_CA_CLIENT_HOME=${WORK}/admin
  ${FAB_PATH}/bin/fabric-ca-client getcainfo -u https://localhost:${port} --caname ${caname} --tls.certfiles ${tlsca}
  # openssl x509 -noout -text -in ${FABRIC_CA_CLIENT_HOME}/msp/cacerts/localhost-${port}.pem
  ${FAB_PATH}/bin/fabric-ca-client enroll -u https://admin:adminpw@localhost:${port} --caname ${caname} --tls.certfiles ${tlsca}

  # register and enroll new user
  # Note: important to make id.name as user@org for signature verification!
  ${FAB_PATH}/bin/fabric-ca-client register --caname ${caname} --tls.certfiles ${tlsca} --id.name ''"${user}@${org}"'' --id.secret ${user}pw --id.type client --id.attrs 'role='"${role}"':ecert,alias='"${user}"',email='"${user}@${org}"''
  export FABRIC_CA_CLIENT_HOME=${WORK}/${user}\@${org}
  ${FAB_PATH}/bin/fabric-ca-client enroll -u https://${user}@${org}:${user}pw@localhost:${port} --caname ${caname} --tls.certfiles ${tlsca} --enrollment.attrs "role,alias,email" -M ${FABRIC_CA_CLIENT_HOME}/msp
  # openssl x509 -noout -text -in ${WORK}/${user}\@${org}/msp/signcerts/cert.pem

  # copy key and cert to test-network orgainizations
  cd ${FAB_PATH}/test-network/organizations/peerOrganizations/${org}/users
  if [ -d "${user}@${org}" ]; then
    echo "remove old crypto ${user}@${org}"
    rm -Rf ${user}\@${org}
  fi
  cp -R User1\@${org} ${user}\@${org}
  cd ${user}\@${org}
  rm -R msp/keystore
  cp -R ${WORK}/${user}\@${org}/msp/keystore msp
  rm msp/signcerts/User1\@${org}-cert.pem
  cp ${WORK}/${user}\@${org}/msp/signcerts/cert.pem msp/signcerts/${user}\@${org}-cert.pem
  openssl x509 -noout -text -in msp/signcerts/${user}\@${org}-cert.pem
}

cd ${FAB_PATH}/test-network

# check CA server
for p in 7054 8054; do
  docker ps | grep "hyperledger/fabric-ca" | grep "${p}->${p}/tcp"
  if [ "$?" -ne 0 ]; then
    echo "CA server not running on port ${p}.  Start test network with '-ca' option, e.g., './network.sh up createChannel -ca -s couchdb'."
    exit 1
  fi
done

# check fabric-ca-client
if [ ! -f "${FAB_PATH}/bin/fabric-ca-client" ]; then
  echo "fabric-ca-client not found in ${FAB_PATH}/bin"
  exit 1
fi

# must rename user cert.pem with org prefix for fabric-sdk-go to work
renameCerts

# generate marble users for cid-based rules
genUser broker broker org1
genUser tom owner org1
genUser jerry owner org2
