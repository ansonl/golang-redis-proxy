Golang Redis proxy
===================

A SET and GET Redis proxy. 

SET

`&key=book&value=library`

GET

`&key=book`

`value` data is expected to be base64 encoded.
`value` data returned is base64 encoded.

Set $REDIS_URL to your Redis instance URL.