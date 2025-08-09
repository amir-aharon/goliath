tcp server :6379
handles lines of messages using goroutines
GET, SET, DEL

persistence:
log every executed command
load snapshot from log

atomic on log file + dict

pubsub:
topics hold groups of channels
SUBSCRIBE topic
PUBLISH topic message

key expiration:
cleaner runs in background
another expiration by key dict
EXPIRE key duration
check expiration on GET
