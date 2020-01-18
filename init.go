package fio

// Old spec uses no zlib + hex string encoding, new is zlib + base64
// setting this to false will allow overriding the new behavior
var CompressOBT = true
