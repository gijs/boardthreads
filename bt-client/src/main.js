import Cycle from '@cycle/rx-run'
import storageDriver from '@cycle/storage'
import {makeDOMDriver} from '@cycle/dom'
import {makeHTTPDriver} from '@cycle/http'
import {makeTrelloDriver} from './trello-driver'
import {makeHumaneDriver} from './humanejs-driver'

const redirectDriver = url$ => url$.filter(u => u).subscribe(u => window.location.href = u)

var app = require('./app').default

const drivers = {
  STORAGE: storageDriver,
  HTTP: makeHTTPDriver(),
  TRELLO: makeTrelloDriver('ac61d8974aa86dd25f9597fa651a2ed8', 'BoardThreads'),
  MAIN: makeDOMDriver('body > main'),
  NAV: makeDOMDriver('body > nav'),
  NOTIFICATION: makeHumaneDriver({timeout: 9000}),
  REDIRECT: redirectDriver
}

Cycle.run(app, drivers)
