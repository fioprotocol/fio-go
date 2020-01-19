package fio

// CompressOBT controls whether the obt payloads should be zlib compressed
// *before* encrypting. This combined with base64 encoding requires about
// 25% of the storage of uncompressed, hex-encoded strings.
//
// Old spec does not use zlib, and hex-encodes the strings , new is zlib + base64
// setting this to true will allow implementing the new behavior, which will hopefully
// be included in an upcoming release. It is exported so the behavior can be specified
// by the client.
var CompressOBT = false
