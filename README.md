# Kafka-for-Dev

## This is a repository for Kafka for Developer cousrse

## Setup Kafka cluster

### 1. Start Zookeeper

```console
./bin/zookeeper-server-start.sh ./config/zookeeper.properties
```

### 2. Start Kafka Broker

```console
./bin/kafka-server-start.sh ./config/server.properties
```

---

## Zookeeper Shell

```console
./bin/zookeeper-shell.sh localhost:2181
```

### 1. List root tree

```console
ls /
```

### 2. List broker

```console
ls /brokers/ids
```

### 3. Broker detail

```console
get /brokers/ids/0
```

### 4. Cluster ID

```console
get /cluster/id
```

### 5. Controller

```console
get /controller
```


---

### Create Kafka Topic


```console
./bin/kafka-topics.sh --create --bootstrap-server localhost:9092 --topic perf-test --partitions 1
```

---


## KRAFT

```console
./bin/kafka-storage.sh random-uuid
./bin/kafka-storage.sh format -t <token> -c ./config/kraft/server.properties
./bin/kafka-server-start.sh ./config/kraft/server.properties

```

---

## Segment

```console
./bin/kafka-run-class.sh kafka.tools.DumpLogSegments --deep-iteration --print-data-log --files /tmp/kafka-logs/latest-product-price1-0/00000000000000000000.log
```

---

## ksqlDB

Create stream

```sql
CREATE STREAM riderLocations (profileId VARCHAR, latitude DOUBLE, longitude DOUBLE)
  WITH (kafka_topic='locations', value_format='json', partitions=1);
```

create materialize view

```sql
CREATE TABLE currentLocation AS
  SELECT profileId,
         LATEST_BY_OFFSET(latitude) AS la,
         LATEST_BY_OFFSET(longitude) AS lo
  FROM riderlocations
  GROUP BY profileId
  EMIT CHANGES;
```

```sql
CREATE TABLE ridersNearMountainView AS
  SELECT ROUND(GEO_DISTANCE(la, lo, 37.4133, -122.1162), -1) AS distanceInMiles,
         COLLECT_LIST(profileId) AS riders,
         COUNT(*) AS count
  FROM currentLocation
  GROUP BY ROUND(GEO_DISTANCE(la, lo, 37.4133, -122.1162), -1);
```

try to query

```sql
-- Mountain View lat, long: 37.4133, -122.1162
SELECT * FROM riderLocations
  WHERE GEO_DISTANCE(latitude, longitude, 37.4133, -122.1162) <= 5 EMIT CHANGES;
```

insert data

```sql
INSERT INTO riderLocations (profileId, latitude, longitude) VALUES ('c2309eec', 37.7877, -122.4205);
INSERT INTO riderLocations (profileId, latitude, longitude) VALUES ('18f4ea86', 37.3903, -122.0643);
INSERT INTO riderLocations (profileId, latitude, longitude) VALUES ('4ab5cbad', 37.3952, -122.0813);
INSERT INTO riderLocations (profileId, latitude, longitude) VALUES ('8b6eae59', 37.3944, -122.0813);
INSERT INTO riderLocations (profileId, latitude, longitude) VALUES ('4a7c7b41', 37.4049, -122.0822);
INSERT INTO riderLocations (profileId, latitude, longitude) VALUES ('4ddad000', 37.7857, -122.4011);
```

---

## Kafka SSL

### 1. Keystore

Generate keystore with keytool

```console
keytool -keystore aorjoa.keystore.jks -alias aorjoa.localhost -validity 365 -genkey -keyalg RSA -storetype pkcs12
```

Read keystore

```console
keytool -list -v -keystore aorjoa.keystore.jks -storepass 123456
```

### 2. Create root certificate authority (CA)

```bash
HOME            = .
RANDFILE        = $ENV::HOME/.rnd

####################################################################
[ ca ]
default_ca    = CA_default      # The default ca section

[ CA_default ]

base_dir      = .
certificate   = $base_dir/cacert.pem   # The CA certifcate
private_key   = $base_dir/cakey.pem    # The CA private key
new_certs_dir = $base_dir              # Location for new certs after signing
database      = $base_dir/index.txt    # Database index file
serial        = $base_dir/serial.txt   # The current serial number

default_days     = 1000         # How long to certify for
default_crl_days = 30           # How long before next CRL
default_md       = sha256       # Use public key default MD
preserve         = no           # Keep passed DN ordering

x509_extensions = ca_extensions # The extensions to add to the cert

email_in_dn     = no            # Don't concat the email in the DN
copy_extensions = copy          # Required to copy SANs from CSR to cert

####################################################################
[ req ]
default_bits       = 4096
default_keyfile    = cakey.pem
distinguished_name = ca_distinguished_name
x509_extensions    = ca_extensions
string_mask        = utf8only

####################################################################
[ ca_distinguished_name ]
countryName         = Country Name (2 letter code)
countryName_default = DE

stateOrProvinceName         = State or Province Name (full name)
stateOrProvinceName_default = Test Province

localityName                = Locality Name (eg, city)
localityName_default        = Test Town

organizationName            = Organization Name (eg, company)
organizationName_default    = Test Company

organizationalUnitName         = Organizational Unit (eg, division)
organizationalUnitName_default = Test Unit

commonName         = Common Name (e.g. server FQDN or YOUR name)
commonName_default = Test Name

emailAddress         = Email Address
emailAddress_default = test@test.com

####################################################################
[ ca_extensions ]

subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid:always, issuer
basicConstraints       = critical, CA:true
keyUsage               = keyCertSign, cRLSign

####################################################################
[ signing_policy ]
countryName            = optional
stateOrProvinceName    = optional
localityName           = optional
organizationName       = optional
organizationalUnitName = optional
commonName             = supplied
emailAddress           = optional

####################################################################
[ signing_req ]
subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid,issuer
basicConstraints       = CA:FALSE
keyUsage               = digitalSignature, keyEncipherment
```

Create keystore with imported CA certificate

```console
keytool -keystore aorjoa.truststore.jks -alias CARoot -import -file cacert.pem
```

Keep track series of certificate

```console
echo 01 > serial.txt
```

Keep track series of certificate

```console
touch index.txt
```

### 3. Signing certificate

Create certificate signing request (CSR)

```console
keytool -certreq -keystore aorjoa.keystore.jks -alias aorjoa.localhost -file aorjoa.unsigned.crt
```

Sign CSR

```console
openssl ca -config ssl-config.conf -policy signing_policy -extensions signing_req -out aorjoa.signed.crt -infiles aorjoa.unsigned.crt
```

Import root CA to keystore

```console
keytool -keystore aorjoa.truststore.jks -alias CARoot -import -file cacert.pem
```

Import signed certificate to keystore

```console
keytool -keystore aorjoa.truststore.jks -alias aorjoa.localhost -import -file aorjoa.signed.crt
```

### 4. Config broker

Allow broker to accecpt SSL incoming request at `config/server.properties`

```conf
listeners=PLAINTEXT://:9092,SSL://:29092
advertised.listeners=PLAINTEXT://:9092,SSL://:29092
```

Config keystore and truststore for authentication protocol

```conf
ssl.keystore.location=/Users/bhuridech.sudsee/Downloads/tmp/aorjoa.keystore.jks
ssl.keystore.password=123456
ssl.key.password=123456
ssl.truststore.location=/Users/bhuridech.sudsee/Downloads/tmp/aorjoa.truststore.jks
ssl.truststore.password=123456
```

Start Zookeeper and Kafka to check SSL is working

```console
openssl s_client -connect localhost:29092
```

### 5. Kafka client

Config

```conf
security.protocol=SSL
ssl.truststore.location=/Users/bhuridech.sudsee/Downloads/tmp/aorjoa.keystore.jks
ssl.truststore.password=123456
ssl.keystore.location=/Users/bhuridech.sudsee/Downloads/tmp/aorjoa.truststore.jks
ssl.keystore.password=123456
ssl.key.password=123456
```

and disable host check (in case of issued hostname does not aligned) with

```conf
ssl.endpoint.identification.algorithm=
```

Producer

```console
./bin/kafka-console-producer.sh  --bootstrap-server :29092  --topic test  --producer.config ./config/ssl-client.conf
```

Consumer

```console
./bin/kafka-console-consumer.sh  --bootstrap-server :29092  --topic test  --consumer.config ./config/ssl-client.conf
```

### 6. SASL/SCARM (SCRAM-SHA-246,SCRAM-SHA-512)

Config user

```console
./bin/kafka-configs.sh --bootstrap-server localhost:9092 --alter --add-config 'SCRAM-SHA-256=[password=alice-secret],SCRAM-SHA-512=[password=alice-secret]' --entity-type users --entity-name alice
```

config admin

```console
./bin/kafka-configs.sh --bootstrap-server localhost:9092 --alter --add-config 'SCRAM-SHA-256=[password=admin-secret],SCRAM-SHA-512=[password=admin-secret]' --entity-type users --entity-name admin
```

and enable authentication mechanism in broker config

```conf
sasl.enabled.mechanisms=SCRAM-SHA-256,SCRAM-SHA-512
```

create JAAS file (eg. config.jaas)

```conf
KafkaServer {
   org.apache.kafka.common.security.scram.ScramLoginModule required
   username="admin"
   password="admin-secret";
};
```

modify listener to support SASL_SSL

```conf
listeners=PLAINTEXT://:9092,SSL://:29092,SASL_SSL://:39092
advertised.listeners=PLAINTEXT://:9092,SSL://:29092,SASL_SSL://:39092

```

start server by passing `KAFKA_OPTS`

```console
KAFKA_OPTS=-Djava.security.auth.login.config=/tmp/config.jaas ./bin/kafka-server-start.sh ./config/ssl-server.properties
```

### 7. SASL/SCRAM Client

Config create JAAS for client (eg. client.jaas)

```conf
KafkaClient {
   org.apache.kafka.common.security.scram.ScramLoginModule required
   username="alice"
   password="alice-secret";
};
```

add following line to consumer config

```conf
sasl.mechanism=SCRAM-SHA-256
security.protocol=SASL_SSL
```

try to consume

```console
KAFKA_OPTS=-Djava.security.auth.login.config=/Users/bhuridech.sudsee/Downloads/kafka_2.12-3.1.0/client.jaas ./bin/kafka-console-consumer.sh  --bootstrap-server :39092  --topic test  --consumer.config ./config/ssl-consumer.conf
```

---

## Monitoring

Navigate to `kafka-grafana` then try to `docker-compose up` then you can now open web browser to check whether everything run just fine on Grafana

another important metric is `consumer group lag` (run after consumed with consumer group)

```console
./bin/kafka-consumer-groups.sh --bootstrap-server 178.128.85.19:9092 --all-groups --describe
```

---

## Performance metric

Create test topic

```console
./bin/kafka-topics.sh \
--create \
--topic test-perf \
--partitions 10 \
--replication-factor 1 \
--config min.insync.replicas=1 \
--bootstrap-server localhost:9092
```

Producer

```console
./bin/kafka-producer-perf-test.sh \
--topic test-perf \
--throughput -1 \
--num-records 3000000 \
--record-size 1024 \
--producer-props bootstrap.servers=localhost:9092
```

Consumer

```console
./bin/kafka-consumer-perf-test.sh \
--topic test-perf \
--broker-list localhost:9092 \
--messages 3000000
```
