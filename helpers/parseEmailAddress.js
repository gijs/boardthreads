#!/usr/bin/env node

var parser = require('email-addresses')
var getStdin = require('get-stdin')

getStdin().then(function (from) {
  var addr = parser.parseOneAddress(from)
  process.stdout.write(addr)
})
