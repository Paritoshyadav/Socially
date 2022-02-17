Run cockroachDb:
example := cockroach start-single-node --insecure --host 127.0.0.1

create tables in db :
cat schema.sql | cockroach sql --insecure

Run socialnetwork.exe
or
Run go build